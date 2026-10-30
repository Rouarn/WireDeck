//go:build !windows

package platform

import "os/exec"

// PrepareCmd 在非 Windows 平台上无需处理。
func PrepareCmd(_ *exec.Cmd) {}
