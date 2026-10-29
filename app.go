package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
	"woyuxiaji.cn/wiredeck/internal/services"
)

// App 是 Wails 绑定的根对象。导出方法会自动生成前端 TypeScript 绑定。
type App struct {
	ctx     context.Context
	server  *adb.Server
	devices *services.DeviceService
}

// NewApp 创建应用实例。
func NewApp() *App {
	app := &App{server: adb.NewServer(adb.DefaultPort)}
	app.server.SetLogger(func(format string, args ...any) {
		println("[adb] " + fmt.Sprintf(format, args...))
	})
	app.devices = services.NewDeviceService(app.server, app.emit)
	return app
}

// startup 由 Wails 在应用启动时调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.devices.Start(ctx)
}

// shutdown 由 Wails 在应用退出时调用。
func (a *App) shutdown(_ context.Context) {
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
