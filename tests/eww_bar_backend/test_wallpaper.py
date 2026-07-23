import sys
import tempfile
import unittest
from collections import namedtuple
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import wallpaper  # noqa: E402

FakeStat = namedtuple("FakeStat", "st_mtime_ns st_size")


class ScanTests(unittest.TestCase):
    def test_filters_supported_extensions_sorted(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ("b.png", "a.jpg", "anim.gif", "skip.jxl", "note.txt", "pic.webp"):
                (root / name).write_bytes(b"x")
            (root / "subdir").mkdir()
            files = wallpaper.scan_wallpaper_files(root)
            self.assertEqual(
                [f.name for f in files], ["a.jpg", "anim.gif", "b.png", "pic.webp"]
            )

    def test_missing_directory_yields_empty(self):
        self.assertEqual(wallpaper.scan_wallpaper_files("/nonexistent/nowhere"), [])


class QueryParseTests(unittest.TestCase):
    def test_parses_image_path(self):
        text = 'eDP-1: 1920x1200, scale: 2, currently displaying: image: /home/u/Pictures/wallpapers/pixel_sunset.png\n'
        self.assertEqual(
            wallpaper.parse_awww_query(text),
            "/home/u/Pictures/wallpapers/pixel_sunset.png",
        )

    def test_color_or_garbage_yields_empty(self):
        self.assertEqual(wallpaper.parse_awww_query("eDP-1: ... displaying: color: 000000"), "")
        self.assertEqual(wallpaper.parse_awww_query(""), "")


class ThumbCacheTests(unittest.TestCase):
    def test_stable_hash_from_path_mtime_size(self):
        path = Path("/x/y.png")
        first = wallpaper.thumb_cache_path(path, FakeStat(1, 2))
        second = wallpaper.thumb_cache_path(path, FakeStat(1, 2))
        changed = wallpaper.thumb_cache_path(path, FakeStat(9, 2))
        self.assertEqual(first, second)
        self.assertNotEqual(first, changed)
        self.assertTrue(str(first).endswith(".png"))


class ItemsAndRowsTests(unittest.TestCase):
    def test_items_flags(self):
        files = [Path("/w/sunset.png"), Path("/w/loop.gif")]
        items = wallpaper.wallpaper_items(files, "/w/loop.gif", thumb_fn=lambda p: f"/thumbs/{p.name}")
        self.assertEqual(items[0]["active"], "false")
        self.assertEqual(items[0]["animated"], "false")
        self.assertEqual(items[1]["active"], "true")
        self.assertEqual(items[1]["animated"], "true")
        self.assertEqual(items[0]["name"], "sunset")
        self.assertEqual(items[0]["thumb"], "/thumbs/sunset.png")

    def test_rows_chunking(self):
        items = list(range(7))
        rows = wallpaper.rows_from_items(items)
        self.assertEqual(rows, [[0, 1, 2], [3, 4, 5], [6]])
        self.assertEqual(wallpaper.rows_from_items([]), [])


class SetWallpaperTests(unittest.TestCase):
    def test_empty_path_raises(self):
        with self.assertRaises(ValueError):
            wallpaper.set_wallpaper("")


from eww_bar_backend import control  # noqa: E402


class WallpaperPayloadTests(unittest.TestCase):
    def test_payloads(self):
        self.assertEqual(
            control.control_payload_from_args(["wallpaper", "set", "/w/a.png"]),
            {"command": "wallpaper", "action": "set", "path": "/w/a.png"},
        )
        self.assertEqual(
            control.control_payload_from_args(["wallpaper", "rescan"]),
            {"command": "wallpaper", "action": "rescan"},
        )

    def test_wallpaper_without_action_is_usage_error(self):
        with self.assertRaises(ValueError):
            control.control_payload_from_args(["wallpaper"])


if __name__ == "__main__":
    unittest.main()
