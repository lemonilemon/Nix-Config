"""SpaceNix boot wallpaper: mocha pixel-art space scene, 480x270 @4x -> 1920x1080.

Deterministic (seed 387, the generation current when it was designed) so the
derivation reproduces byte-identical art. Palette is Catppuccin Mocha only.
Composition keeps the middle third calm: Limine's terminal panel floats there.

Usage: wallpaper.py OUT.png FONT.ttf -- generates, stamps the antialiased
SpaceNix wordmark (Limine cannot render TTFs, so the typography lives in the
wallpaper), then re-parses the PNG header and fails the build unless it is
exactly what Limine's decoder is promised (1920x1080, 8-bit truecolor). The
validator is proven against a known-bad palettized PNG first; a check that
cannot fail is not a check.
"""

import math
import random
import struct
import sys

from PIL import Image, ImageDraw, ImageFont

W, H, SCALE = 480, 270, 4

CRUST = (17, 17, 27)
DEEP = (9, 9, 16)
BASE = (30, 30, 46)
SURFACE0 = (49, 50, 68)
SURFACE1 = (69, 71, 90)
OVERLAY0 = (108, 112, 134)
SUBTEXT0 = (166, 173, 200)
TEXT = (205, 214, 244)
LAVENDER = (180, 190, 254)
MAUVE = (203, 166, 247)
PEACH = (250, 179, 135)


def build() -> Image.Image:
    rng = random.Random(387)
    img = Image.new("RGB", (W, H))
    px = img.load()

    # --- sky: banded vertical gradient, dithered at the seams (pixel-art idiom)
    bands = [DEEP, CRUST, (22, 22, 34), (26, 26, 40), BASE]
    for y in range(H):
        t = y / H * (len(bands) - 1)
        idx = int(t)
        frac = t - idx
        for x in range(W):
            use_next = idx + 1 < len(bands) and rng.random() < frac
            px[x, y] = bands[idx + 1] if use_next else bands[idx]

    # --- stars: sparse, brightness-weighted; a few lavender twinkle crosses
    for _ in range(170):
        x, y = rng.randrange(W), rng.randrange(H)
        px[x, y] = rng.choices([SURFACE1, OVERLAY0, SUBTEXT0, TEXT], [5, 4, 2, 1])[0]
    for _ in range(7):
        x, y = rng.randrange(4, W - 4), rng.randrange(4, H - 4)
        px[x, y] = LAVENDER
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            px[x + dx, y + dy] = OVERLAY0

    # --- shooting star, upper left, clear of the terminal panel
    for i in range(14):
        x, y = 52 + i, 34 + i // 2
        px[x, y] = TEXT if i < 4 else (SUBTEXT0 if i < 9 else OVERLAY0)

    # --- horizon: a large dark body rising from the bottom-right corner, its
    # sunward limb caught by a thin rim light so it reads as a planet
    hx, hy, hr = 434.0, 300.0, 108.0
    for y in range(H):
        for x in range(W):
            dx, dy = (x - hx) / hr, (y - hy) / hr
            if dx * dx + dy * dy > 1.0:
                continue
            px[x, y] = SURFACE0 if rng.random() < 0.08 else BASE
    for deg in range(0, 3600):
        a = math.radians(deg / 10)
        for rim, color in ((1.0, OVERLAY0), (2.5, SURFACE1)):
            x = int(hx + math.cos(a) * (hr - rim))
            y = int(hy + math.sin(a) * (hr - rim))
            if 0 <= x < W and 0 <= y < H and math.sin(a) < -0.25:
                px[x, y] = color

    # --- ringed planet: upper-right sky, fully visible, light from upper-left
    cx, cy, r = 392.0, 64.0, 26.0
    lx, ly = -0.5, -0.87
    for y in range(int(cy - r) - 1, int(cy + r) + 2):
        for x in range(int(cx - r) - 1, int(cx + r) + 2):
            dx, dy = (x - cx) / r, (y - cy) / r
            if dx * dx + dy * dy > 1.0:
                continue
            lam = dx * lx + dy * ly  # +1 lit limb .. -1 dark limb
            if lam > 0.45:
                color = MAUVE
            elif lam > 0.05:
                color = SURFACE1
            elif lam > -0.35:
                color = SURFACE0
            else:
                color = BASE
            # dither the ramp seams so bands read as pixel art, not vector
            if rng.random() < 0.14:
                color = SURFACE1 if color == MAUVE else SURFACE0 if color == SURFACE1 else color
            px[x, y] = color
    for deg in range(0, 3600):
        a = math.radians(deg / 10)
        x = int(cx + math.cos(a) * (r - 1))
        y = int(cy + math.sin(a) * (r - 1))
        if math.cos(a) * lx + math.sin(a) * ly > 0.7:
            px[x, y] = LAVENDER

    # --- ring: peach in front (near half), dim behind, hidden inside the planet
    ring_rx, ring_ry, tilt = 46.0, 11.0, math.radians(-20)
    for step in range(0, 7200):
        a = math.radians(step / 20)
        ex, ey = math.cos(a) * ring_rx, math.sin(a) * ring_ry
        x = cx + ex * math.cos(tilt) - ey * math.sin(tilt)
        y = cy + ex * math.sin(tilt) + ey * math.cos(tilt)
        xi, yi = int(x), int(y)
        if not (0 <= xi < W and 0 <= yi < H):
            continue
        dx, dy = (x - cx) / r, (y - cy) / r
        inside = dx * dx + dy * dy < 1.0
        behind = math.sin(a) < 0  # far half of the ring
        if inside and behind:
            continue  # hidden by the planet
        px[xi, yi] = PEACH if not behind else SURFACE1

    # small companion moon
    mx, my, mr = 336, 176, 4
    for y in range(my - mr, my + mr + 1):
        for x in range(mx - mr, mx + mr + 1):
            if (x - mx) ** 2 + (y - my) ** 2 <= mr * mr:
                px[x, y] = LAVENDER if x - mx < 0 and y - my < 0 else SURFACE1

    return img.resize((W * SCALE, H * SCALE), Image.NEAREST)


