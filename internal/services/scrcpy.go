package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// ScrcpyService 负责启动 scrcpy 投屏窗口（独立进程，不嵌入应用）。
type ScrcpyService struct{}

// NewScrcpyService 创建投屏服务。
func NewScrcpyService() *ScrcpyService {
	return &ScrcpyService{}
}

// Start 为指定设备打开 scrcpy 窗口。
func (s *ScrcpyService) Start(_ context.Context, serial, title string) error {
	if serial == "" {
		return fmt.Errorf("未选择设备")
	}
	if title == "" {
		title = serial
	}

	exe, err := findScrcpy()
	if err != nil {
		return err
	}

	dir := filepath.Dir(exe)
	adbPath := filepath.Join(dir, "adb")
	if _, statErr := os.Stat(adbPath); statErr != nil {
		adbPath = ""
	}

	cmd := exec.Command(exe, "--serial", serial, "--no-audio", "--window-title", "WireDeck · "+title)
	cmd.Dir = dir

	env := os.Environ()
	if adbPath != "" {
		env = append(env, "ADB="+adbPath, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	cmd.Env = env

	if runtime.GOOS != "windows" {
		// 让 scrcpy 独立于应用运行，应用退出时不连带关闭窗口。
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动投屏失败: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	return nil
}

// findScrcpy 按优先级查找 scrcpy：应用内置 → 项目 resources → PATH。
func findScrcpy() (string, error) {
	if exe, err := os.Executable(); err == nil {
		bundled := filepath.Join(filepath.Dir(exe), "..", "Resources", "scrcpy", "scrcpy")
		if info, statErr := os.Stat(bundled); statErr == nil && !info.IsDir() {
			return bundled, nil
		}
	}

	if wd, err := os.Getwd(); err == nil {
		dev := filepath.Join(wd, "resources", "scrcpy", "scrcpy")
		if info, statErr := os.Stat(dev); statErr == nil && !info.IsDir() {
			return dev, nil
		}
	}

	if path, err := exec.LookPath("scrcpy"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("未找到 scrcpy，请重新安装应用")
}
