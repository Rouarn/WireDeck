package adb

import "testing"

func TestExplainInstallFailure(t *testing.T) {
	cases := []struct {
		output   string
		wantCode string
	}{
		{
			output:   "Failure [INSTALL_FAILED_UPDATE_INCOMPATIBLE: Package com.example signatures do not match previously installed version]",
			wantCode: "INSTALL_FAILED_UPDATE_INCOMPATIBLE",
		},
		{
			output:   "Failure [INSTALL_FAILED_USER_RESTRICTED]",
			wantCode: "INSTALL_FAILED_USER_RESTRICTED",
		},
		{
			output:   "Performing Streamed Install\nFailure [INSTALL_FAILED_VERSION_DOWNGRADE: Downgrade detected]",
			wantCode: "INSTALL_FAILED_VERSION_DOWNGRADE",
		},
		{
			output:   "Success",
			wantCode: "",
		},
	}

	for _, tc := range cases {
		code, advice := ExplainInstallFailure(tc.output)
		if code != tc.wantCode {
			t.Errorf("ExplainInstallFailure(%q) code = %q, want %q", tc.output, code, tc.wantCode)
		}
		if tc.wantCode != "" && advice == "" {
			t.Errorf("expected advice for %q, got empty", tc.wantCode)
		}
	}
}
