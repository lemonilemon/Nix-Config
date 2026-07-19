import os
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import display, inhibitors  # noqa: E402


class InhibitorTests(unittest.TestCase):
    def test_set_idle_inhibited_starts_user_service(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        with patch.object(inhibitors.subprocess, "run", side_effect=fake_run):
            with patch.object(inhibitors, "service_active", return_value=True):
                self.assertEqual(inhibitors.set_idle_inhibited(True), "true")

        self.assertEqual(
            calls,
            [["systemctl", "--user", "start", "eww-hypridle-inhibit.service"]],
        )

    def test_set_idle_inhibited_stops_user_service(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        with patch.object(inhibitors.subprocess, "run", side_effect=fake_run):
            with patch.object(inhibitors, "service_active", return_value=False):
                self.assertEqual(inhibitors.set_idle_inhibited(False), "false")

        self.assertEqual(
            calls,
            [["systemctl", "--user", "stop", "eww-hypridle-inhibit.service"]],
        )


class DisplayModeTests(unittest.TestCase):
    def test_external_mode_requires_an_external_monitor(self):
        commands = []
        monitor_data = [{"name": "eDP-1", "disabled": False}]

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "monitor_state", return_value=monitor_data):
                    with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                        with self.assertRaisesRegex(ValueError, "external monitor"):
                            display.set_display_mode("external")

        self.assertEqual(commands, [])

    def test_external_mode_disables_internal_panel_and_keeps_inhibitors(self):
        commands = []
        monitor_data = [
            {"name": "eDP-1", "disabled": False},
            {"name": "HDMI-A-1", "disabled": False},
        ]

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "monitor_state", return_value=monitor_data):
                    with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                        with patch.object(display, "set_idle_inhibited", return_value="true"):
                            with patch.object(display, "set_lid_inhibited", return_value="true"):
                                with patch.object(display, "lid_inhibited_state", return_value="true"):
                                    result = display.set_display_mode("external")

        self.assertEqual(result["mode"], "external")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertIn(("dispatch", "dpms", "on"), commands)
        self.assertIn(("keyword", "monitor", "eDP-1,disable"), commands)
        # The external monitor must be powered on before the internal panel is
        # disabled, otherwise a DPMS-off external screen stays black.
        self.assertLess(
            commands.index(("dispatch", "dpms", "on")),
            commands.index(("keyword", "monitor", "eDP-1,disable")),
        )

    def test_headless_mode_turns_dpms_off(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="true"):
                        with patch.object(display, "set_lid_inhibited", return_value="true"):
                            with patch.object(display, "lid_inhibited_state", return_value="true"):
                                result = display.set_display_mode("headless")

        self.assertEqual(result["mode"], "headless")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertEqual(commands, [("dispatch", "dpms", "off")])

    def test_restore_turns_screens_on_without_clearing_mode(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                display.write_display_mode("headless")
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "lid_inhibited_state", return_value="true"):
                        result = display.set_display_mode("restore")

        self.assertEqual(result["mode"], "headless")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertEqual(commands, [("dispatch", "dpms", "on")])

    def test_normal_mode_restores_and_clears_inhibitors(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                display.write_display_mode("external")
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="false"):
                        with patch.object(display, "set_lid_inhibited", return_value="false"):
                            with patch.object(display, "lid_inhibited_state", return_value="false"):
                                result = display.set_display_mode("normal")

        self.assertEqual(result["mode"], "normal")
        self.assertEqual(result["lid_inhibited"], "false")
        self.assertEqual(commands, [("dispatch", "dpms", "on"), ("reload",)])

    def test_toggle_from_normal_enters_headless(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="true"):
                        with patch.object(display, "set_lid_inhibited", return_value="true"):
                            with patch.object(display, "lid_inhibited_state", return_value="true"):
                                result = display.set_display_mode("toggle")

        self.assertEqual(result["mode"], "headless")
        self.assertEqual(commands, [("dispatch", "dpms", "off")])

    def test_toggle_from_headless_returns_to_normal(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                display.write_display_mode("headless")
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="false"):
                        with patch.object(display, "set_lid_inhibited", return_value="false"):
                            with patch.object(display, "lid_inhibited_state", return_value="false"):
                                result = display.set_display_mode("toggle")

        self.assertEqual(result["mode"], "normal")
        self.assertEqual(commands, [("dispatch", "dpms", "on"), ("reload",)])


if __name__ == "__main__":
    unittest.main()
