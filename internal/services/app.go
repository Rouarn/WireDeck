package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/apk"
	"woyuxiaji.cn/wiredeck/internal/model"
)

// AppService 封装应用列表、详情与常见操作。
type AppService struct {
	server *adb.Server
	emit   func(event string, data any)

	mu     sync.RWMutex
	labels map[string]string // pkg -> 已解析的显示名
}

// NewAppService 创建应用服务。
func NewAppService(server *adb.Server, emit func(event string, data any)) *AppService {
	service := &AppService{server: server, emit: emit, labels: make(map[string]string)}
	service.loadLabels()
	return service
}

func labelsFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "WireDeck", "app-labels.json")
}

func (s *AppService) loadLabels() {
	path := labelsFilePath()
	if path == "" {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var labels map[string]string
	if err := json.Unmarshal(data, &labels); err != nil || labels == nil {
		return
	}

	s.mu.Lock()
	s.labels = labels
	s.mu.Unlock()
}

func (s *AppService) persistLabels() {
	path := labelsFilePath()
	if path == "" {
		return
	}

	s.mu.RLock()
	data, err := json.MarshalIndent(s.labels, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return
	}

	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}

// LabelCacheInfo 返回应用名称缓存的条目数、路径与文件大小。
func (s *AppService) LabelCacheInfo() model.LabelCacheInfo {
	path := labelsFilePath()
	info := model.LabelCacheInfo{Path: path}

	s.mu.RLock()
	info.Count = len(s.labels)
	s.mu.RUnlock()

	if path != "" {
		if stat, err := os.Stat(path); err == nil {
			info.Size = stat.Size()
		}
	}
	return info
}

// ClearLabels 清空应用名称缓存（包括设备上已卸载应用留下的孤儿记录）。
func (s *AppService) ClearLabels() error {
	s.mu.Lock()
	s.labels = make(map[string]string)
	s.mu.Unlock()
	s.persistLabels()
	return nil
}

func (s *AppService) cachedLabel(pkg string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.labels[pkg]
}

func (s *AppService) setLabel(pkg, label string) {
	if label == "" {
		return
	}
	s.mu.Lock()
	s.labels[pkg] = label
	s.mu.Unlock()
	s.persistLabels()
}

// List 读取应用列表。scope: user（第三方）、system（系统）、all。
func (s *AppService) List(ctx context.Context, serial, scope string) ([]model.AppSummary, error) {
	if serial == "" {
		return nil, fmt.Errorf("未选择设备")
	}

	args := []string{"-s", serial, "shell", "pm", "list", "packages"}
	switch scope {
	case "user":
		args = append(args, "-3")
	case "system":
		args = append(args, "-s")
	}

	res, err := s.server.Run(ctx, 25*time.Second, args...)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("读取应用列表失败: %s", strings.TrimSpace(res.Output()))
	}

	apps := make([]model.AppSummary, 0, 128)
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "package:") {
			continue
		}
		pkg := strings.TrimPrefix(line, "package:")
		if pkg == "" {
			continue
		}
		apps = append(apps, model.AppSummary{Package: pkg, Label: s.cachedLabel(pkg)})
	}
	return apps, nil
}

