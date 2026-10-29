package adb

import (
	"strings"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// ParseDevices 解析 `adb devices -l` 的输出。
//
// 示例：
//
//	List of devices attached
//	192.168.1.23:37123     device product:panther model:Pixel_7 device:panther transport_id:1
//	R58M1234567            unauthorized usb:338690048X transport_id:2
func ParseDevices(output string, now int64) []model.Device {
	devices := make([]model.Device, 0)

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" ||
			strings.HasPrefix(line, "List of devices") ||
			strings.HasPrefix(line, "* daemon") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		dev := model.Device{
			Serial:    fields[0],
			State:     fields[1],
			Transport: "usb",
			LastSeen:  now,
		}

		for _, kv := range fields[2:] {
			key, value, ok := strings.Cut(kv, ":")
			if !ok {
				continue
			}
			if key == "model" {
				dev.Model = value
			}
		}

		if isWirelessSerial(dev.Serial) {
			dev.Transport = "wifi"
		}

		devices = append(devices, dev)
	}

	return devices
}

// isWirelessSerial 判断 serial 是否为无线调试地址。
//
// 两种形态：
//   - Android 10 及以下：host:port（tcpip 模式）
//   - Android 11+：adb 以 mDNS 服务名作为 serial，形如
//     adb-d2de5f92-1guyQc._adb-tls-connect._tcp
func isWirelessSerial(serial string) bool {
	if strings.HasPrefix(serial, "emulator-") {
		return false
	}
	if strings.Contains(serial, "_adb-tls-connect._tcp") {
		return true
	}

	host, port, ok := strings.Cut(serial, ":")
	if !ok || port == "" {
		return false
	}
	return strings.Contains(host, ".")
}
