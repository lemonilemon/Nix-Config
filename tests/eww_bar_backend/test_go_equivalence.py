"""Differential test: every ported collector must agree with its Python original.

Table tests pin the cases someone thought of. This pins the ones nobody did --
it generates inputs across each function's domain, runs both implementations
over them, and fails on the first disagreement.

It is the gate for the Go port. A function is not ported until it appears here.

Skipped when the Go toolchain is absent, so the suite still runs on a machine
without it; the flake check builds the Go package separately, where the same
cases run inside the derivation.
"""

import json
import shutil
import subprocess
import sys
import unittest
from decimal import ROUND_HALF_UP, Decimal
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
GO_DIR = EWW_DIR / "go"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, notifications, watchers  # noqa: E402
from eww_bar_backend.common import truncate_text  # noqa: E402


NERD_GLYPHS = "\U000f075f\U000f00af"


def go_available():
    return shutil.which("go") is not None


class GoEquivalenceTests(unittest.TestCase):
    """Each test builds a case list, asks Go, and compares against Python."""

    @classmethod
    def setUpClass(cls):
        if not go_available():
            raise unittest.SkipTest("go toolchain not on PATH")
        cls.cache = REPO_ROOT / ".go-build-cache"
        cls.cache.mkdir(exist_ok=True)

    def _ask_go(self, calls):
        payload = "\n".join(json.dumps(c) for c in calls) + "\n"
        result = subprocess.run(
            ["go", "run", "./internal/collect/diffgen"],
            cwd=GO_DIR,
            input=payload,
            capture_output=True,
            text=True,
            env={
                "PATH": __import__("os").environ.get("PATH", ""),
                "HOME": str(self.cache),
                "GOCACHE": str(self.cache / "gocache"),
                "GOFLAGS": "-mod=mod",
            },
        )
        if result.returncode != 0:
            self.fail(f"diffgen failed:\n{result.stderr}")
        lines = [line for line in result.stdout.splitlines() if line]
        self.assertEqual(len(lines), len(calls), "diffgen returned the wrong number of answers")
        return [json.loads(line) for line in lines]

    def _compare(self, fn, cases, python_fn):
        """cases is a list of arg-tuples; python_fn maps a tuple to the expected value."""
        answers = self._ask_go([{"fn": fn, "args": list(args)} for args in cases])
        mismatches = []
        for args, answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"{fn}{args} errored in Go: {answer.get('error')}")
            expected = python_fn(*args)
            if answer["value"] != expected:
                mismatches.append((args, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(
                f"  {fn}{a}: python={p!r}  go={g!r}" for a, p, g in mismatches[:20]
            )
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")

    # -- truncation: the rune-vs-byte hazard, 13 call sites --------------------

    def test_truncate_text(self):
        texts = [
            "",
            "short",
            "exactly-ten",
            "a" * 60,
            "Built-in Audio Analog Stereo",
            # Nerd Font glyphs are 3-4 bytes each: byte slicing cuts them apart.
            NERD_GLYPHS,
            NERD_GLYPHS + "a" * 40,
            " Speaker",
            "中文" * 30,
            "café" * 12,
            "\U0001f600" * 20,
            "mixed  中 \U0001f600 tail",
        ]
        cases = [(t, n) for t in texts for n in (0, 1, 2, 3, 4, 5, 10, 30, 60, 100)]
        self._compare("TruncateText", cases, truncate_text)

    # -- rounding: Python's round() is half-to-even, math.Round is not ---------

    def test_round_half_even(self):
        values = []
        for whole in range(-4, 5):
            for frac in (0.0, 0.25, 0.5, 0.75):
                values.append(whole + frac)
        values += [0.5, 1.5, 2.5, 3.5, -0.5, -1.5, -2.5, 62.5, 37.5, 99.5, 100.5]
        self._compare("RoundHalfEven", [(v,) for v in values], lambda v: round(v))

    def test_percent_part(self):
        cases = []
        for total in (0, 1, 3, 7, 100, 1000, 12345):
            for value in (0, 1, 2, 3, 50, 99, 100, 101, 6172, 12345):
                cases.append((float(value), float(total)))
        self._compare("PercentPart", cases, collectors.percent_part)

    def test_clamp_percent(self):
        values = [-50.0, -1.0, -0.5, 0.0, 0.5, 1.5, 2.5, 49.5, 50.5, 99.5, 100.0, 100.5, 250.0]
        self._compare("ClampPercent", [(v,) for v in values], collectors.clamp_percent)

    # -- float formatting ------------------------------------------------------

    def test_format_tokens(self):
        values = [
            -1.0, 0.0, 1.0, 9.0, 999.0, 999.4, 999.5, 999.9, 1000.0, 1024.0, 1500.0,
            9999.0, 99_950.0, 999_999.0, 1_000_000.0, 1_050_000.0, 1_150_000.0,
            12_345_678.0, 110_100_000.0,
        ]
        self._compare("FormatTokens", [(v,) for v in values], collectors.format_tokens)

    def test_format_cost(self):
        values = [-1.0, 0.0, 0.001, 0.005, 0.015, 0.025, 1.0, 1.005, 12.345, 99.999, 1234.5]
        self._compare("FormatCost", [(v,) for v in values], collectors.format_cost)

    def test_format_remaining(self):
        values = [-60.0, 0.0, 1.0, 59.0, 60.0, 61.0, 599.0, 3599.0, 3600.0, 3661.0, 86_399.0, 90_061.0]
        self._compare("FormatRemaining", [(v,) for v in values], collectors.format_remaining)

    # -- the volume collector, including the Decimal/float divergence ----------

    def _volume_texts(self):
        texts = ["", "garbage", "Volume: ", "no reading here"]
        # Every two-decimal value wpctl can emit, muted and not.
        for hundredth in range(0, 101):
            raw = f"0.{hundredth // 10}{hundredth % 10}" if hundredth < 100 else "1.00"
            texts.append(f"Volume: {raw}\n")
            texts.append(f"Volume: {raw} [MUTED]\n")
        # The only four three-decimal inputs in 0.000-0.999 where going through
        # float64 disagrees with exact Decimal: each lands on x.4999999999999
        # instead of x.5, so half-up rounds it down. Enumerated rather than
        # sampled -- an earlier version of this test picked eight "interesting
        # looking" three-decimal values, hit none of these four, and passed
        # against a deliberately broken float64 implementation.
        for raw in ("0.145", "0.285", "0.565", "0.575"):
            texts.append(f"Volume: {raw}\n")
            texts.append(f"Volume: {raw} [MUTED]\n")
        for thousandth in (5, 15, 125, 375, 625, 815, 875, 995):
            texts.append(f"Volume: 0.{thousandth:03d}\n")
        texts += ["Volume: 1.5\n", "Volume: 0\n", "Volume: 0.\n", "Volume:   0.42\n"]
        return texts

    def test_volume_label_from_text(self):
        self._compare(
            "VolumeLabelFromText",
            [(t,) for t in self._volume_texts()],
            collectors.volume_label_from_text,
        )

    def test_volume_state_from_text(self):
        cases = [(t,) for t in self._volume_texts()]
        answers = self._ask_go([{"fn": "VolumeStateFromText", "args": list(a)} for a in cases])
        mismatches = []
        for (text,), answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"errored in Go: {answer.get('error')}")
            expected = collectors.volume_state_from_text(text, sinks=[])
            if answer["value"] != expected:
                mismatches.append((text, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(f"  {t!r}: python={p!r} go={g!r}" for t, p, g in mismatches[:10])
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")

    def test_volume_event_is_relevant(self):
        lines = ["", "garbage", "Connection failure"]
        for verb in ("new", "change", "remove"):
            for obj in ("client", "sink", "sink-input", "source", "card", "server"):
                lines.append(f"Event '{verb}' on {obj} #12")
        self._compare(
            "VolumeEventIsRelevant",
            [(line,) for line in lines],
            collectors.volume_event_is_relevant,
        )


    # -- Python string semantics, which the parsers all rest on ---------------

    WEIRD_LINES = [
        "",
        "a",
        "a\n",
        "a\nb",
        "a\nb\n",
        "a\r\nb",
        "a\rb",
        "a\n\nb",
        "\na",
        # The eight boundaries strings.Split(s, "\n") does not know about.
        "a\vb",
        "a\fb",
        "a\x1cb",
        "a\x1db",
        "a\x1eb",
        "a\x85b",
        "a\u2028b",
        "a\u2029b",
        "trailing\r\n",
        "Controller AA:BB\r\n\tPowered: yes\r\n",
    ]

    def test_split_lines(self):
        self._compare("SplitLines", [(t,) for t in self.WEIRD_LINES], lambda t: t.splitlines())

    def test_split_whitespace_n(self):
        texts = [
            "",
            "   ",
            "one",
            "  one  ",
            "a b c",
            "a  b   c",
            "Device 80:99:E7 My Speaker",
            "Device 80:99:E7  Two  Spaces  Inside ",
            "\ta\tb\tc\t",
            "a\u00a0b",
            "a\u3000b",
        ]
        cases = [(t, n) for t in texts for n in (-1, 0, 1, 2, 3)]
        self._compare(
            "SplitWhitespaceN",
            cases,
            lambda t, n: t.split() if n < 0 else t.split(maxsplit=n),
        )

    def test_strip(self):
        texts = ["", " ", "  a  ", "\ta\t", "\x1ca\x1c", "\u00a0a\u00a0", "\u3000a\u3000", "a"]
        self._compare("Strip", [(t,) for t in texts], lambda t: t.strip())

    # -- bluetooth -------------------------------------------------------------

    CONTROLLERS = [
        "",
        "Controller AA:BB:CC\n\tPowered: yes\n",
        "Controller AA:BB:CC\n\tPowered: no\n",
        "Controller AA:BB:CC\n\tAlias: MyBT\n\tPowered: yes\n",
        "Controller AA:BB:CC\n\tAlias:   Spaced Name  \n\tPowered: whatever\n",
        "Controller\n\tPowered: yes\n",
        "\tPowered: yes\n",
        "Controller AA:BB:CC\r\n\tPowered: yes\r\n",
    ]

    def test_parse_controller(self):
        self._compare(
            "ParseController",
            [(t,) for t in self.CONTROLLERS],
            lambda t: list(collectors.parse_controller(t)),
        )

    def test_parse_device_info(self):
        infos = [
            "",
            "\tAlias: Buds Pro\n",
            "\tAlias: Buds Pro\n\tBattery Percentage: 0x55 (85)\n",
            "\tBattery Percentage: 0x64 (100)\n",
            "\tBattery Percentage: nonsense\n",
            "\tAlias:\n",
        ]
        cases = [(i, f) for i in infos for f in ("fallback", "")]
        self._compare(
            "ParseDeviceInfo",
            cases,
            lambda i, f: list(collectors.parse_device_info(i, f)),
        )

    def test_bluetooth_state_from_text(self):
        devices = [
            "",
            "Device 80:99:E7 Buds\n",
            "Device 80:99:E7 My Long Speaker Name That Runs Past The Limit\n",
            "Device AA:11 ugreen_1 Headset\nDevice BB:22 Other\n",
            "Device BB:22 Other\nDevice AA:11 UGREEN_2\n",
            "Device\n",
            "   \n",
        ]
        infos = [{}, {"80:99:E7": "\tAlias: Renamed\n\tBattery Percentage: 0x55 (85)\n"}]
        cases = [(c, d, i) for c in self.CONTROLLERS[:5] for d in devices for i in infos]
        answers = self._ask_go([{"fn": "BluetoothStateFromText", "args": list(a)} for a in cases])
        mismatches = []
        for args, answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"errored in Go: {answer.get('error')}")
            expected = collectors.bluetooth_state_from_text(*args)
            if answer["value"] != expected:
                mismatches.append((args, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(f"  {a}: python={p!r} go={g!r}" for a, p, g in mismatches[:6])
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")

    # -- the small text collectors --------------------------------------------

    def test_submap_from_event(self):
        lines = ["", "submap>>", "submap>>default", "submap>>reset", "submap>>resize",
                 "submap>>a>>b", "workspace>>1", "submap"]
        self._compare("SubmapFromEvent", [(l,) for l in lines], collectors.submap_from_event)

    def test_tray_count_from_text(self):
        texts = ["", "as 0", "as 3 \"a\" \"b\" \"c\"", "as", "as x", "  as   7  ",
                 "ay 2", "3", "as -1"]
        self._compare("TrayCountFromText", [(t,) for t in texts], collectors.tray_count_from_text)

    def test_memory_state_from_text(self):
        texts = [
            "",
            "MemTotal: 0 kB\n",
            "MemTotal:       16316360 kB\nMemAvailable:    8158180 kB\nSwapTotal:      8388604 kB\nSwapFree:       8388604 kB\n",
            "MemTotal:       16316360 kB\nMemAvailable:    1000000 kB\nSwapTotal:      8388604 kB\nSwapFree:       4000000 kB\n",
            "MemTotal:       16316360 kB\nMemAvailable:   16316360 kB\nSwapTotal:            0 kB\nSwapFree:             0 kB\n",
            "MemTotal:       1000 kB\nMemAvailable:    200 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n",
            "garbage\nMemTotal: 1000 kB\nMemAvailable: 500 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n",
        ]
        self._compare(
            "MemoryStateFromText", [(t,) for t in texts], collectors.memory_state_from_text
        )

    def test_media_state_from_text(self):
        statuses = ["", "Playing\n", "playing", "Paused\n", "Stopped\n", "Weird\n", "  Playing  \n"]
        metadata = [
            "",
            "\n",
            "   \n",
            "Artist - Title\n",
            "A" * 90 + "\n",
            "\U000f00af Glyphy - " + "b" * 70 + "\n",
            "中文歌手 - 中文歌名\n",
        ]
        cases = [(s, m) for s in statuses for m in metadata]
        self._compare(
            "MediaStateFromText", cases, collectors.media_state_from_text
        )

    # -- watcher helpers -------------------------------------------------------

    def test_monitor_event(self):
        lines = [
            "", "monitoradded>>", "monitoradded>>HDMI-A-1", "monitoraddedv2>>1,HDMI-A-1",
            "monitorremoved>>eDP-1", "monitorremovedv2>>0,eDP-1", "monitoradded>>  DP-3  ",
            "workspace>>1", "monitoradded",
        ]
        self._compare(
            "MonitorEvent",
            [(l,) for l in lines],
            lambda l: (list(watchers.monitor_event(l)) if watchers.monitor_event(l) else None),
        )

    def test_bar_window_command(self):
        cases = [(a, n) for a in ("added", "removed", "other") for n in ("eDP-1", "HDMI-A-1", "")]
        self._compare("BarWindowCommand", cases, watchers.bar_window_command)

    def test_open_bar_names(self):
        texts = [
            "",
            "bar-eDP-1: bar\n",
            "bar-eDP-1: bar\nbar-HDMI-A-1: bar\nvolume_popup: volume_popup\n",
            "garbage\n",
            "  bar-DP-3  : bar\n",
            "bar-: bar\n",
        ]
        self._compare(
            "OpenBarNames",
            [(t,) for t in texts],
            lambda t: sorted(watchers.open_bar_names(t)),
        )


class DecimalHazardTests(unittest.TestCase):
    """Documents why DecimalTimes100HalfUp does not go through float64.

    Not a Go test -- it asserts the divergence exists in Python, so the reason
    for the extra code in num.go stays visible if someone is tempted to
    simplify it into a multiply.
    """

    def test_float_and_decimal_disagree_on_three_decimals(self):
        disagreements = []
        for thousandth in range(0, 1001):
            raw = f"{thousandth / 1000:.3f}"
            exact = int((Decimal(raw) * 100).to_integral_value(rounding=ROUND_HALF_UP))
            naive = round(float(raw) * 100)
            if exact != naive:
                disagreements.append((raw, exact, naive))
        self.assertTrue(
            disagreements,
            "if this ever passes, float64 became safe here and num.go can be simplified",
        )

    def test_they_agree_on_every_two_decimal_value(self):
        # The range wpctl actually emits, which is why this has never bitten.
        for hundredth in range(0, 101):
            raw = f"{hundredth / 100:.2f}"
            exact = int((Decimal(raw) * 100).to_integral_value(rounding=ROUND_HALF_UP))
            self.assertEqual(exact, round(float(raw) * 100), raw)


if __name__ == "__main__":
    unittest.main()
