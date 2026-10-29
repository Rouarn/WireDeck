package adb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/creack/pty"
)

// DefaultTimeout 是单条 adb 命令的默认超时。
const DefaultTimeout = 15 * time.Second

// Result 保存一次 adb 调用的原始结果，便于诊断与日志留痕。
type Result struct {
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Output 优先返回 stdout；为空时回退 stderr。
func (r *Result) Output() string {
	if strings.TrimSpace(r.Stdout) != "" {
		return r.Stdout
	}
	return r.Stderr
}

// Runner 统一封装 adb 子进程调用：参数数组化、超时、取消、双路输出。
type Runner struct {
	AdbPath string
	Port    int
	Logf    func(format string, args ...any)
}

// NewRunner 创建一个绑定到指定 adb 可执行文件与端口的执行器。
func NewRunner(adbPath string, port int) *Runner {
	return &Runner{AdbPath: adbPath, Port: port}
}

// Run 执行一条 adb 命令。退出码非 0 不作为 error 返回，交由调用方按文本判定。
func (r *Runner) Run(ctx context.Context, timeout time.Duration, args ...string) (*Result, error) {
	if r.AdbPath == "" {
		return nil, ErrAdbNotFound
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	full := r.commandArgs(args...)

	cmd := exec.CommandContext(runCtx, r.AdbPath, full...)
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()
	runErr := cmd.Run()

	res := &Result{
		Args:     full,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Duration: time.Since(started),
	}

	if runCtx.Err() == context.DeadlineExceeded {
		res.ExitCode = -1
		r.log("exec adb %s -> timeout after %s", strings.Join(full, " "), res.Duration.Round(time.Millisecond))
		return res, fmt.Errorf("%w: adb %s", ErrTimeout, strings.Join(args, " "))
	}

	if runCtx.Err() == context.Canceled {
		res.ExitCode = -1
		r.log("exec adb %s -> canceled", strings.Join(full, " "))
		return res, fmt.Errorf("%w: adb %s", ErrCanceled, strings.Join(args, " "))
	}

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			r.log("exec adb %s -> exit=%d (%s)", strings.Join(full, " "), res.ExitCode, res.Duration.Round(time.Millisecond))
			return res, nil
		}
		r.log("exec adb %s -> %v", strings.Join(full, " "), runErr)
		return res, runErr
	}

	r.log("exec adb %s -> exit=0 (%s)", strings.Join(full, " "), res.Duration.Round(time.Millisecond))
	return res, nil
}

func (r *Runner) log(format string, args ...any) {
	if r.Logf != nil {
		r.Logf(format, args...)
	}
}

func (r *Runner) commandArgs(args ...string) []string {
	full := make([]string, 0, len(args)+2)
	if r.Port > 0 {
		full = append(full, "-P", strconv.Itoa(r.Port))
	}
	return append(full, args...)
}

var pushPercentPattern = regexp.MustCompile(`(\d+)%`)

// PushWithProgress 通过伪终端执行 adb push，实时回调传输百分比。
//
// adb 在非 TTY 环境下只输出最终结果，因此必须借助 pty 才能拿到进度行。
func (r *Runner) PushWithProgress(ctx context.Context, serial, local, remote string, onPercent func(int)) error {
	if r.AdbPath == "" {
		return ErrAdbNotFound
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(runCtx, r.AdbPath, r.commandArgs("-s", serial, "push", local, remote)...)
	// adb 只有在 TERM 存在且 stdout 为终端时才输出百分比进度。
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C", "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 120})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPushFailed, err)
	}
	defer func() { _ = ptmx.Close() }()

	var (
		buf   bytes.Buffer
		last  = -1
		chunk = make([]byte, 4096)
	)

	for {
		n, readErr := ptmx.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])

			if line := lastOutputLine(buf.String()); line != "" {
				if match := pushPercentPattern.FindStringSubmatch(line); len(match) == 2 {
					if pct, err := strconv.Atoi(match[1]); err == nil && pct != last {
						last = pct
						if onPercent != nil {
							onPercent(pct)
						}
					}
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		if runCtx.Err() == context.Canceled {
			return fmt.Errorf("%w: %v", ErrCanceled, err)
		}
		return fmt.Errorf("%w: %v: %s", ErrPushFailed, err, lastOutputLine(buf.String()))
	}
	return nil
}

// lastOutputLine 返回输出中最后一段（adb push 用 \r 原地刷新进度）。
func lastOutputLine(text string) string {
	lines := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\r' || r == '\n'
	})
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
