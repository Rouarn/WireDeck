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
	history []model.DeviceHistory
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewDeviceService 创建设备服务；emit 用于向前端推送事件。
func NewDeviceService(server *adb.Server, emit func(event string, data any)) *DeviceService {
	service := &DeviceService{server: server, emit: emit}
	service.loadHistory()
	return service
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

// Refresh 执行 `adb devices -l` 并解析，随后合并同一设备的重复连接。
func (s *DeviceService) Refresh(ctx context.Context) ([]model.Device, error) {
	if _, err := s.server.EnsureReady(ctx); err != nil {
		return nil, err
	}

	res, err := s.server.Run(ctx, 8*time.Second, "devices", "-l")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("adb devices 执行失败: %s", strings.TrimSpace(res.Output()))
	}

	devices := adb.ParseDevices(res.Stdout, time.Now().UnixMilli())
	return s.enrich(ctx, s.mergeDuplicates(ctx, devices)), nil
}

// enrich 为在线设备补充 WLAN IP 与 Wi-Fi 信号强度。
func (s *DeviceService) enrich(ctx context.Context, devices []model.Device) []model.Device {
	for i := range devices {
		if devices[i].State != "device" {
			continue
		}
		devices[i].IP = s.deviceIP(ctx, devices[i].Serial)
		if devices[i].Transport == "wifi" {
			devices[i].RSSI = s.wifiRSSI(ctx, devices[i].Serial)
		}
	}
	return devices
}

func (s *DeviceService) deviceIP(ctx context.Context, serial string) string {
	// Wi-Fi 设备的 serial 形如 192.168.1.23:41257，可直接取出 IP。
	if host, _, ok := strings.Cut(serial, ":"); ok && strings.Count(host, ".") == 3 {
		return host
	}

	res, err := s.server.Run(ctx, 5*time.Second, "-s", serial, "shell", "ip", "route")
	if err != nil || res.ExitCode != 0 {
		return ""
	}

	for _, line := range strings.Split(res.Stdout, "\n") {
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "src" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}
	return ""
}

func (s *DeviceService) wifiRSSI(ctx context.Context, serial string) int {
	res, err := s.server.Run(ctx, 5*time.Second, "-s", serial, "shell", "cmd", "wifi", "status")
	if err != nil || res.ExitCode != 0 {
		return 0
	}

	for _, line := range strings.Split(res.Stdout, "\n") {
		idx := strings.Index(line, "RSSI:")
		if idx < 0 {
			continue
		}
		value := strings.TrimSpace(line[idx+len("RSSI:"):])
		value = strings.TrimSpace(strings.TrimSuffix(value, "dBm"))
		if fields := strings.Fields(value); len(fields) > 0 {
			value = fields[0]
		}
		value = strings.Trim(value, ",;")
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return 0
}

// mergeDuplicates 合并同一物理设备的多条连接记录。
//
// adb 会把「手动 connect 的 IP:port」与「mDNS 自动连接的服务名」视为两台设备，
// 它们指向同一台手机。这里仅在存在同型号设备时才读取 ro.serialno，避免每轮轮询都付出额外调用。
func (s *DeviceService) mergeDuplicates(ctx context.Context, devices []model.Device) []model.Device {
	if len(devices) < 2 {
		return devices
	}

	groups := make(map[string][]int)
	for i, d := range devices {
		if d.Model == "" {
			continue
		}
		groups[d.Model] = append(groups[d.Model], i)
	}

	drop := make(map[int]bool)
	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}

		bySerial := make(map[string]int)
		for _, i := range indexes {
			hw := s.hardwareSerial(ctx, devices[i].Serial)
			if hw == "" {
				continue
			}
			devices[i].HwSerial = hw

			prev, ok := bySerial[hw]
			if !ok {
				bySerial[hw] = i
				continue
			}

			keep, remove := preferConnection(prev, i, devices)
			bySerial[hw] = keep
			drop[remove] = true
		}
	}

	if len(drop) == 0 {
		return devices
	}

	merged := make([]model.Device, 0, len(devices)-len(drop))
	for i, d := range devices {
		if !drop[i] {
			merged = append(merged, d)
		}
	}
	return merged
}

func (s *DeviceService) hardwareSerial(ctx context.Context, serial string) string {
	res, err := s.server.Run(ctx, 5*time.Second, "-s", serial, "shell", "getprop", "ro.serialno")
	if err != nil || res.ExitCode != 0 {
		return ""
	}
	return strings.TrimSpace(res.Stdout)
}

// preferConnection 决定保留哪一条连接：mDNS 优先，其次 Wi-Fi，最后 USB。
func preferConnection(a, b int, devices []model.Device) (int, int) {
	da, db := devices[a], devices[b]

	if isMDNSSerial(da.Serial) != isMDNSSerial(db.Serial) {
		if isMDNSSerial(da.Serial) {
			return a, b
		}
		return b, a
	}

	if (da.Transport == "wifi") != (db.Transport == "wifi") {
		if da.Transport == "wifi" {
			return a, b
		}
		return b, a
	}

	return a, b
}

func isMDNSSerial(serial string) bool {
	return strings.Contains(serial, "_adb-tls-connect._tcp")
}

// EnableWireless 将 USB 连接的设备切换为无线调试（tcpip 5555）并连接。
func (s *DeviceService) EnableWireless(ctx context.Context, serial string) (model.ConnectResult, error) {
	ip := s.wlanIP(ctx, serial)
	if ip == "" || ip == "0.0.0.0" {
		return model.ConnectResult{
			OK:      false,
			Message: "无法获取手机 Wi-Fi IP，请确认手机已连接 Wi-Fi 且与电脑在同一网络",
		}, nil
	}

	res, err := s.server.Run(ctx, 20*time.Second, "-s", serial, "tcpip", "5555")
	if err != nil {
		return model.ConnectResult{}, err
	}
	if res.ExitCode != 0 {
		return model.ConnectResult{OK: false, Message: strings.TrimSpace(res.Output())}, nil
	}

	// adbd 重启为 TCP 模式需要几秒，期间连接会失败，这里轮询重试。
	var lastMessage string
	for attempt := 0; attempt < 6; attempt += 1 {
		select {
		case <-ctx.Done():
			return model.ConnectResult{}, ctx.Err()
		case <-time.After(1500 * time.Millisecond):
		}

		result, connErr := s.Connect(ctx, model.ConnectRequest{Host: ip, Port: 5555})
		if connErr != nil {
			lastMessage = connErr.Error()
			continue
		}
		if result.OK {
			s.RecordConnection(ip, 5555, "")
			return result, nil
		}
		lastMessage = result.Message
	}

	return model.ConnectResult{
		OK: false,
		Message: fmt.Sprintf(
			"已切换为 TCP 模式，但连接 %s:5555 失败（%s）。请确认手机 Wi-Fi 与电脑在同一网络，然后重试",
			ip, lastMessage,
		),
	}, nil
}

// wlanIP 读取设备的 wlan0 IPv4 地址。
func (s *DeviceService) wlanIP(ctx context.Context, serial string) string {
	res, err := s.server.Run(ctx, 6*time.Second, "-s", serial, "shell", "ip", "-f", "inet", "addr", "show", "wlan0")
	if err != nil || res.ExitCode != 0 {
		return ""
	}

	for _, line := range strings.Split(res.Stdout, "\n") {
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "inet" && i+1 < len(fields) {
				return strings.Split(fields[i+1], "/")[0]
			}
		}
	}
	return ""
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
