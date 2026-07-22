import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class NetworkRadioTests(unittest.TestCase):
    def test_enabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="enabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "true")

    def test_disabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="disabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "false")

    def test_state_includes_wifi_enabled(self):
        with unittest.mock.patch.object(
            collectors, "network_connection_state", return_value={"text": "x", "class": "wifi"}
        ):
            with unittest.mock.patch.object(collectors, "network_radio_enabled", return_value="true"):
                state = collectors.network_state()
        self.assertEqual(state["wifi_enabled"], "true")
        self.assertEqual(state["class"], "wifi")


class WifiToggleTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_toggle_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="true"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "off"]])

    def test_toggle_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="false"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "on"]])

    def test_arg_parsing(self):
        self.assertEqual(
            control.control_payload_from_args(["network", "wifi-toggle"]),
            {"command": "network", "action": "wifi-toggle"},
        )


if __name__ == "__main__":
    unittest.main()
