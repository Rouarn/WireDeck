#!/usr/bin/env bash
# WireDeck DMG 打包：构建 → 组装 → 生成 DMG
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:-0.1.0}"
APP="build/bin/WireDeck.app"
STAGE="build/dmg-stage"
DMG="build/WireDeck-${VERSION}.dmg"

./scripts/build.sh

rm -rf "$STAGE" "$DMG"
mkdir -p "$STAGE"
cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
if [ -f "docs/安装说明.txt" ]; then
  cp "docs/安装说明.txt" "$STAGE/安装说明.txt"
fi

hdiutil create -volname "WireDeck" -srcfolder "$STAGE" -ov -format UDZO "$DMG"
rm -rf "$STAGE"

echo "DMG: $DMG"
