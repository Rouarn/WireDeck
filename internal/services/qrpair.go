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

			message := "配对成功，请在手动配对里填写连接端口完成连接"
			if _, connectSvc := s.findPairingServices(ctx, name); connectSvc != nil {
				if conn, connErr := s.Connect(ctx, model.ConnectRequest{
					Host: connectSvc.Host,
					Port: connectSvc.Port,
				}); connErr == nil && conn.OK {
					s.RecordConnection(connectSvc.Host, connectSvc.Port, name)
					message = "配对并连接成功"
				}
			}

			return model.PairResult{OK: true, Message: message, Raw: res.Raw}, nil
		}
	}

	return model.PairResult{
		OK:      false,
		Message: "等待超时，请确认手机已扫描二维码，并且与电脑在同一局域网",
	}, nil
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
