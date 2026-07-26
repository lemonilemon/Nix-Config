import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control, ctl  # noqa: E402


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

    def test_arg_parsing_for_new_verbs(self):
        self.assertEqual(
            ctl.control_payload_from_args(["volume", "set", "40"]),
            {"command": "volume", "action": "set", "value": "40"},
        )
        self.assertEqual(
            ctl.control_payload_from_args(["volume", "sink", "bt_headset"]),
            {"command": "volume", "action": "sink", "sink": "bt_headset"},
        )
        self.assertEqual(
            ctl.control_payload_from_args(["volume", "up"]),
            {"command": "volume", "action": "up"},
        )


if __name__ == "__main__":
    unittest.main()
