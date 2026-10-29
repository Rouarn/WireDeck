package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
)

// App 是 Wails 绑定的根对象。导出方法会自动生成前端 TypeScript 绑定。
type App struct {
	ctx    context.Context
	server *adb.Server
}

// NewApp 创建应用实例。
func NewApp() *App {
	app := &App{server: adb.NewServer(adb.DefaultPort)}
	app.server.SetLogger(func(format string, args ...any) {
		println("[adb] " + fmt.Sprintf(format, args...))
	})
	return app
}

// startup 由 Wails 在应用启动时调用，保存运行上下文。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// EnsureAdbReady 定位 adb 可执行文件、启动独立 server 并返回环境信息。
func (a *App) EnsureAdbReady() (*model.AdbEnv, error) {
	return a.server.EnsureReady(a.context())
}

// SetAdbPath 允许用户在设置中手动指定 adb 路径。
func (a *App) SetAdbPath(path string) error {
	return a.server.Resolve(path)
}

// ListDevices 返回当前 adb 识别到的设备列表。
func (a *App) ListDevices() ([]model.Device, error) {
	if _, err := a.EnsureAdbReady(); err != nil {
		return nil, err
	}

	res, err := a.server.Run(a.context(), 8*time.Second, "devices", "-l")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("adb devices 执行失败: %s", firstNonEmpty(res.Stderr, res.Stdout))
	}

	return adb.ParseDevices(res.Stdout, time.Now().UnixMilli()), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
