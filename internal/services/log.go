package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
)

const (
	logBatchInterval = 100 * time.Millisecond
	logBatchSize     = 200
)

// LogService 负责 logcat 实时流的启动与停止。
type LogService struct {
	server *adb.Server
	emit   func(event string, data any)

	mu      sync.Mutex
	streams map[string]context.CancelFunc
	seq     int
}

// NewLogService 创建日志服务。
func NewLogService(server *adb.Server, emit func(string, any)) *LogService {
	return &LogService{
		server:  server,
		emit:    emit,
		streams: make(map[string]context.CancelFunc),
	}
}

// Start 启动一个 logcat 流。pkg 非空时只采集该应用的日志。
func (s *LogService) Start(ctx context.Context, serial, pkg string) (string, error) {
	if serial == "" {
		return "", fmt.Errorf("未选择设备")
	}
	if s.server.BinaryPath() == "" {
		return "", adb.ErrAdbNotFound
	}

	filterArgs := make([]string, 0, 1)
	if pkg != "" {
		pid := s.pidOf(ctx, serial, pkg)
		if pid == "" {
			return "", fmt.Errorf("应用未运行，请先在手机上打开 %s 再开始", pkg)
		}
		filterArgs = append(filterArgs, "--pid="+pid)
	}

	runCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.seq++
	id := fmt.Sprintf("log-%d", s.seq)
	s.streams[id] = cancel
	s.mu.Unlock()

	go s.stream(runCtx, id, serial, filterArgs)
	return id, nil
}

// pidOf 返回应用当前进程号，未运行时返回空。
func (s *LogService) pidOf(ctx context.Context, serial, pkg string) string {
	res, err := s.server.Run(ctx, 5*time.Second, "-s", serial, "shell", "pidof", pkg)
	if err != nil || res.ExitCode != 0 {
		return ""
	}
	fields := strings.Fields(res.Stdout)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Stop 停止指定日志流。
func (s *LogService) Stop(streamID string) {
	s.mu.Lock()
	cancel := s.streams[streamID]
	delete(s.streams, streamID)
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// StopAll 停止全部日志流（应用退出时调用）。
func (s *LogService) StopAll() {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.streams))
	for _, cancel := range s.streams {
		cancels = append(cancels, cancel)
	}
	s.streams = make(map[string]context.CancelFunc)
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (s *LogService) stream(ctx context.Context, id, serial string, filterArgs []string) {
	defer func() {
		s.mu.Lock()
		delete(s.streams, id)
		s.mu.Unlock()
	}()

	args := []string{"-P", strconv.Itoa(s.server.Port()), "-s", serial, "logcat", "-v", "threadtime"}
	args = append(args, filterArgs...)

	cmd := exec.CommandContext(ctx, s.server.BinaryPath(), args...)
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	defer func() { _ = cmd.Wait() }()

	lineCh := make(chan string, 2048)
	go func() {
		scanner := bufio.NewScanner(stdout)
		// logcat 行可能很长，默认 64KB 会截断。
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case lineCh <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
		close(lineCh)
	}()

	ticker := time.NewTicker(logBatchInterval)
	defer ticker.Stop()

	batch := make([]string, 0, logBatchSize)
	flush := func() {
		if len(batch) == 0 || s.emit == nil {
			return
		}
		lines := append([]string(nil), batch...)
		s.emit("logcat:batch", model.LogcatBatch{StreamID: id, Lines: lines})
		batch = batch[:0]
	}

	for {
		select {
		case line, ok := <-lineCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, line)
			if len(batch) >= logBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}
