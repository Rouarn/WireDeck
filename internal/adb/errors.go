package adb

import "errors"

// 错误集合。上层（Wails 绑定）会把这些错误转成前端可见的提示。
var (
	ErrAdbNotFound = errors.New("未找到 adb 可执行文件，请安装 Android platform-tools，或在设置中手动指定路径")
	ErrAdbServer   = errors.New("adb server 启动失败")
	ErrTimeout     = errors.New("adb 命令执行超时")
	ErrNoDevice    = errors.New("没有可用的设备")
)
