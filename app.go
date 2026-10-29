package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/apk"
	"woyuxiaji.cn/wiredeck/internal/model"
	"woyuxiaji.cn/wiredeck/internal/platform"
	"woyuxiaji.cn/wiredeck/internal/services"
)

// App 是 Wails 绑定的根对象。导出方法会自动生成前端 TypeScript 绑定。
type App struct {
	ctx      context.Context
	server   *adb.Server
	devices  *services.DeviceService
	installs *services.InstallService
	apps     *services.AppService
	files    *services.FileService
	logs     *services.LogService
}

// NewApp 创建应用实例。
func NewApp() *App {
	app := &App{server: adb.NewServer(adb.DefaultPort)}
	app.server.SetLogger(func(format string, args ...any) {
		println("[adb] " + fmt.Sprintf(format, args...))
	})
	app.devices = services.NewDeviceService(app.server, app.emit)
	app.installs = services.NewInstallService(app.server, app.emit)
	app.apps = services.NewAppService(app.server, app.emit)
	app.files = services.NewFileService(app.server, app.emit)
	app.logs = services.NewLogService(app.server, app.emit)
	return app
}

// startup 由 Wails 在应用启动时调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.devices.Start(ctx)
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		println(fmt.Sprintf("[drop] %d file(s) dropped", len(paths)))
		a.emit("files:dropped", paths)
	})
}

// shutdown 由 Wails 在应用退出时调用。
func (a *App) shutdown(_ context.Context) {
	a.logs.StopAll()
	a.devices.Stop()
}

// emit 向前端推送事件；ctx 尚未就绪时静默丢弃。
func (a *App) emit(event string, data any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, event, data)
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// EnsureAdbReady 定位 adb、启动独立 server 并返回环境信息。
func (a *App) EnsureAdbReady() (*model.AdbEnv, error) {
	return a.server.EnsureReady(a.context())
}

// SetAdbPath 允许用户在设置中手动指定 adb 路径。
func (a *App) SetAdbPath(path string) error {
	return a.server.Resolve(path)
}

// ListDevices 主动刷新并返回当前设备列表。
func (a *App) ListDevices() ([]model.Device, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.devices.Refresh(a.context())
}

// DiscoverServices 返回 mDNS 发现的配对 / 连接服务。
func (a *App) DiscoverServices() ([]model.MDNSService, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.devices.Discover(a.context())
}

// PairDevice 执行无线调试配对（adb pair）。
func (a *App) PairDevice(req model.PairRequest) (model.PairResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.PairResult{}, err
	}
	return a.devices.Pair(a.context(), req)
}

// ConnectDevice 连接无线调试设备（adb connect）。
func (a *App) ConnectDevice(req model.ConnectRequest) (model.ConnectResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.ConnectResult{}, err
	}

	res, err := a.devices.Connect(a.context(), req)
	if err != nil {
		return res, err
	}
	if res.OK {
		a.devices.RefreshAndEmit(a.context())
	}
	return res, nil
}

// DisconnectDevice 断开设备。
func (a *App) DisconnectDevice(serial string) error {
	if _, err := a.EnsureAdbReady(); err != nil {
		return err
	}
	if err := a.devices.Disconnect(a.context(), serial); err != nil {
		return err
	}
	a.devices.RefreshAndEmit(a.context())
	return nil
}

// GetHostIdentity 返回电脑在手机「已配对设备」列表中的显示身份。
func (a *App) GetHostIdentity() (model.HostIdentity, error) {
	return platform.GetHostIdentity(), nil
}

// InspectApk 解析 APK 元信息（包名、版本、SDK、ABI）。
func (a *App) InspectApk(path string) (*model.ApkMeta, error) {
	return apk.Inspect(path)
}

// PickApkFile 打开系统文件选择框，返回选中的 APK 路径。
func (a *App) PickApkFile() (string, error) {
	return runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{
		Title: "选择 APK",
		Filters: []runtime.FileFilter{
			{DisplayName: "Android 安装包 (*.apk)", Pattern: "*.apk"},
		},
	})
}

