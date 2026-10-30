//go:build !windows

package services

import (
	"os/exec"
	"syscall"
)

// setDetached 让子进程脱离当前会话，应用退出时不会连带关闭 scrcpy。
func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
