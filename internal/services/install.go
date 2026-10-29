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

	s.progress(id, "parse", 5, "解析安装包")
	if len(req.Paths) == 1 {
		// 解析失败不阻塞安装，仅影响前置信息展示。
		_, _ = apk.Inspect(req.Paths[0])
	}

	var result model.InstallResult
	if len(req.Paths) == 1 {
		result = s.installWithProgress(ctx, id, req, started)
	} else {
		result = s.installDirect(ctx, id, req, started)
	}

	s.emit("task:done", result)
}

// installWithProgress 先 push 到设备再 pm install，从而拿到真实传输进度。
func (s *InstallService) installWithProgress(ctx context.Context, id string, req model.InstallRequest, started time.Time) model.InstallResult {
	remote := fmt.Sprintf("/data/local/tmp/wiredeck-%d.apk", time.Now().UnixNano())
	defer func() {
		_, _ = s.server.Run(context.Background(), 10*time.Second, "shell", "rm", "-f", remote)
	}()

	s.progress(id, "transfer", 0, "传输安装包")
	err := s.server.PushWithProgress(ctx, req.Serial, req.Paths[0], remote, func(pct int) {
		// 传输阶段占 0-90%，给安装阶段留出空间。
		s.progress(id, "transfer", pct*9/10, fmt.Sprintf("传输 %d%%", pct))
	})
	if err != nil {
		if errors.Is(err, adb.ErrCanceled) {
			return s.canceledResult(id, started)
		}
		// 传输失败时回退到 adb install，避免目录权限导致完全不可用。
		s.progress(id, "install", -1, "回退为直接安装")
		return s.installDirect(ctx, id, req, started)
	}

	s.progress(id, "install", 95, "安装中")
	res, runErr := s.server.Run(ctx, installTimeout, s.pmInstallArgs(req, remote)...)
	return s.buildResult(id, res, runErr, started)
}

// installDirect 直接使用 adb install / install-multiple（多 split 或回退路径）。
func (s *InstallService) installDirect(ctx context.Context, id string, req model.InstallRequest, started time.Time) model.InstallResult {
	s.progress(id, "install", -1, "执行 adb install")
	res, err := s.server.Run(ctx, installTimeout, s.buildArgs(req)...)
	return s.buildResult(id, res, err, started)
}

func (s *InstallService) buildResult(id string, res *adb.Result, err error, started time.Time) model.InstallResult {
	result := model.InstallResult{TaskID: id}

	raw := ""
	if res != nil {
		raw = strings.TrimSpace(res.Output())
	}
	result.Raw = raw
	result.DurationMs = time.Since(started).Milliseconds()

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
		return result
	}

	if strings.Contains(raw, "Success") {
		result.OK = true
		result.Message = "安装成功"
		s.progress(id, "verify", 100, "安装完成")
		return result
	}

	code, advice := adb.ExplainInstallFailure(raw)
	result.Code = code
	result.Advice = advice
	result.Message = "安装失败"
	if code == "" {
		result.Message = "安装失败: " + firstLine(raw)
	}
	return result
}

func (s *InstallService) canceledResult(id string, started time.Time) model.InstallResult {
	return model.InstallResult{
		TaskID:     id,
		Code:       "CANCELED",
		Message:    "已取消安装",
		DurationMs: time.Since(started).Milliseconds(),
	}
}

func (s *InstallService) pmInstallArgs(req model.InstallRequest, remote string) []string {
	args := []string{"-s", req.Serial, "shell", "pm", "install"}
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
	return append(args, remote)
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
