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

# 资源变更后原签名失效，重新做 ad-hoc 签名（Apple Silicon 要求至少 ad-hoc）。
codesign --force --sign - "$APP"

echo "Built: $APP"
