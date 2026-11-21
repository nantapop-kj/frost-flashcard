#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
APP="build/bin/Frost Flashcard.app"
BIN="$APP/Contents/MacOS/frost-flashcard"
DMG="build/bin/Frost-Flashcard-${VERSION}.dmg"
STAGE="build/bin/dmg-stage"

FLAGS=(-tags "private_mac_apis,production" -trimpath -ldflags "-s -w -X main.version=$VERSION")
export CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=11.0

echo "▶ build universal binary (arm64 + amd64), version $VERSION"
rm -rf "build/bin/Frost Flashcard.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
GOARCH=arm64 go build "${FLAGS[@]}" -o build/bin/frost-flashcard-arm64 ./cmd
GOARCH=amd64 go build "${FLAGS[@]}" -o build/bin/frost-flashcard-amd64 ./cmd
lipo -create -output "$BIN" build/bin/frost-flashcard-arm64 build/bin/frost-flashcard-amd64
rm -f build/bin/frost-flashcard-arm64 build/bin/frost-flashcard-amd64

echo "▶ assemble app bundle"
cp build/darwin/Info.plist "$APP/Contents/Info.plist"
cp build/darwin/icon.icns "$APP/Contents/Resources/icon.icns"
if [[ "$VERSION" != "dev" ]]; then
  PLIST_VERSION="${VERSION#v}"
  /usr/libexec/PlistBuddy \
    -c "Set :CFBundleShortVersionString $PLIST_VERSION" \
    -c "Set :CFBundleVersion $PLIST_VERSION" \
    "$APP/Contents/Info.plist"
fi
codesign --force --deep --sign - "$APP"

if [[ "${1:-}" == "--dmg" ]]; then
  echo "▶ create $DMG"
  rm -rf build/bin/dmg-stage
  mkdir -p "$STAGE"
  cp -R "$APP" "$STAGE/"
  ln -s /Applications "$STAGE/Applications"
  rm -f "$DMG"
  hdiutil create -volname "Frost Flashcard" -srcfolder "$STAGE" -ov -format UDZO "$DMG" >/dev/null
  rm -rf build/bin/dmg-stage
fi

echo "✅ done: $APP"
