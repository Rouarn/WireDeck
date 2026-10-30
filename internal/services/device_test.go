package services

import (
	"testing"

	"woyuxiaji.cn/wiredeck/internal/model"
)

func TestPreferConnection(t *testing.T) {
	devices := []model.Device{
		{Serial: "192.168.1.5:5555", Transport: "wifi"},
		{Serial: "adb-abc._adb-tls-connect._tcp", Transport: "wifi"},
		{Serial: "usb-device", Transport: "usb"},
	}

	// mDNS 优先于普通 Wi-Fi
	if a, _ := preferConnection(0, 1, devices); a != 1 {
		t.Errorf("mDNS should win, got index %d", a)
	}

	// Wi-Fi 优先于 USB
	if a, _ := preferConnection(0, 2, devices); a != 0 {
		t.Errorf("wifi should win over usb, got index %d", a)
	}
}
