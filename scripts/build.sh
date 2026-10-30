#!/usr/bin/env bash
# WireDeck 构建脚本：wails build + 本地化资源 + 重新签名
set -euo pipefail

cd "$(dirname "$0")/.."

export PATH="$PATH:$(go env GOPATH)/bin"

wails build -clean "$@"

APP="build/bin/WireDeck.app"
RES="$APP/Contents/Resources"

# 系统文件选择面板等原生 UI 依据应用的本地化资源决定语言，
# Wails 不会复制 build/darwin 下的 .lproj 目录，这里手动补齐。
if [ -d "build/darwin/zh-Hans.lproj" ]; then
  cp -R "build/darwin/zh-Hans.lproj" "$RES/"
fi

# 内置 adb，让没有安装 Android SDK 的电脑也能直接使用。
ADB_SRC="${ADB_BIN:-$HOME/Library/Android/sdk/platform-tools/adb}"
if [ -f "$ADB_SRC" ]; then
  mkdir -p "$RES/platform-tools"
  cp "$ADB_SRC" "$RES/platform-tools/adb"
  chmod +x "$RES/platform-tools/adb"
else
  echo "warning: 未找到 adb（$ADB_SRC），发布包将依赖系统 adb" >&2
fi

# 内置 scrcpy，用于投屏。
if [ -d "resources/scrcpy" ]; then
  mkdir -p "$RES/scrcpy"
  cp -R resources/scrcpy/. "$RES/scrcpy/"
  # scrcpy 复用应用内置的 adb，避免打包两份。
  rm -f "$RES/scrcpy/adb"
  chmod +x "$RES/scrcpy/scrcpy" 2>/dev/null || true
fi

# 资源变更后原签名失效，重新做 ad-hoc 签名（Apple Silicon 要求至少 ad-hoc）。
codesign --force --sign - "$APP"

echo "Built: $APP"
