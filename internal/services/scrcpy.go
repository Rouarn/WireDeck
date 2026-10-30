package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"woyuxiaji.cn/wiredeck/internal/platform"
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

	cmd := exec.Command(exe, "--serial", serial, "--no-audio", "--window-title", "WireDeck · "+title)
	platform.PrepareCmd(cmd)
	cmd.Dir = dir

	// 复用应用内置的 adb，避免 scrcpy 再带一份。
	env := os.Environ()
	if adbPath := findBundledAdb(); adbPath != "" {
		env = append(env, "ADB="+adbPath)
	}
	cmd.Env = env

	// 让 scrcpy 独立于应用运行，应用退出时不连带关闭窗口。
	setDetached(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动投屏失败: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	return nil
}

// findBundledAdb 返回应用内置的 adb（发布包或开发目录）。
func findBundledAdb() string {
	if exe, err := os.Executable(); err == nil {
		bundled := filepath.Join(filepath.Dir(exe), "..", "Resources", "platform-tools", "adb")
		if info, statErr := os.Stat(bundled); statErr == nil && !info.IsDir() {
			return bundled
		}
	}

	if wd, err := os.Getwd(); err == nil {
		dev := filepath.Join(wd, "resources", "platform-tools", "adb")
		if info, statErr := os.Stat(dev); statErr == nil && !info.IsDir() {
			return dev
		}
	}

	return ""
}

// findScrcpy 按优先级查找 scrcpy：应用内置 → 项目 resources → PATH。
func findScrcpy() (string, error) {
	name := "scrcpy"
	if runtime.GOOS == "windows" {
		name = "scrcpy.exe"
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(base, "..", "Resources", "scrcpy", name),
			filepath.Join(base, "scrcpy", name),
			filepath.Join(base, "resources", "scrcpy", name),
		}
		for _, c := range candidates {
			if info, statErr := os.Stat(c); statErr == nil && !info.IsDir() {
				return c, nil
			}
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
