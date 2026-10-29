package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
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

	res, pairErr := s.Pair(ctx, model.PairRequest{
		Host:     host,
		PairPort: entry.Port,
		Code:     password,
	})
	qrLog("pair 结果 ok=%v msg=%q raw=%q err=%v", res.OK, res.Message, res.Raw, pairErr)
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

// autoConnect 在配对成功后浏览连接服务，按实例名或 IP 匹配后自动连接。
func (s *DeviceService) autoConnect(ctx context.Context, host, name string) bool {
	browseCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return false
	}

	entries := make(chan *zeroconf.ServiceEntry, 8)
	go func() {
		_ = resolver.Browse(browseCtx, connectServiceType, "local.", entries)
	}()

	for {
		select {
		case entry := <-entries:
			if entry == nil || len(entry.AddrIPv4) == 0 {
				continue
			}
			addr := entry.AddrIPv4[0].String()
			if !serviceMatches(entry.Instance, name) && addr != host {
				continue
			}
			qrLog("发现连接服务 name=%s host=%s port=%d", entry.Instance, addr, entry.Port)
			if res, err := s.Connect(ctx, model.ConnectRequest{Host: addr, Port: entry.Port}); err == nil && res.OK {
				s.RecordConnection(addr, entry.Port, name)
				return true
			} else {
				qrLog("connect 失败 ok=%v msg=%q err=%v", res.OK, res.Message, err)
			}
		case <-browseCtx.Done():
			return false
		}
	}
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
