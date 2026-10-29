package adb

import (
	"net"
	"strconv"
	"strings"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// ParseMDNSServices 解析 `adb mdns services` 的输出。
//
// 真实输出（adb 37）：
//
//	List of discovered mdns services
//	adb-XXXXXX	_adb-tls-pairing._tcp	192.168.1.23:37123
//	adb-YYYYYY	_adb-tls-connect._tcp	192.168.1.23:40123
//
// 字段之间可能是 tab 或空格，统一用 Fields 处理；地址可能是 IPv6，交给 SplitHostPort。
func ParseMDNSServices(output string) []model.MDNSService {
	services := make([]model.MDNSService, 0)

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "List of discovered") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		host, port, err := splitHostPort(fields[2])
		if err != nil {
			continue
		}

		services = append(services, model.MDNSService{
			Name:    fields[0],
			Kind:    classifyMDNSKind(fields[1]),
			Address: fields[2],
			Host:    host,
			Port:    port,
		})
	}

	return services
}

func classifyMDNSKind(kind string) string {
	switch {
	case strings.Contains(kind, "_adb-tls-pairing._tcp"):
		return "pairing"
	case strings.Contains(kind, "_adb-tls-connect._tcp"):
		return "connect"
	default:
		return "other"
	}
}

func splitHostPort(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
