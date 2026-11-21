#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
mkdir -p build/bin

echo "▶ embed icon and manifest"
go run github.com/akavel/rsrc@v0.10.2 \
  -arch amd64 \
  -ico build/windows/icon.ico \
  -manifest build/windows/frost-flashcard.exe.manifest \
  -o cmd/rsrc_windows_amd64.syso
trap 'rm -f cmd/rsrc_windows_amd64.syso' EXIT

echo "▶ build frost-flashcard.exe, version $VERSION"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -tags production -trimpath \
  -ldflags "-s -w -H windowsgui -X main.version=$VERSION" \
  -o build/bin/frost-flashcard.exe ./cmd

echo "✅ done: build/bin/frost-flashcard.exe"
