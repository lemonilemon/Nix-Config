import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class BluetoothStateTests(unittest.TestCase):
    def test_powered_and_devices_parsed(self):
        controller = "Controller AA:BB Alias: MyBT\n\tPowered: yes\n\tAlias: MyBT\n"
        devices = "Device 80:99:E7 WF-1000XM5\n"
        info = {"80:99:E7": "\tAlias: WF-1000XM5\n\tBattery Percentage: 0x50 (80)\n"}
        state = collectors.bluetooth_state_from_text(controller, devices, info)
        self.assertEqual(state["powered"], "true")
        self.assertEqual(len(state["devices"]), 1)
        self.assertEqual(state["devices"][0]["mac"], "80:99:E7")
        self.assertEqual(state["devices"][0]["name"], "WF-1000XM5")
        self.assertEqual(state["devices"][0]["battery"], "80%")
        self.assertEqual(state["devices"][0]["connected"], "true")

    def test_powered_off_no_devices(self):
        controller = "Controller AA:BB\n\tPowered: no\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["powered"], "false")
        self.assertEqual(state["devices"], [])
        self.assertEqual(state["class"], "off")

    def test_powered_off_uses_the_off_class(self):
        controller = "Controller AA:BB\n\tPowered: no\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["class"], "off")

    def test_powered_on_no_devices_has_no_off_class(self):
        controller = "Controller AA:BB\n\tAlias: MyBT\n\tPowered: yes\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["class"], "")
        self.assertEqual(state["powered"], "true")

    def test_class_is_always_a_known_style(self):
        # eww.scss defines .bluetooth and .bluetooth.off only; "connected" is
        # deliberately unstyled and falls through to .bluetooth.
        known = {"", "off", "connected"}
        for powered in ("yes", "no", "whatever", ""):
            controller = f"Controller AA:BB\n\tPowered: {powered}\n"
            self.assertIn(
                collectors.bluetooth_state_from_text(controller, "", {})["class"], known
            )
            self.assertIn(
                collectors.bluetooth_state_from_text(
                    controller, "Device 80:99:E7 Buds\n", {}
                )["class"],
                known,
            )


class BluetoothControlTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_power_toggle_turns_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", side_effect=[{"powered": "true"}, {}]):
                control.toggle_bluetooth_power()
        self.assertEqual(calls, [["bluetoothctl", "power", "off"]])

    def test_power_toggle_turns_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", side_effect=[{"powered": "false"}, {}]):
                control.toggle_bluetooth_power()
        self.assertEqual(calls, [["bluetoothctl", "power", "on"]])

    def test_disconnect(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", return_value={}):
                control.disconnect_bluetooth("80:99:E7")
        self.assertEqual(calls, [["bluetoothctl", "disconnect", "80:99:E7"]])


if __name__ == "__main__":
    unittest.main()
