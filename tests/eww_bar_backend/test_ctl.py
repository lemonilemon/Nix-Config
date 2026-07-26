import subprocess
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import ctl  # noqa: E402


class ControlPayloadTests(unittest.TestCase):
    def test_ping(self):
        self.assertEqual(ctl.control_payload_from_args(["ping"]), {"command": "ping"})

    def test_volume_set_carries_value(self):
        self.assertEqual(
            ctl.control_payload_from_args(["volume", "set", "40"]),
            {"command": "volume", "action": "set", "value": "40"},
        )

    def test_notif_clear_group_joins_multiword_app(self):
        self.assertEqual(
            ctl.control_payload_from_args(["notif", "clear-group", "My", "App"]),
            {"command": "notif", "action": "clear-group", "app": "My App"},
        )

    def test_idle_defaults_to_toggle(self):
        self.assertEqual(
            ctl.control_payload_from_args(["idle"]),
            {"command": "idle", "action": "toggle"},
        )

    def test_help_raises_usage(self):
        with self.assertRaises(ValueError):
            ctl.control_payload_from_args(["--help"])


class ImportGraphTests(unittest.TestCase):
    """The click path must not import the daemon.

    Run in a subprocess so this test is unaffected by modules the rest of the
    suite has already imported into this interpreter.
    """

    FORBIDDEN = (
        "urllib.request",
        "http.client",
        "email.parser",
        "subprocess",
        "eww_bar_backend.common",
    )

    def _modules_after_importing(self, module):
        code = (
            "import sys\n"
            f"sys.path.insert(0, {str(SCRIPTS_DIR)!r})\n"
            f"import {module}\n"
            "print('\\n'.join(sorted(sys.modules)))\n"
        )
        out = subprocess.run(
            [sys.executable, "-c", code], capture_output=True, text=True, check=True
        )
        return set(out.stdout.split())

    def test_ctl_does_not_pull_in_http(self):
        loaded = self._modules_after_importing("eww_bar_backend.ctl")
        for name in self.FORBIDDEN:
            self.assertNotIn(name, loaded)

    def test_ctl_does_not_pull_in_collectors(self):
        loaded = self._modules_after_importing("eww_bar_backend.ctl")
        self.assertNotIn("eww_bar_backend.collectors", loaded)
        self.assertNotIn("eww_bar_backend.app", loaded)

    def test_package_import_is_empty(self):
        loaded = self._modules_after_importing("eww_bar_backend")
        self.assertNotIn("eww_bar_backend.app", loaded)
        self.assertNotIn("eww_bar_backend.collectors", loaded)


if __name__ == "__main__":
    unittest.main()
