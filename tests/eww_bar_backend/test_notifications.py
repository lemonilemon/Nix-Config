import json
import sys
import time
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import notifications  # noqa: E402


def wrap(value):
    return {"type": "s", "data": value}


def history_fixture(items):
    return json.dumps({"type": "aa{sv}", "data": [items]})


ITEM_OLD = {
    "appname": wrap("Element"),
    "summary": wrap("Alice"),
    "body": wrap("check this out"),
    "id": {"type": "u", "data": 3},
    "timestamp": {"type": "x", "data": 100_000_000},
    "urgency": wrap("NORMAL"),
}
ITEM_NEW = {
    "appname": wrap("Element"),
    "summary": wrap("Alice"),
    "body": wrap("are we still on\nfor tonight?"),
    "id": {"type": "u", "data": 4},
    "timestamp": {"type": "x", "data": 200_000_000},
    "urgency": wrap("NORMAL"),
}
ITEM_POWER = {
    "appname": wrap("Power"),
    "summary": wrap("Battery"),
    "body": wrap("Fully charged"),
    "id": {"type": "u", "data": 5},
    "timestamp": {"type": "x", "data": 150_000_000},
    "urgency": wrap("CRITICAL"),
}


class ParseHistoryTests(unittest.TestCase):
    def test_parses_and_sorts_newest_first(self):
        items = notifications.parse_history_items(
            history_fixture([ITEM_OLD, ITEM_NEW, ITEM_POWER])
        )
        self.assertEqual([item["id"] for item in items], [4, 5, 3])
        self.assertEqual(items[0]["app"], "Element")
        self.assertEqual(items[1]["urgency"], "CRITICAL")

    def test_garbage_inputs_yield_empty(self):
        self.assertEqual(notifications.parse_history_items(""), [])
        self.assertEqual(notifications.parse_history_items("not json"), [])
        self.assertEqual(notifications.parse_history_items('{"data": {}}'), [])
        self.assertEqual(notifications.parse_history_items('{"data": [[{"id": {"data": "x"}}]]}'), [])


class FormatAgeTests(unittest.TestCase):
    def test_boundaries(self):
        self.assertEqual(notifications.format_age(0), "now")
        self.assertEqual(notifications.format_age(9.9), "now")
        self.assertEqual(notifications.format_age(59), "59s")
        self.assertEqual(notifications.format_age(60), "1m")
        self.assertEqual(notifications.format_age(3599), "59m")
        self.assertEqual(notifications.format_age(3600), "1h")
        self.assertEqual(notifications.format_age(86_399), "23h")
        self.assertEqual(notifications.format_age(200_000), "2d")


class StateFromPartsTests(unittest.TestCase):
    def build(self, collapsed=frozenset(), last_seen=0, paused="false\n"):
        items = notifications.parse_history_items(
            history_fixture([ITEM_OLD, ITEM_NEW, ITEM_POWER])
        )
        return notifications.notifications_state_from_parts(
            items, paused, 260_000_000, set(collapsed), last_seen
        )

    def test_groups_ordered_by_newest_item(self):
        state = self.build()
        self.assertEqual([group["app"] for group in state["groups"]], ["Element", "Power"])
        self.assertEqual(state["groups"][0]["count"], 2)
        self.assertEqual(state["count"], 3)

    def test_body_newlines_flattened_and_ages_formatted(self):
        state = self.build()
        top = state["groups"][0]["items"][0]
        self.assertEqual(top["body"], "are we still on for tonight?")
        self.assertEqual(top["age"], "1m")

    def test_collapsed_and_new_count(self):
        state = self.build(collapsed={"Element"}, last_seen=150_000_000)
        self.assertEqual(state["groups"][0]["collapsed"], "true")
        self.assertEqual(state["groups"][1]["collapsed"], "false")
        self.assertEqual(state["new"], 1)  # only ITEM_NEW is newer than last_seen

    def test_paused_parsing(self):
        self.assertEqual(self.build(paused="true\n")["paused"], "true")
        self.assertEqual(self.build(paused="")["paused"], "false")

    def test_truncation(self):
        long_item = {
            "appname": wrap("A" * 40),
            "summary": wrap("S" * 60),
            "body": wrap("B" * 90),
            "id": {"type": "u", "data": 9},
            "timestamp": {"type": "x", "data": 1},
            "urgency": wrap("LOW"),
        }
        items = notifications.parse_history_items(history_fixture([long_item]))
        state = notifications.notifications_state_from_parts(items, "false", 2, set(), 0)
        group = state["groups"][0]
        self.assertEqual(len(group["app"]), 20)
        self.assertEqual(len(group["items"][0]["summary"]), 48)
        self.assertEqual(len(group["items"][0]["body"]), 64)


class NotifActionTests(unittest.TestCase):
    def test_toggle_group_flips_membership(self):
        notifications._COLLAPSED.clear()
        calls = []
        original = notifications.notifications_state
        notifications.notifications_state = lambda: calls.append(1) or {"stub": True}
        try:
            notifications.toggle_group("Element")
            self.assertIn("Element", notifications._COLLAPSED)
            notifications.toggle_group("Element")
            self.assertNotIn("Element", notifications._COLLAPSED)
        finally:
            notifications.notifications_state = original
        self.assertEqual(len(calls), 2)

    def test_toggle_group_requires_app(self):
        with self.assertRaises(ValueError):
            notifications.toggle_group("")

    def test_dismiss_requires_numeric_id(self):
        with self.assertRaises(ValueError):
            notifications.dismiss_notification("abc")


class DunstClockTests(unittest.TestCase):
    """notifications.py:73-76 -- dunst stamps history with CLOCK_BOOTTIME.

    The two clocks are identical until the machine suspends, and then they
    diverge by exactly the accumulated suspend time. So a rewrite that reaches
    for the more familiar CLOCK_MONOTONIC passes every test, works all day, and
    then shows every notification age as "now" after the first lid close --
    a bug that needs a suspend cycle to reproduce and looks like nothing in a
    diff.
    """

    def test_it_asks_for_boottime_not_monotonic(self):
        with unittest.mock.patch.object(
            notifications.time, "clock_gettime", return_value=123.0
        ) as clock:
            self.assertEqual(notifications.boottime_seconds(), 123.0)
        clock.assert_called_once_with(time.CLOCK_BOOTTIME)

    def test_boottime_is_never_behind_monotonic_on_this_platform(self):
        # The property the choice rests on, asserted against the real clocks:
        # BOOTTIME counts suspend, MONOTONIC does not, so BOOTTIME >= MONOTONIC.
        # Sample MONOTONIC first -- the two reads are ~500 ns apart, which is
        # enough to invert the comparison on a machine that has never slept.
        monotonic = time.clock_gettime(time.CLOCK_MONOTONIC)
        boottime = time.clock_gettime(time.CLOCK_BOOTTIME)
        self.assertGreaterEqual(boottime, monotonic)

    def test_falls_back_to_monotonic_when_boottime_is_unavailable(self):
        def only_monotonic(clock_id):
            if clock_id == time.CLOCK_BOOTTIME:
                raise OSError("no BOOTTIME here")
            return 42.0

        with unittest.mock.patch.object(
            notifications.time, "clock_gettime", side_effect=only_monotonic
        ):
            self.assertEqual(notifications.boottime_seconds(), 42.0)

    def test_returns_zero_rather_than_raising_when_no_clock_works(self):
        with unittest.mock.patch.object(
            notifications.time, "clock_gettime", side_effect=OSError
        ):
            self.assertEqual(notifications.boottime_seconds(), 0.0)


if __name__ == "__main__":
    unittest.main()
