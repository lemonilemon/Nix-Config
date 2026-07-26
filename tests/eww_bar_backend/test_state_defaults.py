import json
import re
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend.state import BarState  # noqa: E402


class InitialStateTests(unittest.TestCase):
    def test_bar_state_is_deterministic(self):
        # Two constructions a moment apart must be identical, or the literal
        # below can never be asserted against it.
        self.assertEqual(BarState().snapshot(), BarState().snapshot())

    def test_yuck_initial_matches_bar_state(self):
        first_line = (EWW_DIR / "eww.yuck").read_text().splitlines()[0]
        match = re.search(r":initial '(.*)' \"eww-bar-backend bar\"\)", first_line)
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