// Detail 读取应用详情（版本、SDK、首次安装时间、已授予权限）。
func (s *AppService) Detail(ctx context.Context, serial, pkg string) (*model.AppDetail, error) {
	if serial == "" || pkg == "" {
		return nil, fmt.Errorf("设备与应用包名不能为空")
	}

	res, err := s.server.Run(ctx, 25*time.Second, "-s", serial, "shell", "dumpsys", "package", pkg)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("读取应用详情失败: %s", strings.TrimSpace(res.Output()))
	}

	detail := &model.AppDetail{Package: pkg, Permissions: make([]string, 0)}

	for _, raw := range strings.Split(res.Stdout, "\n") {
		line := strings.TrimSpace(raw)

		switch {
		case strings.HasPrefix(line, "versionCode="):
			fields := strings.Fields(line)
			for _, field := range fields {
				key, value, ok := strings.Cut(field, "=")
				if !ok {
					continue
				}
				switch key {
				case "versionCode":
					detail.VersionCode = value
				case "minSdk":
					detail.MinSDK = value
				case "targetSdk":
					detail.TargetSDK = value
				}
			}
		case strings.HasPrefix(line, "versionName="):
			detail.VersionName = strings.TrimPrefix(line, "versionName=")
		case strings.HasPrefix(line, "firstInstallTime="):
			detail.FirstInstall = strings.TrimPrefix(line, "firstInstallTime=")
		case strings.HasPrefix(line, "lastUpdateTime="):
			detail.LastUpdate = strings.TrimPrefix(line, "lastUpdateTime=")
		case strings.HasPrefix(line, "dataDir="):
			detail.DataDir = strings.TrimPrefix(line, "dataDir=")
		case strings.Contains(line, ".permission.") && strings.Contains(line, "granted=true"):
			// 形如：android.permission.CAMERA: granted=true, flags=0x0
			name := line
			if idx := strings.Index(line, ":"); idx > 0 {
				name = line[:idx]
			}
			if name != "" {
				detail.Permissions = append(detail.Permissions, strings.TrimSpace(name))
			}
		}
	}

	detail.Label = s.cachedLabel(pkg)
	if detail.Label == "" {
		if label := s.resolveLabel(ctx, serial, pkg); label != "" {
			s.setLabel(pkg, label)
			detail.Label = label
		}
	}

	return detail, nil
}

