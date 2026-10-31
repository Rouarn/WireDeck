package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/apk"
	"woyuxiaji.cn/wiredeck/internal/model"
	"woyuxiaji.cn/wiredeck/internal/platform"
	"woyuxiaji.cn/wiredeck/internal/services"
)

// appVersion 当前版本号，与 wails.json 的 info.productVersion 保持一致。
const appVersion = "1.0.1"

// App 是 Wails 绑定的根对象。导出方法会自动生成前端 TypeScript 绑定。
type App struct {
	ctx      context.Context
	server   *adb.Server
	devices  *services.DeviceService
	installs *services.InstallService
	apps     *services.AppService
	files    *services.FileService
	logs     *services.LogService
	update   *services.UpdateService
	scrcpy   *services.ScrcpyService
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
	app.update = services.NewUpdateService(appVersion)
	app.scrcpy = services.NewScrcpyService()
	return app
}

// startup 由 Wails 在应用启动时调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.devices.Start(ctx)
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		println(fmt.Sprintf("[drop] %d file(s) dropped", len(paths)))
		a.emit("files:dropped", map[string]any{"x": x, "y": y, "paths": paths})
	})

	// 启动后尝试重连历史设备，避免每次都要手动连接。
	go func() {
		if _, err := a.EnsureAdbReady(); err != nil {
			return
		}
		a.devices.ReconnectKnown(ctx)
	}()
}

// buildMenu 构建 macOS 菜单栏。
//
// Wails 的 Role 菜单（AppMenu/EditMenu/WindowMenu）文案是硬编码英文，
// 不跟随系统语言，因此这里全部手写中文菜单。
func (a *App) buildMenu() *menu.Menu {
	m := menu.NewMenu()

	appMenu := m.AddSubmenu("WireDeck")
	appMenu.AddText("关于 WireDeck", nil, func(_ *menu.CallbackData) {
		if a.ctx == nil {
			return
		}
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "关于 WireDeck",
			Message: fmt.Sprintf("WireDeck v%s\n无线 ADB 桌面控制台\n\n通过 Wi-Fi 或 USB 管理安卓设备。", appVersion),
		})
	})
	appMenu.AddSeparator()
	appMenu.AddText("隐藏 WireDeck", keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			runtime.WindowHide(a.ctx)
		}
	})
	appMenu.AddSeparator()
	appMenu.AddText("退出 WireDeck", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			runtime.Quit(a.ctx)
		}
	})

	editMenu := m.AddSubmenu("编辑")
	// 注册快捷键并将动作落到 WebView 的编辑命令上，
	// 避免按键被菜单拦截后没有行为。
	editMenu.AddText("撤销", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('undo')")
	})
	editMenu.AddText("重做", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('redo')")
	})
	editMenu.AddSeparator()
	editMenu.AddText("剪切", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('cut')")
	})
	editMenu.AddText("复制", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('copy')")
	})
	editMenu.AddText("粘贴", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('paste')")
	})
	editMenu.AddText("全选", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
		a.execJS("document.execCommand('selectAll')")
	})

	windowMenu := m.AddSubmenu("窗口")
	windowMenu.AddText("最小化", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			runtime.WindowMinimise(a.ctx)
		}
	})
	windowMenu.AddText("显示窗口", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			runtime.WindowShow(a.ctx)
		}
	})

	helpMenu := m.AddSubmenu("帮助")
	helpMenu.AddText("打开项目主页", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			runtime.BrowserOpenURL(a.ctx, "https://woyuxiaji.cn")
		}
	})

	return m
}

// execJS 在 WebView 中执行一段脚本，用于菜单项的编辑命令。
func (a *App) execJS(js string) {
	if a.ctx != nil {
		runtime.WindowExecJS(a.ctx, js)
	}
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

// PairByQRCode 根据无线调试二维码完成配对，并尽量自动连接。
func (a *App) PairByQRCode(content string) (model.PairResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.PairResult{}, err
	}
	return a.devices.PairByQRCode(a.context(), content)
}

// CreatePairingQR 生成扫码配对会话（电脑显示二维码，手机扫描）。
func (a *App) CreatePairingQR() (model.QRPairingSession, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.QRPairingSession{}, err
	}
	return a.devices.CreatePairingQR()
}

// WaitQRPairing 等待手机扫描二维码，并通过 mDNS 发现后完成配对。
func (a *App) WaitQRPairing(name, password string) (model.PairResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.PairResult{}, err
	}
	return a.devices.WaitPairing(a.context(), name, password, 120*time.Second)
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
		a.devices.RecordConnection(req.Host, req.Port, "")
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

// EnableWireless 把 USB 连接的设备切换为无线调试并连接。
func (a *App) EnableWireless(serial string) (model.ConnectResult, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return model.ConnectResult{}, err
	}
	res, err := a.devices.EnableWireless(a.context(), serial)
	if err == nil && res.OK {
		a.devices.RefreshAndEmit(a.context())
	}
	return res, err
}

// ReconnectDevices 尝试重连历史设备并自动连接 mDNS 发现的无线设备。
func (a *App) ReconnectDevices() error {
	if _, err := a.EnsureAdbReady(); err != nil {
		return err
	}
	a.devices.ReconnectKnown(a.context())
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

// LoadAppLabels 批量解析应用显示名，返回 包名 -> 名称。
func (a *App) LoadAppLabels(serial string, pkgs []string) (map[string]string, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.apps.LoadLabels(a.context(), serial, pkgs), nil
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

// ListDeviceFiles 读取设备目录内容，供文件浏览器使用。
func (a *App) ListDeviceFiles(serial, path string) ([]model.DeviceFile, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}
	return a.files.ListDir(a.context(), serial, path)
}

// StartLogcat 启动实时日志流。pkg 非空时只采集该应用的日志。
func (a *App) StartLogcat(serial, pkg string) (string, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return "", err
	}
	return a.logs.Start(a.context(), serial, pkg)
}

// StopLogcat 停止日志流。
func (a *App) StopLogcat(streamID string) error {
	a.logs.Stop(streamID)
	return nil
}

// ExportLogcat 把日志内容保存到用户选择的文件，返回保存路径（取消时为空）。
func (a *App) ExportLogcat(content string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出日志",
		DefaultFilename: "wiredeck-logcat.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "文本文件", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".txt") {
		path += ".txt"
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// CopyText 将文本写入系统剪贴板。
func (a *App) CopyText(text string) error {
	runtime.ClipboardSetText(a.context(), text)
	return nil
}

// AppVersion 返回当前应用版本。
func (a *App) AppVersion() string {
	return appVersion
}

// CheckUpdate 检查 GitHub Releases 是否有新版本。
func (a *App) CheckUpdate() (model.UpdateInfo, error) {
	return a.update.Check(a.context())
}

// OpenExternal 用系统默认浏览器打开链接。
func (a *App) OpenExternal(url string) error {
	if a.ctx == nil {
		return fmt.Errorf("应用未就绪")
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
}

// StartScrcpy 为指定设备启动 scrcpy 投屏窗口，title 用于窗口标题（通常是设备名）。
func (a *App) StartScrcpy(serial, title string) error {
	return a.scrcpy.Start(a.context(), serial, title)
}

// DownloadDir 返回本机下载目录，用于文件传输的默认保存路径。
func (a *App) DownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, "Downloads")
	if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
		return dir
	}
	return home
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
