import os
import sys
import unittest
import unittest.mock
from pathlib import Path
from tempfile import TemporaryDirectory

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, common  # noqa: E402


def _write_bat(root, **files):
    bat = Path(root) / "BAT0"
    bat.mkdir(parents=True)
    for name, value in files.items():
        (bat / name).write_text(str(value))


class BatteryStateTests(unittest.TestCase):
    def test_charge_units_health_and_power(self):
        with TemporaryDirectory() as root:
            _write_bat(
                root,
                capacity="80",
                status="Discharging",
                charge_now="2400000",
                current_now="1000000",
                charge_full="3000000",
                charge_full_design="3077000",
                voltage_now="12000000",
            )
            state = collectors.battery_state(root=Path(root))
        self.assertEqual(state["status"], "Discharging")
        self.assertEqual(state["health"], "97%")          # 3000000/3077000
        self.assertEqual(state["power"], "12.0 W")          # 1.0 A * 12.0 V
        self.assertEqual(state["capacity"], 80)
        self.assertNotEqual(state["time"], "")

    def test_full_reports_zero_draw(self):
        with TemporaryDirectory() as root:
            _write_bat(
                root,
                capacity="100",
                status="Full",
                charge_now="3077000",
                current_now="0",
                charge_full="3077000",
                charge_full_design="3077000",
                voltage_now="12716000",
            )
            state = collectors.battery_state(root=Path(root))
        self.assertEqual(state["status"], "Full")
        self.assertEqual(state["health"], "100%")
        self.assertEqual(state["power"], "—")


class BatteryModuleDisabledTests(unittest.TestCase):
    def test_disabled_module_skips_collection(self):
        with unittest.mock.patch.dict("os.environ", {"EWW_BAR_BATTERY": "0"}, clear=False):
            state = collectors.battery_state()
        self.assertEqual(state["status"], "Unknown")

    def test_enabled_by_default(self):
        with unittest.mock.patch.dict("os.environ", {}, clear=False):
            os.environ.pop("EWW_BAR_BATTERY", None)
            self.assertTrue(common.module_enabled("BATTERY"))


if __name__ == "__main__":
    unittest.main()