// Uninstall 卸载应用。keepData 为 true 时保留数据与缓存（-k）。
func (s *AppService) Uninstall(ctx context.Context, serial, pkg string, keepData bool) (model.AppOpResult, error) {
	args := []string{"-s", serial, "uninstall"}
	if keepData {
		args = append(args, "-k")
	}
	args = append(args, "--user", "0", pkg)

	res, err := s.server.Run(ctx, 60*time.Second, args...)
	if err != nil {
		return model.AppOpResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	ok := strings.Contains(raw, "Success")
	return model.AppOpResult{OK: ok, Message: humanizeUninstall(ok, raw), Raw: raw}, nil
}

func humanizeUninstall(ok bool, raw string) string {
	if ok {
		return "卸载成功"
	}
	switch {
	case strings.Contains(raw, "DELETE_FAILED_DEVICE_POLICY_MANAGER"):
		return "设备策略禁止卸载该应用"
	case strings.Contains(raw, "DELETE_FAILED_INTERNAL_ERROR"):
		return "卸载失败：系统内部错误，可重启设备后重试"
	case strings.Contains(raw, "not installed"):
		return "设备上不存在该应用"
	default:
		return "卸载失败: " + firstLine(raw)
	}
}

// Launch 通过 monkey 启动应用的 Launcher Activity。
func (s *AppService) Launch(ctx context.Context, serial, pkg string) (model.AppOpResult, error) {
	res, err := s.server.Run(ctx, 20*time.Second,
		"-s", serial, "shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
	if err != nil {
		return model.AppOpResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	ok := strings.Contains(raw, "Events injected: 1")
	message := "已启动"
	if !ok {
		message = "启动失败: " + firstLine(raw)
	}
	return model.AppOpResult{OK: ok, Message: message, Raw: raw}, nil
}

// ForceStop 强制停止应用，并检查进程是否真的退出。
func (s *AppService) ForceStop(ctx context.Context, serial, pkg string) (model.AppOpResult, error) {
	res, err := s.server.Run(ctx, 15*time.Second, "-s", serial, "shell", "am", "force-stop", pkg)
	if err != nil {
		return model.AppOpResult{}, err
	}

	raw := strings.TrimSpace(res.Output())

	// am force-stop 本身没有输出，用 pidof 判断进程是否还在。
	pid := ""
	if pidRes, pidErr := s.server.Run(ctx, 10*time.Second, "-s", serial, "shell", "pidof", pkg); pidErr == nil && pidRes != nil {
		pid = strings.TrimSpace(pidRes.Stdout)
	}

	ok := res.ExitCode == 0
	message := "已强制停止，进程已退出"
	if pid != "" {
		message = "已发送停止指令，但进程仍在运行（可能被系统立即拉起）"
	} else if !ok {
		message = "操作失败: " + firstLine(raw)
	}

	return model.AppOpResult{OK: ok, Message: message, Raw: raw}, nil
}

// ClearData 清除应用数据。
func (s *AppService) ClearData(ctx context.Context, serial, pkg string) (model.AppOpResult, error) {
	res, err := s.server.Run(ctx, 30*time.Second, "-s", serial, "shell", "pm", "clear", pkg)
	if err != nil {
		return model.AppOpResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	ok := strings.Contains(raw, "Success")
	message := "已清除应用数据"
	if !ok {
		message = "清除失败: " + firstLine(raw)
	}
	return model.AppOpResult{OK: ok, Message: message, Raw: raw}, nil
}

// Export 将应用 APK（含 split）拉取到指定目录，返回本地文件路径。
func (s *AppService) Export(ctx context.Context, serial, pkg, outDir string) ([]string, error) {
	if outDir == "" {
		return nil, fmt.Errorf("未选择导出目录")
	}

	res, err := s.server.Run(ctx, 15*time.Second, "-s", serial, "shell", "pm", "path", pkg)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("读取 APK 路径失败: %s", strings.TrimSpace(res.Output()))
	}

	devicePaths := make([]string, 0, 2)
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			devicePaths = append(devicePaths, strings.TrimPrefix(line, "package:"))
		}
	}
	if len(devicePaths) == 0 {
		return nil, fmt.Errorf("未找到该应用的 APK 路径")
	}

	localFiles := make([]string, 0, len(devicePaths))
	label := s.cachedLabel(pkg)

	for _, remote := range devicePaths {
		// 用包名重命名，split 保留后缀，避免导出一堆 base.apk。
		target := filepath.Join(outDir, pkg+".apk")
		if !strings.HasPrefix(filepath.Base(remote), "base") {
			target = filepath.Join(outDir, pkg+"-"+filepath.Base(remote))
		}

		pullErr := s.server.PullWithProgress(ctx, serial, remote, target, func(pct int) {
			if s.emit != nil {
				s.emit("app:progress", model.AppProgress{
					Pkg:     pkg,
					Stage:   "export",
					Percent: pct,
					Message: fmt.Sprintf("导出 %d%%", pct),
				})
			}
		})
		if pullErr != nil {
			return localFiles, pullErr
		}
		localFiles = append(localFiles, target)

		// 导出时顺便解析显示名，避免后续再拉一次 APK。
		if label == "" {
			if meta, err := apk.Inspect(target); err == nil && meta.Label != "" {
				label = meta.Label
				s.setLabel(pkg, label)
			}
		}
	}

	return localFiles, nil
}

// resolveLabel 拉取 base.apk 并解析 application-label，仅用于详情页按需获取。
func (s *AppService) resolveLabel(ctx context.Context, serial, pkg string) string {
	res, err := s.server.Run(ctx, 10*time.Second, "-s", serial, "shell", "pm", "path", pkg)
	if err != nil || res.ExitCode != 0 {
		return ""
	}

	remote := ""
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") && strings.HasSuffix(line, "base.apk") {
			remote = strings.TrimPrefix(line, "package:")
			break
		}
	}
	if remote == "" {
		return ""
	}

	dir, err := os.MkdirTemp("", "wiredeck-label-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(dir)

	local := filepath.Join(dir, "base.apk")
	pullErr := s.server.PullWithProgress(ctx, serial, remote, local, func(pct int) {
		if s.emit != nil {
			s.emit("app:progress", model.AppProgress{
				Pkg:     pkg,
				Stage:   "label",
				Percent: pct,
				Message: fmt.Sprintf("读取应用信息 %d%%", pct),
			})
		}
	})
	if pullErr != nil {
		return ""
	}

	meta, inspectErr := apk.Inspect(local)
	if inspectErr != nil {
		return ""
	}
	return meta.Label
}
