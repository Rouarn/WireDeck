package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"woyuxiaji.cn/wiredeck/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	lock, err := platform.LockSingleInstance("WireDeck")
	if err != nil {
		fmt.Fprintf(os.Stderr, "WireDeck 启动失败：%v\n", err)
		os.Exit(1)
	}
	defer func() { _ = lock.Unlock() }()

	app := NewApp()

	err = wails.Run(&options.App{
		Title:            "WireDeck",
		Width:            1180,
		Height:           760,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		OnStartup:        app.startup,
		Menu:             app.buildMenu(),
		// 点击窗口关闭按钮时直接退出应用，而不是留在后台。
		OnBeforeClose: func(ctx context.Context) bool {
			go runtime.Quit(ctx)
			return true
		},
		OnShutdown: func(ctx context.Context) {
			app.shutdown(ctx)
			_ = lock.Unlock()
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		AssetServer: &assetserver.Options{Assets: assets},
		Bind:        []interface{}{app},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "WireDeck 运行失败：%v\n", err)
		os.Exit(1)
	}
}
