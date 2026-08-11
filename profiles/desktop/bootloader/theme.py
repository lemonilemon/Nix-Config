"""Assemble the SpaceNix GRUB theme: pixmaps, icons, theme.txt.

Usage: theme.py OUTDIR BACKGROUND.png ITEM.pf2 SMALL.pf2

Everything GRUB will decode is validated against its actual decoder limits
before the build may succeed (the July lesson: grub-core's png.c rejects
palette PNGs silently, and a theme.txt font name that mismatches the pf2's
NAME section falls back to whatever font loaded first). Both failure classes
are designed out here: every PNG is re-parsed byte-level, and theme.txt is
GENERATED from the names the pf2 files actually report.
"""

import math
import shutil
import struct
import sys

from PIL import Image, ImageDraw

BASE = (30, 30, 46)
SURFACE0 = (49, 50, 68)
TEXT = "#cdd6f4"
MAUVE = (203, 166, 247)
LAVENDER = (180, 190, 254)
BLUE = (137, 180, 250)
OVERLAY0 = "#6c7086"

manifest = []


def validate_png(path, allowed_color_types=(2, 6)):
    """Re-parse the IHDR; grub-core/video/readers/png.c takes 8/16-bit depth
    and truecolor; palette output is the classic silent rejection."""
    with open(path, "rb") as f:
        header = f.read(33)
    if header[:8] != b"\x89PNG\r\n\x1a\n" or header[12:16] != b"IHDR":
        raise SystemExit(f"{path}: not a PNG")
    bit_depth, color_type = header[24], header[25]
    if bit_depth != 8:
        raise SystemExit(f"{path}: bit depth {bit_depth}, want 8")
    if color_type not in allowed_color_types:
        raise SystemExit(f"{path}: color type {color_type}, want {allowed_color_types}")


def save(img, outdir, name):
    path = f"{outdir}/{name}"
    img.save(path, format="PNG")
    validate_png(path)
    manifest.append(name)


def slice9(img, corner, outdir, prefix):
    s = img.size[0]
    boxes = {
        "nw": (0, 0, corner, corner),
        "n": (corner, 0, s - corner, corner),
        "ne": (s - corner, 0, s, corner),
        "w": (0, corner, corner, s - corner),
        "c": (corner, corner, s - corner, s - corner),
        "e": (s - corner, corner, s, s - corner),
        "sw": (0, s - corner, corner, s),
        "s": (corner, s - corner, s - corner, s),
        "se": (s - corner, s - corner, s, s),
    }
    for part, box in boxes.items():
        save(img.crop(box), outdir, f"{prefix}_{part}.png")


def rounded(size, radius, fill, outline=None, width=0):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    ImageDraw.Draw(img).rounded_rectangle(
        [0, 0, size - 1, size - 1], radius=radius, fill=fill,
        outline=outline, width=width,
    )
    return img


