//go:build windows

package platform

import (
	"os/exec"
	"syscall"
)

// CREATE_NO_WINDOW 让 GUI 进程启动的控制台子进程不创建窗口。
const createNoWindow = 0x08000000

// PrepareCmd 防止 Windows 下 adb / aapt2 / scrcpy 等子进程弹出黑色控制台窗口。
func PrepareCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
