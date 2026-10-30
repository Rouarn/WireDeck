//go:build windows

package services

import "os/exec"

// setDetached 在 Windows 上无需额外处理（子进程默认独立于父进程）。
func setDetached(_ *exec.Cmd) {}
