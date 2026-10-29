package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// CreatePairingQR 生成一次扫码配对会话。
//
// 二维码内容与 Android Studio 一致：
//
//	WIFI:T:ADB;S:<name>;P:<password>;;
//
// 手机扫描后会广播 _adb-tls-pairing._tcp 服务，电脑轮询发现后执行 adb pair。
func (s *DeviceService) CreatePairingQR() (model.QRPairingSession, error) {
	name := "adb-" + randomString(6)
	password := randomDigits(6)

	return model.QRPairingSession{
		Name:     name,
		Password: password,
		Payload:  fmt.Sprintf("WIFI:T:ADB;S:%s;P:%s;;", name, password),
	}, nil
}

// WaitPairing 轮询 mDNS，等待手机扫描二维码后出现的配对服务，并完成配对与连接。
func (s *DeviceService) WaitPairing(ctx context.Context, name, password string, timeout time.Duration) (model.PairResult, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return model.PairResult{}, ctx.Err()
		case <-time.After(time.Second):
		}

		services, err := s.Discover(ctx)
		if err != nil {
			continue
		}

		for _, svc := range services {
			if svc.Kind != "pairing" || !serviceMatches(svc.Name, name) {
				continue
			}

			res, pairErr := s.Pair(ctx, model.PairRequest{
				Host:     svc.Host,
				PairPort: svc.Port,
				Code:     password,
			})
			if pairErr != nil {
				return res, pairErr
			}
			if !res.OK {
				return res, nil
			}

			connected := s.tryAutoConnect(ctx, svc.Host, name)
			message := "配对成功，请在手动配对里填写连接端口完成连接"
			if connected {
				message = "配对并连接成功"
			}

			return model.PairResult{OK: true, Message: message, Raw: res.Raw}, nil
		}
	}

	return model.PairResult{
		OK:      false,
		Message: "等待超时，请确认手机已扫描二维码，并且与电脑在同一局域网",
	}, nil
}

// tryAutoConnect 在配对成功后等待连接服务出现并建立连接。
func (s *DeviceService) tryAutoConnect(ctx context.Context, host, name string) bool {
	before := s.serialSet()

	for i := 0; i < 15; i++ {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(700 * time.Millisecond):
		}

		if svc := s.findConnectByHost(ctx, host); svc != nil {
			if res, err := s.Connect(ctx, model.ConnectRequest{Host: svc.Host, Port: svc.Port}); err == nil && res.OK {
				s.RecordConnection(svc.Host, svc.Port, name)
				return true
			}
		}

		// 新版 adb 有时会自行完成连接，这里同时观察设备列表。
		s.RefreshAndEmit(ctx)
		if s.hasNewDevice(before) {
			return true
		}
	}

	return false
}

func (s *DeviceService) serialSet() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set := make(map[string]bool, len(s.devices))
	for _, device := range s.devices {
		set[device.Serial] = true
	}
	return set
}

func (s *DeviceService) hasNewDevice(before map[string]bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.devices {
		if device.State == "device" && !before[device.Serial] {
			return true
		}
	}
	return false
}

func (s *DeviceService) findConnectByHost(ctx context.Context, host string) *model.MDNSService {
	services, err := s.Discover(ctx)
	if err != nil {
		return nil
	}

	for i := range services {
		if services[i].Kind == "connect" && services[i].Host == host {
			return &services[i]
		}
	}
	return nil
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
