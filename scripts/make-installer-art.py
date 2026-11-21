#!/usr/bin/env python3
import math
import os
import struct
import zlib

SIZES = {"100": (596, 432), "150": (994, 720), "200": (1272, 922)}

BASE = (18, 18, 26)
STRENGTH = 0.38

ORBS = [
    ((139, 157, 255), 0.02, 0.05, 0.42),
    ((255, 158, 203), 1.00, 1.30, 0.48),
    ((127, 227, 212), 0.55, 0.62, 0.26),
]
ORB_WEIGHT = [1.0, 1.0, 0.6]


def render(width, height, base, strength):
    rows = []
    orbs = [
        (c, cx * width, cy * height, r * width, w)
        for (c, cx, cy, r), w in zip(ORBS, ORB_WEIGHT)
    ]
    for y in range(height):
        row = bytearray(b"\x00")
        for x in range(width):
            r, g, b = base
            for (oc, ox, oy, rad, weight) in orbs:
                d2 = ((x - ox) ** 2 + (y - oy) ** 2) / (rad * rad)
                a = strength * weight * math.exp(-d2 * 1.6)
                if a > 0.002:
                    r += (oc[0] - r) * a
                    g += (oc[1] - g) * a
                    b += (oc[2] - b) * a
            row += bytes((int(r + 0.5), int(g + 0.5), int(b + 0.5)))
        rows.append(bytes(row))
    return b"".join(rows)


def write_png(path, width, height, raw):
    def chunk(kind, data):
        body = kind + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

    ihdr = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n")
        f.write(chunk(b"IHDR", ihdr))
        f.write(chunk(b"IDAT", zlib.compress(raw, 9)))
        f.write(chunk(b"IEND", b""))


def main():
    out = os.path.join(os.path.dirname(__file__), "..", "build", "windows")
    for scale, (w, h) in SIZES.items():
        path = os.path.join(out, f"wizard-bg-{scale}.png")
        write_png(path, w, h, render(w, h, BASE, STRENGTH))
        print("wrote", os.path.normpath(path))


if __name__ == "__main__":
    main()
