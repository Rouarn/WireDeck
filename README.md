# WireDeck

> 在电脑上通过 Wi-Fi（无线调试）管理安卓手机、一键安装 APK 的桌面客户端。

## 状态

**立项阶段**：仅完成开发文档，代码尚未开始。

## 文档

- [开发文档](docs/开发文档.md) —— 需求、架构、Wails 绑定契约、ADB 命令映射、错误码、UI、打包、测试与排期

## 项目标识

| 项 | 值 |
| --- | --- |
| 名称 | WireDeck |
| Bundle ID | `cn.woyuxiaji.wiredeck` |
| 域名 | `woyuxiaji.cn` |
| Go module | `woyuxiaji.cn/wiredeck` |

## 目标能力（MVP）

1. 局域网自动发现开启无线调试的安卓设备，输入配对码完成连接（Android 11+）
2. USB 数据线直连：插入即识别，可直接安装与调试，也支持低版本设备转为无线
3. 拖拽 APK 安装，支持批量、覆盖、降级、授予权限、多 split
4. 安装失败给出中文原因与处理建议（含各厂商"USB 安装"开关路径）
5. 应用列表 / 卸载 / 导出，截图与文件互传，logcat 实时查看

## 技术栈

Wails v2（系统 WebView）+ Go 1.27 后端 + Vue 3 + TypeScript + Vite + Pinia 前端；设备操作通过内置或首次启动下载的 `platform-tools`（`adb`）执行。

## 分发

首发仅 macOS，采用 ad-hoc 签名、不公证（无付费开发者账号），DMG 内附首次打开放行指引；后续有账号再升级 Developer ID + 公证。Windows 排到 v0.2 之后。

## 快速开始

待 M0 脚手架完成后补充。当前可先阅读开发文档确认范围与技术决策。
