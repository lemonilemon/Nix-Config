"""The safety rails in __init__.py, tested.

A rail nobody exercises is a convention with extra steps. These assert that the
exact failure which motivated them -- a module under test holding an unpatched
`subprocess` and reaching the developer's live session -- now raises instead of
running, and that no test can address the running daemon's runtime files.
"""

import os
import subprocess
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from tests.eww_bar_backend import (  # noqa: E402
    SubprocessEscape,
    allow_real_subprocess,
)


class SubprocessRailTests(unittest.TestCase):
    def test_the_commands_that_actually_escaped_now_raise(self):
        escaped = [
            ["nmcli", "radio", "wifi", "off"],
            ["bluetoothctl", "power", "off"],
            ["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"],
            ["playerctl", "next"],
            ["pactl", "set-default-sink", "x"],
        ]
        for argv in escaped:
            with self.subTest(argv=argv):
                with self.assertRaises(SubprocessEscape) as caught:
                    subprocess.run(argv, check=False)
                self.assertIn(argv[0], str(caught.exception))

    def test_the_guard_survives_a_broad_except(self):
        """The code under test swallows Exceptions on purpose.

        wallpaper.set_wallpaper wraps its subprocess.run in
        `except Exception: pass`, and serve_control_connection turns any
        Exception into an error reply. A guard those handlers can catch is a
        guard that reports nothing.
        """
        with self.assertRaises(SubprocessEscape):
            try:
                subprocess.run(["nmcli", "radio", "wifi", "off"], check=False)
            except Exception:  # noqa: BLE001 -- exactly what the daemon does
                self.fail("a broad except swallowed the subprocess guard")

    def test_popen_is_guarded_too(self):
        # watchers uses Popen rather than run, so guarding only run would leave
        # the entire watcher layer -- the next thing to be ported -- uncovered.
        with self.assertRaises(SubprocessEscape):
            subprocess.Popen(["hyprctl", "monitors"])

    def test_a_bare_string_command_is_guarded(self):
        with self.assertRaises(SubprocessEscape):
            subprocess.run("nmcli", check=False)

    def test_the_go_toolchain_is_the_one_exception(self):
        from tests.eww_bar_backend import _check

        _check(["go", "run", "./x"])  # must not raise
        _check(["/nix/store/whatever-go-1.26/bin/go", "version"])
        with self.assertRaises(SubprocessEscape):
            _check(["gomad", "run"])  # a prefix match would let this through

    def test_the_escape_hatch_works_and_is_scoped(self):
        from tests.eww_bar_backend import _check

        with allow_real_subprocess():
            _check(["nmcli", "radio", "wifi"])
        with self.assertRaises(SubprocessEscape):
            _check(["nmcli", "radio", "wifi"])


class RuntimeDirRailTests(unittest.TestCase):
    def test_the_runtime_dir_is_not_the_session_one(self):
        runtime = Path(os.environ["XDG_RUNTIME_DIR"])
        self.assertTrue(runtime.name.startswith("eww-tests-runtime-"), runtime)
        # The two files the live daemon owns. control_server unlinks the socket
        # before binding, so a test that reached this path would take over the
        # bar's control channel rather than merely reading it.
        self.assertFalse((runtime / "eww-backend.sock").exists())
        self.assertFalse((runtime / "eww-backend.pid").exists())

    def test_the_daemon_paths_resolve_inside_it(self):
        from eww_bar_backend import paths

        runtime = os.environ["XDG_RUNTIME_DIR"]
        for path in (paths.control_socket_path(), paths.backend_pidfile_path(),
                     paths.display_mode_path()):
            self.assertTrue(str(path).startswith(runtime), path)


if __name__ == "__main__":
    unittest.main()
