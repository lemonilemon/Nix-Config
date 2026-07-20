import json
import sys
import time
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors  # noqa: E402


def epoch_for(date_str):
    # Local-time epoch so it lines up with the collector's time.localtime() usage.
    return time.mktime(time.strptime(date_str + " 08:00:00", "%Y-%m-%d %H:%M:%S"))


class AiUsagePeriodTests(unittest.TestCase):
    def test_today_without_row_shows_current_date_and_zero(self):
        now = epoch_for("2026-07-20")  # Monday; no daily row for this date
        rows = [
            {"date": "2026-07-17", "totalTokens": 1000, "totalCost": 1.0},
            {"date": "2026-07-18", "totalTokens": 2000, "totalCost": 2.0},
        ]

        today = collectors.period_state(rows, "daily", "Today", now_epoch=now)

        # Range reflects the actual current day, not the last active row.
        self.assertEqual(today["range"], collectors.period_range_label("daily", "2026-07-20", now))
        self.assertEqual(today["range"], time.strftime("%a %b %-d", time.localtime(now)))
        self.assertEqual(today["tokens"], "--")
        self.assertEqual(today["cost"], "--")
        self.assertEqual(today["agents"], [])

    def test_today_with_row_still_uses_it(self):
        now = epoch_for("2026-07-20")
        rows = [
            {"date": "2026-07-18", "totalTokens": 2000, "totalCost": 2.0},
            {"date": "2026-07-20", "totalTokens": 5_000_000, "totalCost": 3.5},
        ]

        today = collectors.period_state(rows, "daily", "Today", now_epoch=now)

        self.assertEqual(today["tokens"], "5.0M")
        self.assertEqual(today["cost"], "$3.50")

    def test_week_without_row_advances_to_current_week(self):
        # Last known week starts 2026-07-12; now is in the following week.
        now = epoch_for("2026-07-20")
        rows = [{"period": "2026-07-12", "totalTokens": 9000, "totalCost": 4.0}]

        week = collectors.period_state(rows, "weekly", "This week", now_epoch=now)

        expected_start = time.strftime("%F", time.localtime(epoch_for("2026-07-19")))
        self.assertEqual(week["range"], collectors.period_range_label("weekly", expected_start, now))
        self.assertEqual(week["tokens"], "--")

    def test_from_json_treats_zero_today_as_valid_data(self):
        now = epoch_for("2026-07-20")
        report = json.dumps(
            {
                "daily": [{"date": "2026-07-18", "totalTokens": 2000, "totalCost": 2.0}],
                "weekly": [],
                "monthly": [{"period": "2026-07", "totalTokens": 2000, "totalCost": 2.0}],
            }
        )

        state = collectors.ai_usage_state_from_json(report, now_epoch=now)

        # Not the "missing" placeholder: it parsed real rows.
        self.assertEqual(state["source"], "ccusage")
        self.assertEqual(state["periods"]["today"]["range"], time.strftime("%a %b %-d", time.localtime(now)))
        self.assertEqual(state["periods"]["today"]["tokens"], "--")
        # Month-to-date still shows the July row.
        self.assertEqual(state["periods"]["month"]["tokens"], "2.0K")

    def test_from_json_without_any_rows_is_missing(self):
        now = epoch_for("2026-07-20")
        state = collectors.ai_usage_state_from_json(json.dumps({"daily": [], "weekly": [], "monthly": []}), now_epoch=now)
        self.assertEqual(state["source"], "missing")


if __name__ == "__main__":
    unittest.main()
