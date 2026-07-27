import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class VolumeEventFilterTests(unittest.TestCase):
    """`pactl subscribe` client events must not re-trigger the volume collector.

    Every child `volume_state` forks (wpctl, pactl) connects to PulseAudio as a
    client, and each connection emits new/change/remove back onto the very
    stream the volume watcher reads -- a closed loop. Measured at rest on this
    host before the filter: 1902 events in 90 s, 100% of them `client`.
    """

    def test_client_events_are_ignored(self):
        for verb in ("new", "change", "remove"):
            self.assertFalse(
                collectors.volume_event_is_relevant(f"Event '{verb}' on client #75388")
            )

    def test_sink_and_server_events_still_collect(self):
        for line in (
            "Event 'change' on sink #1",
            "Event 'new' on sink #2",
            "Event 'remove' on sink #2",
            "Event 'change' on server #0",
            "Event 'change' on sink-input #45",
            "Event 'new' on source #3",
        ):
            self.assertTrue(collectors.volume_event_is_relevant(line), line)

    def test_unrecognized_lines_still_collect(self):
        # Fail open: an unparseable line costs one collect, a missed sink event
        # leaves the bar stale until the next 60 s poll.
        self.assertTrue(collectors.volume_event_is_relevant("Connection failure"))
        self.assertTrue(collectors.volume_event_is_relevant(""))


class VolumeSinkCacheTests(unittest.TestCase):
    """volume_state runs on every scroll tick; enumerating sinks costs 2 forks."""

    def setUp(self):
        collectors.reset_volume_sinks_cache()
        self.addCleanup(collectors.reset_volume_sinks_cache)

    def _fake_run_text(self, calls):
        def fake(command, **_kwargs):
            calls.append(list(command))
            if command[:2] == ["pactl", "get-default-sink"]:
                return "alsa_speaker\n"
            if "json" in command:
                return '[{"index":1,"name":"alsa_speaker","description":"Built-in"}]'
            return ""

        return fake

    def test_first_call_populates_the_cache(self):
        calls = []
        with unittest.mock.patch.object(collectors, "run_text", side_effect=self._fake_run_text(calls)):
            sinks = collectors.volume_sinks()
        self.assertEqual(sinks, [{"name": "alsa_speaker", "description": "Built-in", "active": "true"}])
        self.assertEqual(len(calls), 2)

    def test_refresh_false_reuses_the_cache_without_forking(self):
        calls = []
        with unittest.mock.patch.object(collectors, "run_text", side_effect=self._fake_run_text(calls)):
            first = collectors.volume_sinks()
            calls.clear()
            second = collectors.volume_sinks(refresh=False)
        self.assertEqual(second, first)
        self.assertEqual(calls, [])

    def test_cold_cache_refreshes_even_when_not_asked_to(self):
        calls = []
        with unittest.mock.patch.object(collectors, "run_text", side_effect=self._fake_run_text(calls)):
            sinks = collectors.volume_sinks(refresh=False)
        self.assertEqual(len(calls), 2)
        self.assertEqual(len(sinks), 1)

    def test_cached_sinks_are_copies(self):
        calls = []
        with unittest.mock.patch.object(collectors, "run_text", side_effect=self._fake_run_text(calls)):
            first = collectors.volume_sinks()
            first[0]["description"] = "mutated"
            second = collectors.volume_sinks(refresh=False)
        self.assertEqual(second[0]["description"], "Built-in")

    def test_volume_state_without_sink_refresh_forks_only_wpctl(self):
        calls = []
        with unittest.mock.patch.object(collectors, "run_text", side_effect=self._fake_run_text(calls)):
            collectors.volume_sinks()
            calls.clear()
            collectors.volume_state(refresh_sinks=False)
        self.assertEqual(calls, [["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"]])


class VolumeStateTests(unittest.TestCase):
    def test_unmuted_state(self):
        state = collectors.volume_state_from_text("Volume: 0.45\n", sinks=[])
        self.assertEqual(state["percent"], 45)
        self.assertEqual(state["muted"], "false")
        self.assertNotEqual(state["text"], "")
        self.assertEqual(state["sinks"], [])

    def test_muted_state(self):
        state = collectors.volume_state_from_text("Volume: 0.80 [MUTED]", sinks=[])
        self.assertEqual(state["muted"], "true")
        self.assertEqual(state["class"], "muted")

    def test_no_reading_falls_back_to_default(self):
        state = collectors.volume_state_from_text("garbage", sinks=[])
        self.assertEqual(state["percent"], 0)


class SinkParsingTests(unittest.TestCase):
    def test_json_sinks_mark_default_and_fall_back_description(self):
        payload = (
            '[{"index":56,"name":"alsa_speaker","description":"Built-in Speaker"},'
            '{"index":83,"name":"bt_headset","description":""}]'
        )
        sinks = collectors.sinks_from_pactl_json(payload, "bt_headset")
        self.assertEqual(
            sinks,
            [
                {"name": "alsa_speaker", "description": "Built-in Speaker", "active": "false"},
                {"name": "bt_headset", "description": "bt_headset", "active": "true"},
            ],
        )

    def test_short_sinks_fallback(self):
        short = "56\talsa_speaker\tPipeWire\ts32le\tSUSPENDED\n83\tbt_headset\tPipeWire\ts16le\tRUNNING\n"
        sinks = collectors.sinks_from_pactl_short(short, "bt_headset")
        self.assertEqual(
            sinks,
            [
                {"name": "alsa_speaker", "description": "alsa_speaker", "active": "false"},
                {"name": "bt_headset", "description": "bt_headset", "active": "true"},
            ],
        )


class VolumeControlTests(unittest.TestCase):
    def _capture_run(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_set_volume_clamps_and_formats_percent(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={"ok": 1}):
                control.set_volume("142.6")
        self.assertEqual(calls, [["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", "100%"]])

    def test_toggle_mute(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={}):
                control.toggle_mute()
        self.assertEqual(calls, [["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"]])

    def test_set_sink(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={}):
                control.set_sink("bt_headset")
        self.assertEqual(calls, [["pactl", "set-default-sink", "bt_headset"]])

    def test_volume_nudges_do_not_re_enumerate_sinks(self):
        # The scroll path. A volume nudge cannot change which sinks exist or
        # which one is default, so it must reuse the cached list.
        for name, invoke in (
            ("adjust_volume", lambda: control.adjust_volume("up")),
            ("set_volume", lambda: control.set_volume("40")),
            ("toggle_mute", control.toggle_mute),
        ):
            _calls, fake_run = self._capture_run()
            with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
                with unittest.mock.patch.object(control, "volume_state", return_value={}) as spy:
                    invoke()
            spy.assert_called_once_with(refresh_sinks=False)

    def test_set_sink_does_re_enumerate(self):
        # Switching the default sink changes every entry's `active` flag.
        _calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={}) as spy:
                control.set_sink("bt_headset")
        spy.assert_called_once_with()

if __name__ == "__main__":
    unittest.main()