def stamp_wordmark(img, font_path):
    """SpaceNix in JetBrains Mono SemiBold, centered in the band above the
    terminal panel (term_margin 320 leaves the top 320px to the wallpaper)."""
    font = ImageFont.truetype(font_path, 88)
    draw = ImageDraw.Draw(img)
    space_w = draw.textlength("Space", font=font)
    nix_w = draw.textlength("Nix", font=font)
    x = (img.width - space_w - nix_w) / 2
    draw.text((x, 160), "Space", font=font, fill=TEXT, anchor="lm")
    draw.text((x + space_w, 160), "Nix", font=font, fill=MAUVE, anchor="lm")


def validate(path: str) -> None:
    """Re-parse the produced PNG's IHDR; exit non-zero on any violation.

    Limine reads PNG itself, and bootloader decoders reject quietly -- the
    first GRUB theme shipped broken because ImageMagick silently optimised
    to palette PNGs its decoder refused. So the checks are on the BYTES,
    not on what PIL believes it saved.
    """
    with open(path, "rb") as f:
        header = f.read(33)
    if header[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"{path}: not a PNG")
    if header[12:16] != b"IHDR":
        raise SystemExit(f"{path}: first chunk is not IHDR")
    width, height = struct.unpack(">II", header[16:24])
    bit_depth, color_type = header[24], header[25]
    if (width, height) != (W * SCALE, H * SCALE):
        raise SystemExit(f"{path}: {width}x{height}, want {W * SCALE}x{H * SCALE}")
    if bit_depth != 8:
        raise SystemExit(f"{path}: bit depth {bit_depth}, want 8")
    if color_type != 2:
        raise SystemExit(f"{path}: color type {color_type}, want 2 (truecolor)")


def main() -> None:
    out, font_path = sys.argv[1], sys.argv[2]

    # Prove the validator can fail: a palettized PNG (color type 3), the
    # exact shape of the original GRUB failure.
    bad = out + ".known-bad.png"
    Image.new("P", (W * SCALE, H * SCALE)).save(bad)
    try:
        validate(bad)
    except SystemExit:
        pass
    else:
        raise SystemExit("validator accepted a palettized PNG; it cannot fail")

    art = build()
    stamp_wordmark(art, font_path)
    art.save(out, format="PNG", optimize=True)
    validate(out)


if __name__ == "__main__":
    main()
