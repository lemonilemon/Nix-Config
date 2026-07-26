import json
import re
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend.common import ACTIVE_WINDOW_DEFAULT, CLOCK_DEFAULT  # noqa: E402
from eww_bar_backend.state import BarState  # noqa: E402


class InitialStateTests(unittest.TestCase):
    def test_initial_clock_is_the_placeholder_not_a_live_reading(self):
        state = json.loads(BarState().snapshot())
        self.assertEqual(state["clock"], CLOCK_DEFAULT)

    def test_initial_active_window_is_the_default(self):
        state = json.loads(BarState().snapshot())
        self.assertEqual(state["active_window"], ACTIVE_WINDOW_DEFAULT)

    def test_clock_default_matches_the_live_collector_shape(self):
        # CLOCK_DEFAULT stands in for clock_state() until the daemon's first
        # update; if the collector's shape changes, the placeholder must follow.
        from eww_bar_backend.collectors import clock_state

        self.assertEqual(set(CLOCK_DEFAULT), set(clock_state()))

    def test_yuck_initial_matches_bar_state(self):
        first_line = (EWW_DIR / "eww.yuck").read_text().splitlines()[0]
        match = re.search(r":initial '(.*?)' \"eww-bar-backend bar\"\)", first_line)
        self.assertIsNotNone(match, "could not find the deflisten :initial literal")
        from_yuck = json.loads(match.group(1))
        from_python = json.loads(BarState().snapshot())
        if from_yuck != from_python:
            self.fail(
                "eww.yuck :initial has drifted from BarState().\n"
                "Replace the literal on eww.yuck line 1 with exactly:\n\n"
                + BarState().snapshot()
                + "\n"
            )


if __name__ == "__main__":
    unittest.main()
