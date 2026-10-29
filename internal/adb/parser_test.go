package adb

import "testing"

func TestParseDevices(t *testing.T) {
	output := `List of devices attached
192.168.1.23:37123	device product:panther model:Pixel_7 device:panther transport_id:1
R58M1234567	unauthorized usb:338690048X transport_id:2
emulator-5554	device product:sdk_gphone64_arm64 model:sdk_gphone64_arm64 device:emu64a transport_id:3
* daemon started successfully`

	devices := ParseDevices(output, 1700000000000)
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}

	wifi := devices[0]
	if wifi.Serial != "192.168.1.23:37123" {
		t.Errorf("unexpected serial: %s", wifi.Serial)
	}
	if wifi.Transport != "wifi" {
		t.Errorf("expected wifi transport, got %s", wifi.Transport)
	}
	if wifi.Model != "Pixel_7" {
		t.Errorf("unexpected model: %s", wifi.Model)
	}
	if wifi.State != "device" {
		t.Errorf("unexpected state: %s", wifi.State)
	}

	usb := devices[1]
	if usb.Transport != "usb" {
		t.Errorf("expected usb transport, got %s", usb.Transport)
	}
	if usb.State != "unauthorized" {
		t.Errorf("unexpected state: %s", usb.State)
	}

	emulator := devices[2]
	if emulator.Transport != "usb" {
		t.Errorf("expected usb transport for emulator serial, got %s", emulator.Transport)
	}
}

func TestIsWirelessSerial(t *testing.T) {
	cases := map[string]bool{
		"192.168.1.23:37123": true,
		"10.0.0.5:5555":      true,
		"adb-d2de5f92-1guyQc._adb-tls-connect._tcp": true,
		"R58M1234567":    false,
		"emulator-5554":  false,
		"localhost:5555": false,
	}
	for serial, want := range cases {
		if got := isWirelessSerial(serial); got != want {
			t.Errorf("isWirelessSerial(%q) = %v, want %v", serial, got, want)
		}
	}
}
