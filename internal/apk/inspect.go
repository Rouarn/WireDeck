package apk

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"woyuxiaji.cn/wiredeck/internal/model"
	"woyuxiaji.cn/wiredeck/internal/platform"
)

// ErrAapt2NotFound 表示找不到用于解析 APK 的 aapt2。
var ErrAapt2NotFound = errors.New("未找到 aapt2，无法解析 APK 信息；请安装 Android SDK build-tools")

// FindAapt2 按 PATH → Android SDK build-tools 的顺序查找 aapt2。
func FindAapt2() string {
	name := "aapt2"
	if runtime.GOOS == "windows" {
		name = "aapt2.exe"
	}

	if path, err := exec.LookPath("aapt2"); err == nil {
		return path
	}

	// 安装包内置的 aapt2（Windows 发布包会带在 platform-tools 目录）。
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(base, name),
			filepath.Join(base, "platform-tools", name),
			filepath.Join(base, "build-tools", name),
			filepath.Join(base, "..", "Resources", "platform-tools", name),
		}
		for _, candidate := range candidates {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	home, _ := os.UserHomeDir()
	roots := []string{
		os.Getenv("ANDROID_HOME"),
		os.Getenv("ANDROID_SDK_ROOT"),
		filepath.Join(home, "Library/Android/sdk"),
		filepath.Join(home, "Android/Sdk"),
	}

	for _, root := range roots {
		if root == "" {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, "build-tools", "*", name))
		if err != nil || len(matches) == 0 {
			continue
		}
		// build-tools 版本按目录名排序，取最高版本。
		sort.Strings(matches)
		return matches[len(matches)-1]
	}

	return ""
}

// Inspect 调用 aapt2 dump badging 解析 APK 元信息。
func Inspect(path string) (*model.ApkMeta, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取安装包: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("这是一个目录，请选择 APK 文件")
	}
	if !strings.EqualFold(filepath.Ext(path), ".apk") {
		return nil, fmt.Errorf("不是 APK 文件: %s", filepath.Base(path))
	}

	aapt2 := FindAapt2()
	if aapt2 == "" {
		return nil, ErrAapt2NotFound
	}

	cmd := exec.Command(aapt2, "dump", "badging", path)
	platform.PrepareCmd(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("解析 APK 失败: %w", err)
	}

	meta := &model.ApkMeta{
		Path:        path,
		Size:        info.Size(),
		PackageName: "",
		ABIs:        []string{},
	}
	parseBadging(string(out), meta)
	return meta, nil
}

func parseBadging(output string, meta *model.ApkMeta) {
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "package:"):
			meta.PackageName = extractQuoted(line, "name")
			meta.VersionCode = atoi(extractQuoted(line, "versionCode"))
			meta.VersionName = extractQuoted(line, "versionName")
		case strings.HasPrefix(line, "sdkVersion:"):
			meta.MinSDK = atoi(trimQuotes(strings.TrimPrefix(line, "sdkVersion:")))
		case strings.HasPrefix(line, "targetSdkVersion:"):
			meta.TargetSDK = atoi(trimQuotes(strings.TrimPrefix(line, "targetSdkVersion:")))
		case strings.HasPrefix(line, "application-label:"):
			meta.Label = trimQuotes(strings.TrimPrefix(line, "application-label:"))
		case strings.HasPrefix(line, "native-code:"):
			meta.ABIs = parseQuotedList(strings.TrimPrefix(line, "native-code:"))
		}
	}
}

func extractQuoted(line, key string) string {
	marker := key + "='"
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(marker):]
	if end := strings.Index(rest, "'"); end >= 0 {
		return rest[:end]
	}
	return ""
}

func trimQuotes(value string) string {
	return strings.Trim(strings.TrimSpace(value), "'")
}

func parseQuotedList(text string) []string {
	fields := strings.Fields(strings.TrimSpace(text))
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		value := trimQuotes(field)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func atoi(value string) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return n
}
