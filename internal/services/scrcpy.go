package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

	if _, statErr := os.Stat(filepath.Join(dir, "scrcpy-server")); statErr != nil {
		return fmt.Errorf("安装包不完整：缺少 scrcpy-server，请重新解压完整压缩包")
	}

	// 把 scrcpy 的输出写入日志，便于排查无法唤起的问题。
	logPath := ""
	if cacheDir, cacheErr := os.UserCacheDir(); cacheErr == nil {
		logPath = filepath.Join(cacheDir, "WireDeck", "scrcpy.log")
		if mkErr := os.MkdirAll(filepath.Dir(logPath), 0o755); mkErr == nil {
			if f, createErr := os.Create(logPath); createErr == nil {
				cmd.Stdout = f
				cmd.Stderr = f
			}
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动投屏失败: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// scrcpy 正常启动后会持续运行；3 秒内退出说明启动失败，直接返回原因。
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("投屏启动失败: %v\n%s", err, tailLog(logPath, 600))
		}
		return fmt.Errorf("投屏进程已退出，日志：%s", logPath)
	case <-time.After(3 * time.Second):
	}

	go func() { <-done }()
	return nil
}

// tailLog 读取日志文件末尾，用于错误提示。
func tailLog(path string, limit int) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	text := strings.TrimSpace(string(data))
	if len(text) > limit {
		text = text[len(text)-limit:]
	}
	return "日志：" + path + "\n" + text
}

// findBundledAdb 返回应用内置的 adb（发布包或开发目录）。
func findBundledAdb() string {
	name := "adb"
	if runtime.GOOS == "windows" {
		name = "adb.exe"
	}

	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(base, "..", "Resources", "platform-tools", name),
			filepath.Join(base, "platform-tools", name),
			filepath.Join(base, "resources", "platform-tools", name),
		}
		for _, candidate := range candidates {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		dev := filepath.Join(wd, "resources", "platform-tools", name)
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
