import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import popups  # noqa: E402


class ParseOpenWindowsTests(unittest.TestCase):
    def test_parses_ids_before_colon(self):
        text = "bar-eDP-1: bar\nvolume_popup: volume_popup\npopup_backdrop: popup_backdrop\n"
        self.assertEqual(
            popups.parse_open_windows(text),
            {"bar-eDP-1", "volume_popup", "popup_backdrop"},
        )

    def test_ignores_blank_and_malformed_lines(self):
        self.assertEqual(popups.parse_open_windows("\n   \ngarbage\n"), set())


class PopupCallsTests(unittest.TestCase):
    def test_toggle_open_closes_all_then_opens_backdrop_then_popup(self):
        calls = popups.popup_eww_calls("toggle", ["volume_popup", "eDP-1"], open_ids=set())
        self.assertEqual(
            calls,
            [
                ["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW],
                ["open", "popup_backdrop", "--screen", "eDP-1"],
                ["open", "volume_popup", "--screen", "eDP-1"],
            ],
        )

    def test_toggle_when_already_open_only_closes(self):
        calls = popups.popup_eww_calls(
            "toggle", ["volume_popup", "eDP-1"], open_ids={"volume_popup"}
        )
        self.assertEqual(calls, [["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW]])

    def test_close_closes_all(self):
        calls = popups.popup_eww_calls("close", [], open_ids=set())
        self.assertEqual(calls, [["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW]])

    def test_unknown_window_rejected(self):
        with self.assertRaises(ValueError):
            popups.popup_eww_calls("toggle", ["nope_popup", "eDP-1"], open_ids=set())

    def test_toggle_requires_two_args(self):
        with self.assertRaises(ValueError):
            popups.popup_eww_calls("toggle", ["volume_popup"], open_ids=set())


class RunPopupTests(unittest.TestCase):
    def test_run_popup_toggle_invokes_eww_in_order(self):
        seen = []
        rc = popups.run_popup(
            ["toggle", "volume_popup", "HDMI-A-1"],
            active_windows_fn=lambda: "bar-eDP-1: bar\n",
            eww_fn=lambda args: seen.append(args),
        )
        self.assertEqual(rc, 0)
        self.assertEqual(
            seen,
            [
                ["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW],
                ["open", "popup_backdrop", "--screen", "HDMI-A-1"],
                ["open", "volume_popup", "--screen", "HDMI-A-1"],
            ],
        )


class FocusedMonitorTests(unittest.TestCase):
    def test_picks_focused_monitor(self):
        text = '[{"name": "eDP-1", "focused": false}, {"name": "HDMI-A-1", "focused": true}]'
        self.assertEqual(popups.focused_monitor_from_json(text), "HDMI-A-1")

    def test_falls_back_to_zero(self):
        self.assertEqual(popups.focused_monitor_from_json(""), "0")
        self.assertEqual(popups.focused_monitor_from_json("[]"), "0")
        self.assertEqual(popups.focused_monitor_from_json('[{"name": "eDP-1"}]'), "0")


class SingleArgToggleTests(unittest.TestCase):
    def test_toggle_without_screen_uses_focused_monitor(self):
        calls = []
        rc = popups.run_popup(
            ["toggle", "volume_popup"],
            active_windows_fn=lambda: "",
            eww_fn=calls.append,
            monitor_fn=lambda: "HDMI-A-1",
        )
        self.assertEqual(rc, 0)
        self.assertIn(["open", "volume_popup", "--screen", "HDMI-A-1"], calls)


if __name__ == "__main__":
    unittest.main()
