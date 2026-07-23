import hashlib
import subprocess
from pathlib import Path

from .common import run_text, truncate_text

WALLPAPER_DEFAULT = {"current": "", "count": 0, "rows": []}

WALLPAPER_DIR = Path("~/Pictures/wallpapers").expanduser()
THUMB_DIR = Path("~/.cache/eww-bar/wallpaper-thumbs").expanduser()
# awww-decodable only — notably NOT .jxl.
EXTENSIONS = {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff"}
GRID_COLUMNS = 3
NAME_MAX = 22
# Calibration values; grow-from-top-right matches where the picker popup sits.
AWWW_TRANSITION = [
    "--transition-type", "grow",
    "--transition-pos", "top-right",
    "--transition-duration", "0.8",
    "--transition-fps", "60",
]


def scan_wallpaper_files(directory=None):
    directory = Path(directory) if directory else WALLPAPER_DIR
    try:
        entries = sorted(directory.iterdir())
    except Exception:
        return []
    return [
        entry for entry in entries
        if entry.is_file() and entry.suffix.lower() in EXTENSIONS
    ]


def parse_awww_query(text):
    # `awww query` prints one line per output, e.g.
    # `eDP-1: 1920x1200, scale: 2, currently displaying: image: /path/img.png`.
    # Both monitors mirror, so the first image path wins.
    for line in text.splitlines():
        if "image: " in line:
            return line.split("image: ", 1)[1].strip()
    return ""


def thumb_cache_path(path, stat=None):
    stat = stat or path.stat()
    digest = hashlib.sha1(
        f"{path}:{stat.st_mtime_ns}:{stat.st_size}".encode()
    ).hexdigest()
    return THUMB_DIR / f"{digest}.png"


def ensure_thumbnail(path):
    try:
        thumb = thumb_cache_path(path)
    except Exception:
        return ""
    if thumb.exists():
        return str(thumb)
    try:
        THUMB_DIR.mkdir(parents=True, exist_ok=True)
        subprocess.run(
            [
                "magick", f"{path}[0]",
                "-thumbnail", "320x200^", "-gravity", "center", "-extent", "320x200",
                str(thumb),
            ],
            stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            check=False, timeout=15,
        )
    except Exception:
        return ""
    return str(thumb) if thumb.exists() else ""


def wallpaper_items(files, current, thumb_fn=ensure_thumbnail):
    return [
        {
            "name": truncate_text(path.stem, NAME_MAX),
            "path": str(path),
            "thumb": thumb_fn(path),
            "animated": "true" if path.suffix.lower() == ".gif" else "false",
            "active": "true" if str(path) == current else "false",
        }
        for path in files
    ]


def rows_from_items(items, columns=GRID_COLUMNS):
    return [items[i:i + columns] for i in range(0, len(items), columns)]


def wallpaper_state():
    current = parse_awww_query(run_text(["awww", "query"]))
    items = wallpaper_items(scan_wallpaper_files(), current)
    return {"current": current, "count": len(items), "rows": rows_from_items(items)}


def set_wallpaper(path):
    if not path:
        raise ValueError("wallpaper set requires a path")
    subprocess.run(
        ["awww", "img", path, *AWWW_TRANSITION],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        check=False,
    )
    return wallpaper_state()
