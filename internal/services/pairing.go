package services

import (
	"context"
	"fmt"
	"strings"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// ParseQRCode 解析 Android 无线调试二维码。
//
// 二维码内容形如：
//
//	WIFI:T:ADB;S:adb-XXXXXX;P:123456;;
//
// S 为 mDNS 服务名，P 为 6 位配对码；部分版本还会带 H（主机地址）。
func ParseQRCode(content string) (model.QRCodeInfo, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "WIFI:") {
		return model.QRCodeInfo{}, fmt.Errorf("这不是无线调试配对二维码")
	}

	info := model.QRCodeInfo{}
	for _, part := range strings.Split(content, ";") {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(key)) {
		case "S":
			info.Service = value
		case "P":
			info.Password = value
		case "H":
			info.Host = value
		}
	}

	if strings.TrimSpace(info.Password) == "" {
		return info, fmt.Errorf("二维码中缺少配对码")
	}
	return info, nil
}

// PairByQRCode 根据二维码完成配对，并尽量自动连接。
func (s *DeviceService) PairByQRCode(ctx context.Context, content string) (model.PairResult, error) {
	info, err := ParseQRCode(content)
	if err != nil {
		return model.PairResult{}, err
	}

	if strings.TrimSpace(info.Service) == "" && info.Host == "" {
		return model.PairResult{}, fmt.Errorf("二维码中缺少服务信息")
	}

	pairSvc, connectSvc := s.findPairingServices(ctx, info.Service)

	if pairSvc == nil && info.Host == "" {
		return model.PairResult{
			OK:      false,
			Message: "未发现配对服务，请确认手机停留在「使用二维码配对设备」页面后重新扫码",
		}, nil
	}

	host, port := info.Host, info.Port
	if pairSvc != nil {
		host, port = pairSvc.Host, pairSvc.Port
	}
	if host == "" || port == 0 {
		return model.PairResult{OK: false, Message: "无法确定配对地址，请改用配对码方式"}, nil
	}

	pairResult, err := s.Pair(ctx, model.PairRequest{Host: host, PairPort: port, Code: info.Password})
	if err != nil {
		return pairResult, err
	}
	if !pairResult.OK {
		return pairResult, nil
	}

	// 配对成功后再次发现连接服务，尝试直接连接。
	if connectSvc == nil {
		_, connectSvc = s.findPairingServices(ctx, info.Service)
	}

	connected := false
	if connectSvc != nil {
		if res, err := s.Connect(ctx, model.ConnectRequest{Host: connectSvc.Host, Port: connectSvc.Port}); err == nil && res.OK {
			connected = true
			s.RecordConnection(connectSvc.Host, connectSvc.Port, info.Service)
		}
	}

	message := "配对成功，请在手动配对里填写连接端口完成连接"
	if connected {
		message = "配对并连接成功"
	}

	return model.PairResult{OK: true, Message: message, Raw: pairResult.Raw}, nil
}

func (s *DeviceService) findPairingServices(ctx context.Context, serviceName string) (*model.MDNSService, *model.MDNSService) {
	services, err := s.Discover(ctx)
	if err != nil {
		return nil, nil
	}

	var pairSvc, connectSvc *model.MDNSService
	for i := range services {
		svc := services[i]
		if serviceName != "" && !serviceMatches(svc.Name, serviceName) {
			continue
		}
		switch svc.Kind {
		case "pairing":
			if pairSvc == nil {
				pairSvc = &services[i]
			}
		case "connect":
			if connectSvc == nil {
				connectSvc = &services[i]
			}
		}
	}
	return pairSvc, connectSvc
}

func serviceMatches(name, target string) bool {
	name = strings.TrimSpace(name)
	target = strings.TrimSpace(target)
	if name == "" || target == "" {
		return true
	}
	return strings.Contains(name, target) || strings.Contains(target, name)
}
