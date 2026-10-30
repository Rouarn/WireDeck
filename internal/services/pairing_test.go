package services

import "testing"

func TestParseQRCode(t *testing.T) {
	info, err := ParseQRCode("WIFI:T:ADB;S:adb-abc123;P:123456;;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Service != "adb-abc123" {
		t.Errorf("Service = %q, want adb-abc123", info.Service)
	}
	if info.Password != "123456" {
		t.Errorf("Password = %q, want 123456", info.Password)
	}

	if _, err := ParseQRCode("https://example.com"); err == nil {
		t.Error("expected error for non-WIFI payload")
	}

	if _, err := ParseQRCode("WIFI:T:ADB;S:adb-x;;"); err == nil {
		t.Error("expected error when password is missing")
	}
}

func TestServiceMatches(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   bool
	}{
		{"adb-abc123", "adb-abc123", true},
		{"adb-d2de5f92-1guyQc", "adb-d2de5f92", true},
		{"adb-other", "adb-abc", false},
		{"", "adb-abc", true},
	}

	for _, c := range cases {
		if got := serviceMatches(c.name, c.target); got != c.want {
			t.Errorf("serviceMatches(%q, %q) = %v, want %v", c.name, c.target, got, c.want)
		}
	}
}
