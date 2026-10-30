package services

import "testing"

func TestExportFileName(t *testing.T) {
	cases := []struct {
		pkg   string
		label string
		split string
		want  string
	}{
		{"com.example", "", "", "com.example.apk"},
		{"com.example", "示例应用", "", "示例应用 (com.example).apk"},
		{"com.example", "示例应用", "split_config.arm64_v8a", "示例应用 (com.example)-split_config.arm64_v8a.apk"},
		{"com.example", "A/B", "", "A_B (com.example).apk"},
	}

	for _, c := range cases {
		if got := exportFileName(c.pkg, c.label, c.split); got != c.want {
			t.Errorf("exportFileName(%q, %q, %q) = %q, want %q", c.pkg, c.label, c.split, got, c.want)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(`a:b*c?"d<e>f|g`)
	want := "a_b_c__d_e_f_g"
	if got != want {
		t.Errorf("sanitizeFileName = %q, want %q", got, want)
	}
}