// StartInstall 将 APK 安装任务加入队列，返回 taskId。
func (a *App) StartInstall(req model.InstallRequest) (string, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return "", err
	}
	return a.installs.Start(a.context(), req)
}

// CancelInstall 取消一个安装任务。
func (a *App) CancelInstall(taskID string) error {
	a.installs.Cancel(taskID)
	return nil
}

// ListApps 读取设备应用列表。scope: user | system | all。
func (a *App) ListApps(serial, scope string) ([]model.AppSummary, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.apps.List(a.context(), serial, scope)
}

// AppDetail 读取应用详情。
func (a *App) AppDetail(serial, pkg string) (*model.AppDetail, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.apps.Detail(a.context(), serial, pkg)
}

// UninstallApp 卸载应用。
func (a *App) UninstallApp(serial, pkg string, keepData bool) (model.AppOpResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.AppOpResult{}, err
	}
	return a.apps.Uninstall(a.context(), serial, pkg, keepData)
}

// LaunchApp 启动应用。
func (a *App) LaunchApp(serial, pkg string) (model.AppOpResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.AppOpResult{}, err
	}
	return a.apps.Launch(a.context(), serial, pkg)
}

// ForceStopApp 强制停止应用。
func (a *App) ForceStopApp(serial, pkg string) (model.AppOpResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.AppOpResult{}, err
	}
	return a.apps.ForceStop(a.context(), serial, pkg)
}

// ClearAppData 清除应用数据。
func (a *App) ClearAppData(serial, pkg string) (model.AppOpResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.AppOpResult{}, err
	}
	return a.apps.ClearData(a.context(), serial, pkg)
}

// PickDirectory 打开目录选择框，用于导出 APK。
func (a *App) PickDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.context(), runtime.OpenDialogOptions{
		Title: "选择导出目录",
	})
}

// ExportApp 将应用 APK 导出到指定目录。
func (a *App) ExportApp(serial, pkg, outDir string) ([]string, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.apps.Export(a.context(), serial, pkg, outDir)
}

// AppLabelCacheInfo 返回应用名称缓存的占用情况。
func (a *App) AppLabelCacheInfo() (model.LabelCacheInfo, error) {
	return a.apps.LabelCacheInfo(), nil
}

// ClearAppLabelCache 清空应用名称缓存。
func (a *App) ClearAppLabelCache() error {
	return a.apps.ClearLabels()
}

// PickFile 打开文件选择框。
func (a *App) PickFile() (string, error) {
	return runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{
		Title: "选择文件",
	})
}

// Screenshot 截取设备屏幕并保存到图片目录。
func (a *App) Screenshot(serial string) (*model.ScreenshotResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.files.Screenshot(a.context(), serial, "")
}

// PushFile 将本地文件推送到设备。
func (a *App) PushFile(serial, local, remote string) error {
	if _, err := a.EnsureAdbReady(); err != nil {
		return err
	}
	return a.files.Push(a.context(), serial, local, remote)
}

// PullFile 从设备拉取文件到本地。
func (a *App) PullFile(serial, remote, local string) error {
	if _, err := a.EnsureAdbReady(); err != nil {
		return err
	}
	return a.files.Pull(a.context(), serial, remote, local)
}

// StartLogcat 启动实时日志流，返回 streamId。
func (a *App) StartLogcat(serial, filter string) (string, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return "", err
	}
	return a.logs.Start(a.context(), serial, filter)
}

// StopLogcat 停止日志流。
func (a *App) StopLogcat(streamID string) error {
	a.logs.Stop(streamID)
	return nil
}

// ExecShell 执行一条 adb shell 命令。
func (a *App) ExecShell(serial, command string) (model.ShellResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.ShellResult{}, err
	}

	res, err := a.server.Run(a.context(), 30*time.Second, "-s", serial, "shell", command)
	if err != nil {
		return model.ShellResult{}, err
	}

	return model.ShellResult{
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: res.ExitCode,
	}, nil
}
