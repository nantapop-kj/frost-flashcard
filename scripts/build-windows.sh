#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
GO="${GO:-go}"
if ! command -v "$GO" >/dev/null 2>&1 && command -v go.exe >/dev/null 2>&1; then
  GO=go.exe
fi
mkdir -p build/bin

echo "▶ embed icon and manifest"
"$GO" run github.com/akavel/rsrc@v0.10.2 \
  -arch amd64 \
  -ico build/windows/icon.ico \
  -manifest build/windows/frost-flashcard.exe.manifest \
  -o cmd/rsrc_windows_amd64.syso
trap 'rm -f cmd/rsrc_windows_amd64.syso' EXIT

echo "▶ build frost-flashcard.exe, version $VERSION"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 WSLENV="${WSLENV:+$WSLENV:}CGO_ENABLED:GOOS:GOARCH" \
  "$GO" build \
  -tags production -trimpath \
  -ldflags "-s -w -H windowsgui -X main.version=$VERSION" \
  -o build/bin/frost-flashcard.exe ./cmd

echo "✅ done: build/bin/frost-flashcard.exe"

[[ "${1:-}" == "--installer" ]] || exit 0

APP_VERSION="${VERSION#v}"
[[ "$APP_VERSION" =~ ^[0-9]+(\.[0-9]+)*$ ]] || APP_VERSION="0.0.0"

find_iscc() {
  if command -v iscc >/dev/null 2>&1; then
    command -v iscc
    return
  fi
  local dirs=()
  if command -v cygpath >/dev/null 2>&1; then
    dirs+=("$(cygpath -u "${LOCALAPPDATA:-}")/Programs" "$(cygpath -u "${PROGRAMFILES:-}")" "/c/Program Files (x86)")
  fi
  for user in /mnt/c/Users/*; do dirs+=("$user/AppData/Local/Programs"); done
  dirs+=("/mnt/c/Program Files (x86)" "/mnt/c/Program Files")
  for dir in "${dirs[@]}"; do
    if [[ -x "$dir/Inno Setup 6/ISCC.exe" ]]; then
      echo "$dir/Inno Setup 6/ISCC.exe"
      return
    fi
  done
}

win_path() {
  if command -v wslpath >/dev/null 2>&1; then
    wslpath -w "$1"
  elif command -v cygpath >/dev/null 2>&1; then
    cygpath -w "$1"
  else
    echo "$1"
  fi
}

echo "▶ build installer, version $APP_VERSION"
SETUP="build/bin/Frost-Flashcard-Setup-$APP_VERSION.exe"
rm -f "$SETUP"
ISCC="$(find_iscc || true)"
if [[ -n "$ISCC" ]]; then
  "$ISCC" /Q "/DAppVersion=$APP_VERSION" "$(win_path build/windows/installer.iss)"
elif command -v docker >/dev/null 2>&1; then
  if [[ "$(uname -m)" == "arm64" || "$(uname -m)" == "aarch64" ]] && [[ "${ALLOW_EMULATED_ISCC:-}" != "1" ]]; then
    echo "Refusing to build the installer with Docker on $(uname -m): Wine under emulation writes a corrupt setup file." >&2
    echo "Build it on Windows, or push a tag so .github/workflows/release.yml builds it." >&2
    exit 1
  fi
  docker run --rm -v "$PWD:/work" -w /work/build/windows amake/innosetup \
    /Q "/DAppVersion=$APP_VERSION" installer.iss
else
  echo "Inno Setup not found. Install it with:  winget install JRSoftware.InnoSetup" >&2
  echo "or install Docker to use the amake/innosetup image." >&2
  exit 1
fi

[[ -f "$SETUP" ]] || { echo "Inno Setup did not write $SETUP" >&2; exit 1; }

setup_size=$(wc -c <"$SETUP")
exe_size=$(wc -c <build/bin/frost-flashcard.exe)
if (( setup_size * 5 < exe_size )); then
  echo "$SETUP is only $setup_size bytes for a $exe_size byte exe, so it is truncated" >&2
  exit 1
fi

echo "✅ done: $SETUP"