def icon(kind, size=36):
    s4 = size * 4
    tile = Image.new("RGBA", (s4, s4), (0, 0, 0, 0))
    d = ImageDraw.Draw(tile)
    c = s4 // 2
    if kind == "snowflake":
        r = s4 * 0.42
        for k in range(6):
            a = math.radians(k * 60 + 30)
            d.line([c, c, c + r * math.cos(a), c + r * math.sin(a)],
                   fill=LAVENDER + (255,), width=s4 // 12)
            bx, by = c + r * 0.6 * math.cos(a), c + r * 0.6 * math.sin(a)
            for da in (-38, 38):
                b = math.radians(k * 60 + 30 + da)
                d.line([bx, by, bx + s4 * 0.12 * math.cos(b), by + s4 * 0.12 * math.sin(b)],
                       fill=LAVENDER + (255,), width=s4 // 16)
    else:
        g, sq = s4 * 0.04, s4 * 0.36
        for dx in (0, 1):
            for dy in (0, 1):
                x0 = c - g - sq + dx * (sq + 2 * g)
                y0 = c - g - sq + dy * (sq + 2 * g)
                d.rectangle([x0, y0, x0 + sq, y0 + sq], fill=BLUE + (255,))
    return tile.resize((size, size), Image.LANCZOS)


def pf2_name(path):
    """The font name grub actually registers, read from the NAME section --
    theme.txt must use this string verbatim, so it is derived, never typed."""
    data = open(path, "rb").read()
    if data[:4] != b"FILE" or data[8:12] != b"PFF2":
        raise SystemExit(f"{path}: not a PFF2 font")
    pos = 12
    while pos + 8 <= len(data):
        tag = data[pos : pos + 4]
        (length,) = struct.unpack(">I", data[pos + 4 : pos + 8])
        if tag == b"DATA":
            break
        if tag == b"NAME":
            name = data[pos + 8 : pos + 8 + length].rstrip(b"\x00").decode()
            if not name:
                raise SystemExit(f"{path}: empty NAME section")
            return name
        pos += 8 + length
    raise SystemExit(f"{path}: no NAME section before DATA")


def main():
    outdir, background, item_pf2, small_pf2 = sys.argv[1:5]

    # Prove the PNG validator can fail: a palette PNG, the July failure shape.
    bad = outdir + "/known-bad.png"
    Image.new("P", (8, 8)).save(bad)
    try:
        validate_png(bad)
    except SystemExit:
        import os
        os.remove(bad)
    else:
        raise SystemExit("validator accepted a palette PNG; it cannot fail")

    shutil.copyfile(background, f"{outdir}/background.png")
    validate_png(f"{outdir}/background.png")
    manifest.append("background.png")

    # Panel: the translucent mocha card; selection: the mauve ring. A styled
    # box's slice sizes become the item's content insets, so the selection
    # slices are slim (10px) and mirrored by a fully transparent twin for
    # unselected items -- identical insets in both states is what keeps the
    # text from shifting on selection.
    slice9(rounded(96, 18, BASE + (200,)), 24, outdir, "panel")
    slice9(rounded(64, 10, MAUVE + (66,), outline=MAUVE + (255,), width=2),
           10, outdir, "select")
    slice9(Image.new("RGBA", (64, 64), (0, 0, 0, 0)), 10, outdir, "noselect")

    # Flat opaque center slice for the gfxterm window (see terminal-* below).
    save(Image.new("RGBA", (8, 8), BASE + (255,)), outdir, "term_c.png")

    import os
    os.makedirs(f"{outdir}/icons", exist_ok=True)
    for name, kind in (("nixos", "snowflake"), ("submenu", "snowflake"),
                       ("windows", "win"), ("os", "win")):
        path = f"{outdir}/icons/{name}.png"
        icon(kind).save(path, format="PNG")
        validate_png(path)
        manifest.append(f"icons/{name}.png")

    item_font = pf2_name(item_pf2)
    small_font = pf2_name(small_pf2)

    theme = f"""\
# SpaceNix GRUB theme -- generated by theme.py; fonts referenced by the names
# their pf2 files report, so a grub-mkfont naming change cannot desync this.
desktop-image: "background.png"
desktop-image-scale-method: "stretch"
desktop-color: "#1e1e2e"
terminal-font: "{small_font}"
# gfxmenu draws a built-in "GRUB Boot Menu" title over the wordmark unless
# the theme overrides it (review finding, confirmed in gfxmenu.mod).
title-text: ""

# The post-ENTER terminal: full screen, borderless, flat mocha -- without
# this it appears as a white-bordered popup window mid-screen.
terminal-box: "term_*.png"
terminal-border: 0
terminal-left: 0
terminal-top: 0
terminal-width: 100%
terminal-height: 100%

+ boot_menu {{
    left = 27%
    top = 39%
    width = 46%
    height = 32%
    item_font = "{item_font}"
    item_color = "{TEXT}"
    selected_item_color = "{TEXT}"
    item_height = 64
    item_spacing = 14
    item_padding = 8
    icon_width = 36
    icon_height = 36
    item_icon_space = 16
    menu_pixmap_style = "panel_*.png"
    item_pixmap_style = "noselect_*.png"
    selected_item_pixmap_style = "select_*.png"
    scrollbar = false
}}

+ progress_bar {{
    id = "__timeout__"
    left = 29%
    top = 65%
    width = 42%
    height = 8
    fg_color = "#cba6f7"
    bg_color = "#313244"
    border_color = "#313244"
}}

+ label {{
    id = "__timeout__"
    left = 27%
    top = 67%
    width = 46%
    align = "center"
    color = "{OVERLAY0}"
    font = "{small_font}"
    text = "booting the highlighted entry in %ds"
}}
"""
    with open(f"{outdir}/theme.txt", "w") as f:
        f.write(theme)

    # Every file theme.txt names must exist (pixmap globs expand per-slice;
    # term_* deliberately ships only its center slice -- absent slices are
    # documented to render empty).
    referenced = ["background.png", "term_c.png"]
    referenced += [f"panel_{p}.png" for p in "nw n ne w c e sw s se".split()]
    referenced += [f"select_{p}.png" for p in "nw n ne w c e sw s se".split()]
    referenced += [f"noselect_{p}.png" for p in "nw n ne w c e sw s se".split()]
    missing = [f for f in referenced if f not in manifest]
    if missing:
        raise SystemExit(f"theme.txt references missing files: {missing}")

    print(f"theme assembled: fonts {item_font!r} / {small_font!r}, "
          f"{len(manifest)} validated assets")


if __name__ == "__main__":
    main()
