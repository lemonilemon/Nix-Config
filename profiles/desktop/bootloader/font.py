"""Convert a PSF console font to Limine's raw terminal font format.

Limine wants a code page 437 set: 256 consecutive glyph bitmaps, one byte per
row, 16 rows per glyph (term_font_size 8x16). PSF fonts index glyphs by their
own order plus a unicode table, so this rebuilds the CP437 order explicitly.

Usage: font.py IN.psf[u][.gz] OUT -- converts, then re-reads OUT and fails the
build unless it is exactly 256 * 16 bytes with the glyphs the menu draws
present. The required-glyph check is proven against a table stripped of the
arrow glyphs first; a check that cannot fail is not a check.
"""

import gzip
import struct
import sys

HEIGHT = 16

# CP437's graphical low range; python's cp437 codec maps 0x00-0x1F to control
# characters, but the font slots hold these symbols.
CP437_GRAPHIC = {
    0x01: "☺", 0x02: "☻", 0x03: "♥", 0x04: "♦", 0x05: "♣", 0x06: "♠",
    0x07: "•", 0x08: "◘", 0x09: "○", 0x0A: "◙", 0x0B: "♂", 0x0C: "♀",
    0x0D: "♪", 0x0E: "♫", 0x0F: "☼", 0x10: "►", 0x11: "◄", 0x12: "↕",
    0x13: "‼", 0x14: "¶", 0x15: "§", 0x16: "▬", 0x17: "↨", 0x18: "↑",
    0x19: "↓", 0x1A: "→", 0x1B: "←", 0x1C: "∟", 0x1D: "↔", 0x1E: "▲",
    0x1F: "▼", 0x7F: "⌂",
}

# Present in the menu chrome (tree markers, help line); a font missing any of
# these would boot with holes in the interface.
REQUIRED = set(chr(c) for c in range(0x20, 0x7F)) | {"↑", "↓", "▲", "▼"}


def read_psf(path):
    data = gzip.open(path).read() if path.endswith(".gz") else open(path, "rb").read()
    unicode_map = {}
    if data[:2] == b"\x36\x04":  # PSF1
        mode, charsize = data[2], data[3]
        count = 512 if mode & 1 else 256
        height, offset = charsize, 4
        glyphs = [data[offset + i * charsize : offset + (i + 1) * charsize] for i in range(count)]
        if mode & 2:
            pos = offset + count * charsize
            for idx in range(count):
                while True:
                    (u,) = struct.unpack_from("<H", data, pos)
                    pos += 2
                    if u == 0xFFFF:
                        break
                    if u != 0xFFFE:
                        unicode_map.setdefault(chr(u), idx)
    elif data[:4] == b"\x72\xb5\x4a\x86":  # PSF2
        _, headersize, flags, count, charsize, height, width = struct.unpack_from("<6I I", data, 4)
        if width != 8:
            raise SystemExit(f"{path}: width {width}, Limine fonts must be 8 wide")
        glyphs = [data[headersize + i * charsize : headersize + (i + 1) * charsize] for i in range(count)]
        if flags & 1:
            pos = headersize + count * charsize
            for idx in range(count):
                start = pos
                while data[pos] != 0xFF:
                    pos += 1
                for ch in data[start:pos].split(b"\xfe")[0].decode("utf-8", "ignore"):
                    unicode_map.setdefault(ch, idx)
                pos += 1
    else:
        raise SystemExit(f"{path}: not a PSF font")
    if height != HEIGHT:
        raise SystemExit(f"{path}: glyph height {height}, want {HEIGHT}")
    if not unicode_map:
        raise SystemExit(f"{path}: no unicode table; cannot build CP437 order safely")
    return glyphs, unicode_map


def cp437_char(i):
    if i in CP437_GRAPHIC:
        return CP437_GRAPHIC[i]
    if i == 0x00:
        return None  # NUL slot stays blank
    return bytes([i]).decode("cp437")


def assemble(glyphs, unicode_map):
    out, blank_filled = bytearray(), []
    for i in range(256):
        ch = cp437_char(i)
        idx = unicode_map.get(ch) if ch is not None else None
        if idx is None:
            if ch in REQUIRED:
                raise SystemExit(f"font is missing required menu glyph {ch!r} (CP437 0x{i:02X})")
            blank_filled.append(i)
            out += bytes(HEIGHT)
        else:
            glyph = glyphs[idx][:HEIGHT]
            out += glyph + bytes(HEIGHT - len(glyph))
    return bytes(out), blank_filled


def main():
    src, dst = sys.argv[1], sys.argv[2]
    glyphs, unicode_map = read_psf(src)

    # Prove the required-glyph check can fail before trusting it.
    crippled = {ch: idx for ch, idx in unicode_map.items() if ch != "↑"}
    try:
        assemble(glyphs, crippled)
    except SystemExit:
        pass
    else:
        raise SystemExit("validator accepted a font without ↑; it cannot fail")

    blob, blank_filled = assemble(glyphs, unicode_map)
    with open(dst, "wb") as f:
        f.write(blob)

    written = open(dst, "rb").read()
    if len(written) != 256 * HEIGHT:
        raise SystemExit(f"{dst}: {len(written)} bytes, want {256 * HEIGHT}")
    print(f"{dst}: 256 glyphs x {HEIGHT} rows; blank-filled slots: {len(blank_filled)}")


if __name__ == "__main__":
    main()
