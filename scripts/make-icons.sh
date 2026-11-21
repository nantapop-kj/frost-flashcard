#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

SRC=assets/icons/appicon.png
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

ICONSET="$TMP/icon.iconset"
mkdir -p "$ICONSET"
for size in 16 32 128 256 512; do
  sips -z $size $size "$SRC" --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
  sips -z $((size * 2)) $((size * 2)) "$SRC" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o build/darwin/icon.icns

for size in 16 24 32 48 64 128 256; do
  sips -z $size $size "$SRC" --out "$TMP/ico_$size.png" >/dev/null
done
python3 - "$TMP" build/windows/icon.ico <<'PY'
import struct, sys
src, out = sys.argv[1], sys.argv[2]
sizes = [16, 24, 32, 48, 64, 128, 256]
images = [open(f"{src}/ico_{s}.png", "rb").read() for s in sizes]
offset = 6 + 16 * len(images)
header = struct.pack("<HHH", 0, 1, len(images))
entries = b""
for size, data in zip(sizes, images):
    dim = 0 if size == 256 else size
    entries += struct.pack("<BBBBHHII", dim, dim, 0, 0, 1, 32, len(data), offset)
    offset += len(data)
with open(out, "wb") as f:
    f.write(header + entries + b"".join(images))
PY

echo "icons written to build/darwin/icon.icns and build/windows/icon.ico"
