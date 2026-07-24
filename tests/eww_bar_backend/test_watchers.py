import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend.watchers import bar_window_command, monitor_event  # noqa: E402


class MonitorEventTests(unittest.TestCase):
    def test_added_parses_name(self):
        self.assertEqual(monitor_event("monitoradded>>HDMI-A-1"), ("added", "HDMI-A-1"))

    def test_removed_parses_name(self):
        self.assertEqual(monitor_event("monitorremoved>>HDMI-A-1"), ("removed", "HDMI-A-1"))

    def test_v2_events_ignored_to_avoid_double_fire(self):
        self.assertIsNone(monitor_event("monitoraddedv2>>1,HDMI-A-1,Dell U2723QE"))
        self.assertIsNone(monitor_event("monitorremovedv2>>1,HDMI-A-1,Dell U2723QE"))

    def test_unrelated_and_empty_events_ignored(self):
        self.assertIsNone(monitor_event("workspace>>3"))
        self.assertIsNone(monitor_event("monitoradded>>"))
        self.assertIsNone(monitor_event(""))


class BarWindowCommandTests(unittest.TestCase):
    def test_added_matches_startup_script_shape(self):
        self.assertEqual(
            bar_window_command("added", "HDMI-A-1"),
            [
                "eww", "open", "bar",
                "--id", "bar-HDMI-A-1",
                "--screen", "HDMI-A-1",
                "--arg", "output=HDMI-A-1",
            ],
        )

    def test_removed_closes_bar_window(self):
        self.assertEqual(
            bar_window_command("removed", "HDMI-A-1"),
            ["eww", "close", "bar-HDMI-A-1"],
        )


if __name__ == "__main__":
    unittest.main()
