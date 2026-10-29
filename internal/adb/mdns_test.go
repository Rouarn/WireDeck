package adb

import "testing"

func TestParseMDNSServices(t *testing.T) {
	output := `List of discovered mdns services
adb-1234	_adb-tls-pairing._tcp	192.168.1.23:37123
adb-5678	_adb-tls-connect._tcp	192.168.1.23:40123
adb-9999	_other._tcp	192.168.1.23:1234`

	services := ParseMDNSServices(output)
	if len(services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(services))
	}

	if services[0].Kind != "pairing" {
		t.Errorf("unexpected kind: %s", services[0].Kind)
	}
	if services[1].Kind != "connect" {
		t.Errorf("unexpected kind: %s", services[1].Kind)
	}
	if services[1].Host != "192.168.1.23" || services[1].Port != 40123 {
		t.Errorf("unexpected address parse: %s:%d", services[1].Host, services[1].Port)
	}
	if services[2].Kind != "other" {
		t.Errorf("unexpected kind: %s", services[2].Kind)
	}
}

func TestParseMDNSServicesEmpty(t *testing.T) {
	services := ParseMDNSServices("List of discovered mdns services\n\n")
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}
}

func TestParseMDNSServicesMalformed(t *testing.T) {
	output := `List of discovered mdns services
bad-line
adb-1 _adb-tls-pairing._tcp not-an-address`
	services := ParseMDNSServices(output)
	if len(services) != 0 {
		t.Fatalf("expected malformed lines to be skipped, got %d", len(services))
	}
}
