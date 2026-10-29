package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/apk"
	"woyuxiaji.cn/wiredeck/internal/model"
)

const installTimeout = 10 * time.Minute

// InstallService 负责安装队列：同设备串行、跨设备并行。
type InstallService struct {
	server *adb.Server
	emit   func(event string, data any)

	mu          sync.Mutex
	deviceLocks map[string]*sync.Mutex
	cancels     map[string]context.CancelFunc
	seq         int
}

// NewInstallService 创建安装服务。
func NewInstallService(server *adb.Server, emit func(event string, data any)) *InstallService {
	return &InstallService{
		server:      server,
		emit:        emit,
		deviceLocks: make(map[string]*sync.Mutex),
		cancels:     make(map[string]context.CancelFunc),
	}
}

// Start 将安装任务加入队列并立即返回 taskId。
func (s *InstallService) Start(ctx context.Context, req model.InstallRequest) (string, error) {
	if req.Serial == "" {
		return "", fmt.Errorf("请先选择目标设备")
	}
	if len(req.Paths) == 0 {
		return "", fmt.Errorf("请选择至少一个 APK 文件")
	}
	for _, path := range req.Paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("无法读取文件: %s", path)
		}
	}

	s.mu.Lock()
	s.seq++
	id := fmt.Sprintf("task-%d-%d", time.Now().UnixMilli(), s.seq)
	s.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancels[id] = cancel
	s.mu.Unlock()

	go s.run(runCtx, id, req)
	return id, nil
}

// Cancel 取消一个进行中的安装任务。
func (s *InstallService) Cancel(taskID string) {
	s.mu.Lock()
	cancel := s.cancels[taskID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *InstallService) lockFor(serial string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lock, ok := s.deviceLocks[serial]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	s.deviceLocks[serial] = lock
	return lock
}

func (s *InstallService) run(ctx context.Context, id string, req model.InstallRequest) {
	defer func() {
		s.mu.Lock()
		delete(s.cancels, id)
		s.mu.Unlock()
	}()

	started := time.Now()

	// 同一设备的安装串行执行，避免 adb 客户端竞争。
	lock := s.lockFor(req.Serial)
	lock.Lock()
	defer lock.Unlock()

	s.progress(id, "parse", 10, "解析安装包")
	if len(req.Paths) == 1 {
		// 解析失败不阻塞安装，仅影响前置信息展示。
		_, _ = apk.Inspect(req.Paths[0])
	}

	s.progress(id, "transfer", -1, "传输并安装中")
	res, err := s.server.Run(ctx, installTimeout, s.buildArgs(req)...)

	raw := ""
	if res != nil {
		raw = strings.TrimSpace(res.Output())
	}

	result := model.InstallResult{
		TaskID:     id,
		Raw:        raw,
		DurationMs: time.Since(started).Milliseconds(),
	}

	if err != nil {
		switch {
		case errors.Is(err, adb.ErrCanceled):
			result.Code = "CANCELED"
			result.Message = "已取消安装"
		case errors.Is(err, adb.ErrTimeout):
			result.Code = "TIMEOUT"
			result.Message = "安装超时（超过 10 分钟）"
			result.Advice = "请检查设备连接与存储空间后重试"
		default:
			result.Message = err.Error()
		}
		s.emit("task:done", result)
		return
	}

	if strings.Contains(raw, "Success") {
		result.OK = true
		result.Message = "安装成功"
		s.progress(id, "verify", 100, "安装完成")
	} else {
		code, advice := adb.ExplainInstallFailure(raw)
		result.Code = code
		result.Advice = advice
		result.Message = "安装失败"
		if code == "" {
			result.Message = "安装失败: " + firstLine(raw)
		}
	}

	s.emit("task:done", result)
}

func (s *InstallService) buildArgs(req model.InstallRequest) []string {
	args := []string{"-s", req.Serial}

	if len(req.Paths) > 1 {
		args = append(args, "install-multiple")
	} else {
		args = append(args, "install")
	}

	if req.Options.Replace {
		args = append(args, "-r")
	}
	if req.Options.AllowDowngrade {
		args = append(args, "-d")
	}
	if req.Options.GrantPermissions {
		args = append(args, "-g")
	}
	if req.Options.AllowTest {
		args = append(args, "-t")
	}
	if req.Options.UserID != nil {
		args = append(args, "--user", strconv.Itoa(*req.Options.UserID))
	}

	return append(args, req.Paths...)
}

func (s *InstallService) progress(id, stage string, percent int, message string) {
	s.emit("task:progress", model.TaskProgress{
		TaskID:  id,
		Stage:   stage,
		Percent: percent,
		Message: message,
	})
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}
