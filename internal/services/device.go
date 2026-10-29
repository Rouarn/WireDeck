package services

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
)

const watchInterval = 2 * time.Second

// DeviceService 负责设备发现、配对、连接与状态轮询。
type DeviceService struct {
	server *adb.Server
	emit   func(event string, data any)

	mu      sync.RWMutex
	devices []model.Device
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewDeviceService 创建设备服务；emit 用于向前端推送事件。
func NewDeviceService(server *adb.Server, emit func(event string, data any)) *DeviceService {
	return &DeviceService{server: server, emit: emit}
}

// Start 启动后台轮询，设备状态变化时推送 device:changed。
func (s *DeviceService) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done

	go s.watch(runCtx, done)
}

// Stop 停止轮询并等待 goroutine 退出。
func (s *DeviceService) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *DeviceService) watch(ctx context.Context, done chan struct{}) {
	defer close(done)

	s.refreshAndEmit(ctx)
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshAndEmit(ctx)
		}
	}
}

// RefreshAndEmit 立即刷新设备并推送事件（用于配对/连接后即时反馈）。
func (s *DeviceService) RefreshAndEmit(ctx context.Context) {
	s.refreshAndEmit(ctx)
}

func (s *DeviceService) refreshAndEmit(ctx context.Context) {
	devices, err := s.Refresh(ctx)
	if err != nil {
		return
	}
	if !s.changed(devices) {
		return
	}

	s.mu.Lock()
	s.devices = devices
	s.mu.Unlock()

	println(fmt.Sprintf("[devices] changed: %d device(s)", len(devices)))
	s.emit("device:changed", devices)
}

func (s *DeviceService) changed(next []model.Device) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.devices) != len(next) {
		return true
	}

	prev := make(map[string]string, len(s.devices))
	for _, d := range s.devices {
		prev[d.Serial] = d.State + "|" + d.Model
	}
	for _, d := range next {
		if prev[d.Serial] != d.State+"|"+d.Model {
			return true
		}
	}
	return false
}

// Snapshot 返回最近一次已知的设备列表。
func (s *DeviceService) Snapshot() []model.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.Device(nil), s.devices...)
}

// Refresh 执行 `adb devices -l` 并解析。
func (s *DeviceService) Refresh(ctx context.Context) ([]model.Device, error) {
	res, err := s.server.Run(ctx, 8*time.Second, "devices", "-l")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("adb devices 执行失败: %s", strings.TrimSpace(res.Output()))
	}
	return adb.ParseDevices(res.Stdout, time.Now().UnixMilli()), nil
}

// Discover 执行 `adb mdns services` 并解析配对/连接服务。
func (s *DeviceService) Discover(ctx context.Context) ([]model.MDNSService, error) {
	res, err := s.server.Run(ctx, 6*time.Second, "mdns", "services")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("adb mdns services 执行失败: %s", strings.TrimSpace(res.Output()))
	}
	return adb.ParseMDNSServices(res.Stdout), nil
}

// Pair 执行无线调试配对（配对端口与连接端口不同）。
func (s *DeviceService) Pair(ctx context.Context, req model.PairRequest) (model.PairResult, error) {
	if req.Host == "" || req.PairPort <= 0 || req.Code == "" {
		return model.PairResult{}, fmt.Errorf("IP、配对端口和配对码不能为空")
	}

	addr := net.JoinHostPort(req.Host, strconv.Itoa(req.PairPort))
	res, err := s.server.Run(ctx, 25*time.Second, "pair", addr, req.Code)
	if err != nil {
		return model.PairResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	return model.PairResult{
		OK:      strings.Contains(raw, "Successfully paired"),
		Message: humanizePairMessage(raw),
		Raw:     raw,
	}, nil
}

func humanizePairMessage(raw string) string {
	switch {
	case strings.Contains(raw, "Successfully paired"):
		return "配对成功"
	case strings.Contains(raw, "wrong password"):
		return "配对码错误或已过期，请在手机上重新生成后再试"
	case strings.Contains(raw, "failed to connect"), strings.Contains(raw, "No route to host"):
		return "无法连接到手机，请确认 IP、配对端口正确，且手机与电脑在同一局域网"
	case strings.Contains(raw, "unknown host"):
		return "无法解析该 IP，请检查填写是否正确"
	default:
		return raw
	}
}

// Connect 执行 `adb connect`（使用无线调试主页面的连接端口）。
func (s *DeviceService) Connect(ctx context.Context, req model.ConnectRequest) (model.ConnectResult, error) {
	if req.Host == "" || req.Port <= 0 {
		return model.ConnectResult{}, fmt.Errorf("IP 与连接端口不能为空")
	}

	addr := net.JoinHostPort(req.Host, strconv.Itoa(req.Port))
	res, err := s.server.Run(ctx, 15*time.Second, "connect", addr)
	if err != nil {
		return model.ConnectResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	return model.ConnectResult{
		OK:      strings.Contains(raw, "connected to"),
		Address: addr,
		Message: raw,
	}, nil
}

// Disconnect 断开设备。mDNS 服务名形式的 serial 无法反查地址，此时断开全部无线连接。
func (s *DeviceService) Disconnect(ctx context.Context, serial string) error {
	args := []string{"disconnect"}
	if strings.Contains(serial, ":") {
		args = append(args, serial)
	}
	_, err := s.server.Run(ctx, 10*time.Second, args...)
	return err
}
