package adb

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// DefaultPort 使用独立端口，避免与 Android Studio 等工具的 5037 冲突。
const DefaultPort = 5038

// Server 负责 adb 可执行文件定位、独立 server 启动与版本探测。
type Server struct {
	mu         sync.Mutex
	binaryPath string
	version    string
	port       int
	runner     *Runner
	logf       func(format string, args ...any)
}

// NewServer 创建一个指定端口的 adb server 管理器。
func NewServer(port int) *Server {
	if port <= 0 {
		port = DefaultPort
	}
	return &Server{port: port}
}

// SetLogger 设置 adb 调用日志回调。
func (s *Server) SetLogger(fn func(format string, args ...any)) {
	s.logf = fn
}

// Port 返回当前使用的 adb server 端口。
func (s *Server) Port() int { return s.port }

// BinaryPath 返回已解析的 adb 可执行文件路径。
func (s *Server) BinaryPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.binaryPath
}

// Resolve 定位 adb 可执行文件。explicit 为空时按 PATH 与常见安装路径查找。
func (s *Server) Resolve(explicit string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveLocked(explicit)
}

func (s *Server) resolveLocked(explicit string) error {
	path, err := FindBinary(explicit)
	if err != nil {
		return err
	}
	s.binaryPath = path
	s.runner = NewRunner(path, s.port)
	s.runner.Logf = s.logf
	if s.logf != nil {
		s.logf("adb binary resolved: %s", path)
	}
	return nil
}

// FindBinary 按优先级查找 adb：用户显式指定 → PATH → 常见安装目录。
func FindBinary(explicit string) (string, error) {
	if explicit != "" {
		info, err := os.Stat(explicit)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("%w: %s", ErrAdbNotFound, explicit)
		}
		return explicit, nil
	}

	if p, err := exec.LookPath("adb"); err == nil {
		return p, nil
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		"/opt/homebrew/bin/adb",
		"/usr/local/bin/adb",
		filepath.Join(home, "Library/Android/sdk/platform-tools/adb"),
		filepath.Join(home, "Android/Sdk/platform-tools/adb"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}

	return "", ErrAdbNotFound
}

// Restart 重启 adb server，用于清除异常状态（例如配对时的 protocol fault）。
func (s *Server) Restart(ctx context.Context) error {
	_, _ = s.Run(ctx, 5*time.Second, "kill-server")
	time.Sleep(500 * time.Millisecond)
	_, err := s.EnsureReady(ctx)
	return err
}

// EnsureReady 定位 adb、启动独立 server 并探测版本，返回环境信息（幂等）。
func (s *Server) EnsureReady(ctx context.Context) (*model.AdbEnv, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.binaryPath == "" {
		if err := s.resolveLocked(""); err != nil {
			return nil, err
		}
	}

	res, err := s.runner.Run(ctx, 15*time.Second, "start-server")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("%w: %s", ErrAdbServer, strings.TrimSpace(res.Output()))
	}

	version, err := s.versionLocked(ctx)
	if err != nil {
		return nil, err
	}
	s.version = version

	return &model.AdbEnv{
		BinaryPath: s.binaryPath,
		Version:    version,
		ServerPort: s.port,
		Ready:      true,
	}, nil
}

func (s *Server) versionLocked(ctx context.Context) (string, error) {
	res, err := s.runner.Run(ctx, 10*time.Second, "version")
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("%w: %s", ErrAdbServer, strings.TrimSpace(res.Output()))
	}

	// 输出示例：
	// Android Debug Bridge version 1.0.41
	// Version 35.0.2-12147458
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version ") {
			return strings.TrimPrefix(line, "Version "), nil
		}
	}
	return strings.TrimSpace(res.Stdout), nil
}

// Run 以已解析的 adb 执行一条命令。
func (s *Server) Run(ctx context.Context, timeout time.Duration, args ...string) (*Result, error) {
	s.mu.Lock()
	runner := s.runner
	s.mu.Unlock()
	if runner == nil {
		return nil, ErrAdbNotFound
	}
	return runner.Run(ctx, timeout, args...)
}

// RunRaw 执行命令并返回 stdout 原始字节（截图等二进制输出）。
func (s *Server) RunRaw(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	s.mu.Lock()
	runner := s.runner
	s.mu.Unlock()
	if runner == nil {
		return nil, ErrAdbNotFound
	}
	return runner.RunRaw(ctx, timeout, args...)
}

// PushWithProgress 以伪终端执行 adb push 并回调真实传输百分比。
func (s *Server) PushWithProgress(ctx context.Context, serial, local, remote string, onPercent func(int)) error {
	s.mu.Lock()
	runner := s.runner
	s.mu.Unlock()
	if runner == nil {
		return ErrAdbNotFound
	}
	return runner.PushWithProgress(ctx, serial, local, remote, onPercent)
}

// PullWithProgress 以伪终端执行 adb pull 并回调真实传输百分比。
func (s *Server) PullWithProgress(ctx context.Context, serial, remote, local string, onPercent func(int)) error {
	s.mu.Lock()
	runner := s.runner
	s.mu.Unlock()
	if runner == nil {
		return ErrAdbNotFound
	}
	return runner.PullWithProgress(ctx, serial, remote, local, onPercent)
}
