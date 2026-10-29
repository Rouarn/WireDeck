package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
)

// AppService 封装应用列表、详情与常见操作。
type AppService struct {
	server *adb.Server
}

// NewAppService 创建应用服务。
func NewAppService(server *adb.Server) *AppService {
	return &AppService{server: server}
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
		apps = append(apps, model.AppSummary{Package: pkg})
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
	currentPerm := ""

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
		case strings.HasPrefix(line, "android.permission."):
			currentPerm = line
		case strings.HasPrefix(line, "granted=true") && currentPerm != "":
			detail.Permissions = append(detail.Permissions, currentPerm)
			currentPerm = ""
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

// ForceStop 强制停止应用。
func (s *AppService) ForceStop(ctx context.Context, serial, pkg string) (model.AppOpResult, error) {
	res, err := s.server.Run(ctx, 15*time.Second, "-s", serial, "shell", "am", "force-stop", pkg)
	if err != nil {
		return model.AppOpResult{}, err
	}

	raw := strings.TrimSpace(res.Output())
	ok := res.ExitCode == 0
	message := "已强制停止"
	if !ok {
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
	for _, remote := range devicePaths {
		pullRes, pullErr := s.server.Run(ctx, 120*time.Second, "-s", serial, "pull", remote, outDir)
		if pullErr != nil {
			return localFiles, pullErr
		}
		if pullRes.ExitCode != 0 {
			return localFiles, fmt.Errorf("拉取失败: %s", strings.TrimSpace(pullRes.Output()))
		}
		localFiles = append(localFiles, outDir)
	}

	return localFiles, nil
}
