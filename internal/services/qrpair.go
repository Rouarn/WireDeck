package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"

	"woyuxiaji.cn/wiredeck/internal/model"
)

const (
	pairingServiceType = "_adb-tls-pairing._tcp"
	connectServiceType = "_adb-tls-connect._tcp"
)

// qrLog 记录扫码配对的诊断信息，便于定位问题。
func qrLog(format string, args ...any) {
	f, err := os.OpenFile("/tmp/wiredeck-qr.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}

// CreatePairingQR 生成一次扫码配对会话。
//
// 二维码内容与 Android Studio 一致：
//
//	WIFI:T:ADB;S:<name>;P:<password>;;
//
// 手机扫描后会广播 _adb-tls-pairing._tcp 服务，电脑用 DNS-SD 发现后执行 adb pair。
func (s *DeviceService) CreatePairingQR() (model.QRPairingSession, error) {
	name := "adb-" + randomString(6)
	password := randomDigits(6)

	return model.QRPairingSession{
		Name:     name,
		Password: password,
		Payload:  fmt.Sprintf("WIFI:T:ADB;S:%s;P:%s;;", name, password),
	}, nil
}

// WaitPairing 等待手机扫描二维码后广播的配对服务，并完成配对与连接。
//
// 注意：这里不使用 `adb mdns services`，因为它只列出 `_adb-tls-connect._tcp`，
// 看不到配对服务；必须用系统 DNS-SD 自己浏览。
func (s *DeviceService) WaitPairing(ctx context.Context, name, password string, timeout time.Duration) (model.PairResult, error) {
	entry, err := browseService(ctx, pairingServiceType, name, timeout)
	if err != nil {
		return model.PairResult{
			OK:      false,
			Message: "等待超时，请确认手机已扫描二维码，并且与电脑在同一局域网",
		}, nil
	}

	host := firstIPv4(entry)
	if host == "" {
		return model.PairResult{OK: false, Message: "未能解析手机地址，请改用配对码方式"}, nil
	}

	qrLog("发现配对服务 name=%s host=%s port=%d code=%s", entry.Instance, host, entry.Port, password)

	pairReq := model.PairRequest{Host: host, PairPort: entry.Port, Code: password}
	res, pairErr := s.Pair(ctx, pairReq)
	qrLog("pair 结果 ok=%v msg=%q raw=%q err=%v", res.OK, res.Message, res.Raw, pairErr)

	// adb server 偶尔会处于半死状态，表现为 protocol fault；重启后重试一次。
	if pairErr == nil && !res.OK && strings.Contains(res.Message, "protocol fault") {
		qrLog("pair 协议错误，重启 adb server 后重试")
		if restartErr := s.server.Restart(ctx); restartErr == nil {
			res, pairErr = s.Pair(ctx, pairReq)
			qrLog("重试 pair 结果 ok=%v msg=%q raw=%q err=%v", res.OK, res.Message, res.Raw, pairErr)
		} else {
			qrLog("重启 adb server 失败: %v", restartErr)
		}
	}

	if pairErr != nil {
		return res, pairErr
	}
	if !res.OK {
		return res, nil
	}

	connected := s.autoConnect(ctx, host, name)
	qrLog("autoConnect 结果 connected=%v", connected)
	message := "配对成功，请在手动配对里填写连接端口完成连接"
	if connected {
		message = "配对并连接成功"
	}

	return model.PairResult{OK: true, Message: message, Raw: res.Raw}, nil
}

// autoConnect 在配对成功后等待连接服务出现并连接。
//
// 连接服务用 adb 自己的 mdns 列表（它能列出 _adb-tls-connect._tcp）。
func (s *DeviceService) autoConnect(ctx context.Context, host, name string) bool {
	deadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Second):
		}

		services, err := s.Discover(ctx)
		if err != nil {
			continue
		}

		for _, svc := range services {
			if svc.Kind != "connect" {
				continue
			}
			if svc.Host != host && !serviceMatches(svc.Name, name) {
				continue
			}

			qrLog("发现连接服务 name=%s host=%s port=%d", svc.Name, svc.Host, svc.Port)
			res, connErr := s.Connect(ctx, model.ConnectRequest{Host: svc.Host, Port: svc.Port})
			if connErr == nil && res.OK {
				// 先刷新设备列表，记录历史时才能拿到真实 serial 与型号。
				s.RefreshAndEmit(ctx)
				s.RecordConnection(svc.Host, svc.Port, name)
				return true
			}
			qrLog("connect 失败 ok=%v msg=%q err=%v", res.OK, res.Message, connErr)
		}
	}

	return false
}

// browseService 浏览一个 DNS-SD 服务，直到找到实例名匹配的条目或超时。
func browseService(ctx context.Context, serviceType, instance string, timeout time.Duration) (*zeroconf.ServiceEntry, error) {
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}

	entries := make(chan *zeroconf.ServiceEntry, 8)
	go func() {
		_ = resolver.Browse(browseCtx, serviceType, "local.", entries)
	}()

	for {
		select {
		case entry := <-entries:
			if entry == nil {
				continue
			}
			if serviceMatches(entry.Instance, instance) {
				return entry, nil
			}
		case <-browseCtx.Done():
			return nil, fmt.Errorf("等待服务超时")
		}
	}
}

func firstIPv4(entry *zeroconf.ServiceEntry) string {
	if len(entry.AddrIPv4) > 0 {
		return entry.AddrIPv4[0].String()
	}
	if len(entry.AddrIPv6) > 0 {
		return entry.AddrIPv6[0].String()
	}
	return ""
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf)
}

func randomDigits(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = '0' + buf[i]%10
	}
	return string(buf)
}
