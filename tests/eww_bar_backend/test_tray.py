import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors  # noqa: E402


class TrayCountTests(unittest.TestCase):
    def test_parses_count_after_as_tag(self):
        text = 'as 6 ":1.2/StatusNotifierItem" ":1.42/StatusNotifierItem"\n'
        self.assertEqual(collectors.tray_count_from_text(text), 6)

    def test_zero_items(self):
        self.assertEqual(collectors.tray_count_from_text("as 0\n"), 0)

    def test_empty_or_garbage(self):
        self.assertEqual(collectors.tray_count_from_text(""), 0)
        self.assertEqual(collectors.tray_count_from_text("garbage output"), 0)
        self.assertEqual(collectors.tray_count_from_text("as notanumber tail"), 0)


if __name__ == "__main__":
    unittest.main()
