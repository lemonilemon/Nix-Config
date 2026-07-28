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
import re
import shutil
import subprocess
import time
import sys
import unittest
import unittest.mock
from decimal import ROUND_HALF_UP, Decimal
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
GO_DIR = EWW_DIR / "go"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import (  # noqa: E402
    collectors,
    control,
    display,
    inhibitors,
    notifications,
    paths,
    wallpaper,
    watchers,
)
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
                # Pure Go, no C toolchain. net/http pulls in cgo for the
                # resolver, which would make this gate depend on gcc being on
                # PATH -- a dependency the skip-when-go-is-absent guard does not
                # cover, so it would fail rather than skip. Nothing here
                # resolves a hostname: httpGetText is stubbed by the fixture.
                "CGO_ENABLED": "0",
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


    # -- network: the densest workaround in the package -----------------------

    ROUTE_TEXTS = [
        "",
        "default via 192.168.0.1 dev eno1 proto dhcp src 192.168.0.88 metric 100\n",
        # Docked: two default routes, lowest metric wins.
        "default via 192.168.0.1 dev wlan0 proto dhcp metric 600\n"
        "default via 192.168.0.1 dev eno1 proto dhcp metric 100\n",
        # Reversed order: the winner must not depend on line order.
        "default via 192.168.0.1 dev eno1 proto dhcp metric 100\n"
        "default via 192.168.0.1 dev wlan0 proto dhcp metric 600\n",
        # No metric at all means 0, which beats an explicit 100.
        "default via 192.168.0.1 dev wlan0 proto dhcp metric 100\n"
        "default via 192.168.0.1 dev eno1 proto dhcp\n",
        # Tied metrics: Python keeps the FIRST, because its comparison is
        # strictly less-than. A <= would silently prefer the last, and no other
        # fixture here distinguishes them -- the mutation check found that gap.
        "default via 192.168.0.1 dev wlan0 proto dhcp metric 100\n"
        "default via 192.168.0.1 dev eno1 proto dhcp metric 100\n",
        # Both metric-less, so both 0: the same tie, reached a different way.
        "default via 192.168.0.1 dev wlan0 proto dhcp\n"
        "default via 192.168.0.1 dev eno1 proto dhcp\n",
        "default via 192.168.0.1 dev eno1 metric notanumber\n",
        "default via 192.168.0.1 dev\n",
        "default via 192.168.0.1\n",
        "192.168.0.0/24 dev eno1 proto kernel scope link src 192.168.0.88\n",
        # ECMP: deliberately unhandled, must yield "".
        "default proto static\n\tnexthop via 10.0.0.1 dev eno1 weight 1\n",
        "\n\n",
    ]

    ADDR_TEXTS = [
        "",
        "2: eno1    inet 192.168.0.88/24 brd 192.168.0.255 scope global dynamic eno1\n",
        "1: lo    inet 127.0.0.1/8 scope host lo\n"
        "2: eno1    inet 192.168.0.88/24 brd 192.168.0.255 scope global dynamic eno1\n"
        "3: wlan0    inet 10.0.0.5/24 scope global dynamic wlan0\n",
        "2: eno1 inet\n",
        "garbage\n",
    ]

    def test_default_route_device(self):
        self._compare(
            "DefaultRouteDevice",
            [(t,) for t in self.ROUTE_TEXTS],
            collectors.default_route_device,
        )

    def test_device_ipv4(self):
        cases = [(t, d) for t in self.ADDR_TEXTS for d in ("eno1", "wlan0", "lo", "missing")]
        self._compare("DeviceIPv4", cases, collectors.device_ipv4)

    def test_link_state_from_text(self):
        # The dead-NetworkManager workaround. None vs a value is the whole
        # point: None means believe nmcli, a value means the kernel still has a
        # route so we are online and nmcli is simply not reporting.
        cases = [(r, a) for r in self.ROUTE_TEXTS for a in self.ADDR_TEXTS]
        self._compare("LinkStateFromText", cases, collectors.link_state_from_text)

    def test_connected_device(self):
        statuses = [
            "",
            "eno1:ethernet:connected:Wired connection 1\n",
            "wlan0:wifi:connected:MyNet\nen1:ethernet:connected:Wired\n",
            "wlan0:wifi:disconnected:\n",
            "malformed\n",
            "a:b\n",
        ]
        cases = [(s, t) for s in statuses for t in ("wifi", "ethernet", "bridge")]
        self._compare("ConnectedDevice", cases, collectors.connected_device)

    def test_nmcli_value_and_first_ip(self):
        texts = [
            "",
            "GENERAL.CONNECTION:MyNet\nIP4.ADDRESS[1]:10.0.0.5/24\n",
            "IP4.ADDRESS[1]:10.0.0.5/24\n",
            "GENERAL.CONNECTION:\n",
            "no colon here\n",
            "GENERAL.CONNECTION:has:colons:inside\n",
        ]
        self._compare(
            "NmcliValue",
            [(t, k) for t in texts for k in ("GENERAL.CONNECTION", "IP4.ADDRESS[1]", "MISSING")],
            collectors.nmcli_value,
        )
        self._compare("FirstIP", [(t,) for t in texts], collectors.first_ip)

    def test_wireless_signal_percent(self):
        texts = [
            "",
            "Inter-| sta-|   Quality        |   Discarded packets\n"
            " face | tus | link level noise |  nwid  crypt   frag\n"
            " wlan0: 0000   70.  -40.  -256        0      0      0\n",
            " wlan0: 0000   35.  -40.  -256\n",
            " wlan0: 0000   0.  -40.  -256\n",
            " wlan0: 0000   105.  -40.  -256\n",
            " wlan0: 0000   notanumber  -40.\n",
            " wlan0: 0000\n",
            " wlan1: 0000   70.  -40.\n",
        ]
        cases = [(t, i) for t in texts for i in ("wlan0", "wlan1", "eno1")]
        self._compare(
            "WirelessSignalPercent", cases, collectors.wireless_signal_percent
        )

    def test_network_state_from_text(self):
        statuses = [
            "",
            "wlan0:wifi:connected:MyNet\n",
            "eno1:ethernet:connected:Wired\n",
            "wlan0:wifi:connected:MyNet\neno1:ethernet:connected:Wired\n",
            "wlan0:wifi:disconnected:\neno1:ethernet:disconnected:\n",
        ]
        wifis = ["", "yes:MyNet:72\n", "no:Other:30\nyes:MyNet:72\n", "yes::0\n", "yes:Net\n"]
        ips = [
            {},
            {"wlan0": "IP4.ADDRESS[1]:10.0.0.5/24\n"},
            {"eno1": "IP4.ADDRESS[1]:192.168.0.88/24\n"},
            {"wlan0": "", "eno1": ""},
        ]
        cases = [(s, w, i) for s in statuses for w in wifis for i in ips]
        answers = self._ask_go([{"fn": "NetworkStateFromText", "args": list(a)} for a in cases])
        mismatches = []
        for args, answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"errored in Go: {answer.get('error')}")
            expected = collectors.network_state_from_text(*args)
            if answer["value"] != expected:
                mismatches.append((args, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(f"  {a}: python={p!r} go={g!r}" for a, p, g in mismatches[:6])
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")


    # -- notifications ---------------------------------------------------------

    def _dunst(self, entries):
        """Wrap entries the way `dunstctl history` does."""
        return json.dumps({"type": "aa{sv}", "data": [entries]})

    def _wrap(self, **fields):
        return {k: {"type": "s", "data": v} for k, v in fields.items()}

    def test_parse_history_items(self):
        payloads = [
            "",
            "not json",
            "[]",
            '{"data": []}',
            '{"data": [[]]}',
            '{"data": "wrong"}',
            '{"data": [["not a dict"]]}',
            self._dunst([self._wrap(id=1, timestamp=1000, appname="kitty",
                                    summary="Hi", body="There", urgency="NORMAL")]),
            # Falsy fields fall back: "" -> unknown / NORMAL.
            self._dunst([self._wrap(id=2, timestamp=2000, appname="",
                                    summary="", body="", urgency="")]),
            # Missing wrappers entirely.
            self._dunst([{"id": {"type": "i", "data": 3}}]),
            # A bare (unwrapped) field reads as absent -- _field returns the
            # default unless the value is a {"type","data"} dict.
            self._dunst([{"id": {"type": "i", "data": 4}, "appname": "bare"}]),
            # int() truncates a float id and parses a numeric string.
            self._dunst([self._wrap(id=5.9, timestamp=5000, appname="a")]),
            self._dunst([self._wrap(id="6", timestamp="6000", appname="a")]),
            # Non-numeric id: the whole entry is skipped.
            self._dunst([self._wrap(id="abc", timestamp=7000, appname="a")]),
            # str() distinguishes 5 from 5.0 -- Go loses that without UseNumber.
            self._dunst([self._wrap(id=8, timestamp=8000, appname=5)]),
            self._dunst([self._wrap(id=9, timestamp=9000, appname=5.0)]),
            # Stable sort: equal timestamps must keep input order. Twenty of
            # them, not three -- Go's sort.Slice falls back to insertion sort
            # below a dozen elements, which happens to be stable, so a small
            # fixture cannot tell sort.Slice from sort.SliceStable.
            self._dunst([
                self._wrap(id=100 + n, timestamp=500, appname=f"app{n:02d}")
                for n in range(20)
            ]),
            # Interleaved ties: two timestamps, ten entries each.
            self._dunst([
                self._wrap(id=200 + n, timestamp=500 if n % 2 else 900,
                           appname=f"mix{n:02d}")
                for n in range(20)
            ]),
            self._dunst([
                self._wrap(id=13, timestamp=100, appname="old"),
                self._wrap(id=14, timestamp=900, appname="new"),
                self._wrap(id=15, timestamp=500, appname="mid"),
            ]),
        ]
        self._compare(
            "ParseHistoryItems",
            [(p,) for p in payloads],
            notifications.parse_history_items,
        )

    def test_format_age(self):
        values = [0.0, 0.4, 9.0, 9.999, 10.0, 10.5, 59.9, 60.0, 61.0, 3599.0,
                  3600.0, 3661.0, 86399.0, 86400.0, 172800.0, 1000000.0]
        self._compare("FormatAge", [(v,) for v in values], notifications.format_age)

    def test_notifications_state_from_parts(self):
        def items(*specs):
            return [
                {"id": i, "app": a, "summary": s, "body": b, "urgency": u, "timestamp": t}
                for (i, a, s, b, u, t) in specs
            ]
        cases = [
            ([], "", 0.0, [], 0),
            (items((1, "kitty", "Sum", "Body", "NORMAL", 0)), "true\n", 5_000_000.0, [], 0),
            (items((1, "kitty", "Sum", "Body", "NORMAL", 0)), " true ", 5_000_000.0, ["kitty"], 0),
            (items((1, "kitty", "S", "B", "NORMAL", 0)), "false", 5_000_000.0, [], 0),
            # Grouping keeps first-appearance order, not sorted order.
            (items((1, "zed", "a", "b", "LOW", 300),
                   (2, "alpha", "c", "d", "NORMAL", 200),
                   (3, "zed", "e", "f", "CRITICAL", 100)), "", 1_000_000.0, [], 0),
            # `new` counts strictly-greater than last seen.
            (items((1, "a", "s", "b", "NORMAL", 100),
                   (2, "a", "s", "b", "NORMAL", 200),
                   (3, "a", "s", "b", "NORMAL", 300)), "", 1_000_000.0, [], 200),
            # A future timestamp must clamp the age to 0, not go negative.
            (items((1, "a", "s", "b", "NORMAL", 9_000_000)), "", 1_000_000.0, [], 0),
            # Truncation at each of the three limits, and newlines flattened.
            (items((1, "a" * 40, "s" * 80, "line1\nline2\n" + "b" * 80, "NORMAL", 0)),
             "", 1_000_000.0, [], 0),
            # Group names are the TRUNCATED app, so two long names collapse into one.
            (items((1, "x" * 30, "s", "b", "NORMAL", 200),
                   (2, "x" * 31, "s", "b", "NORMAL", 100)), "", 1_000_000.0, [], 0),
        ]
        answers = self._ask_go([
            {"fn": "NotificationsStateFromParts", "args": [i, p, n, c, l]}
            for (i, p, n, c, l) in cases
        ])
        mismatches = []
        for (i, p, n, c, l), answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"errored in Go: {answer.get('error')}")
            expected = notifications.notifications_state_from_parts(i, p, n, set(c), l)
            if answer["value"] != expected:
                mismatches.append(((i, p, n, c, l), expected, answer["value"]))
        if mismatches:
            detail = "\n".join(f"  python={p!r}\n  go    ={g!r}" for _a, p, g in mismatches[:4])
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")

    # -- wallpaper --------------------------------------------------------------

    def test_path_stem(self):
        paths = [
            "/w/a.png", "/w/a.tar.gz", "/w/noext", "/w/.bashrc", "/w/.hidden.png",
            "a.png", "/w/", "", "/w/name with spaces.jpeg", "/w/\u4e2d\u6587.png",
            # The shapes where filepath.Base and pathlib disagree.
            ".", "..", "/", "//", "/w/.", "/w/..", "./a.png", "/w//a.png",
            # Dot handling: pathlib ignores leading dots when finding the
            # suffix boundary, and 3.14 changed what a TRAILING dot means.
            "a.", "a..", ".a", "..a", "...", "....", ".a.b", "..a.b", "a.b..", "a.b.c",
        ]
        self._compare("PathStem", [(p,) for p in paths], lambda p: Path(p).stem)

    def test_wallpaper_items(self):
        files = ["/w/a.png", "/w/b.GIF", "/w/c.gif", "/w/" + "d" * 40 + ".jpg", "/w/e"]
        # realBy stands in for Path.resolve(): the seed is a store symlink, so
        # the literal strings differ and only the resolved paths match.
        real_by = {"/w/a.png": "/nix/store/xxx-seed.png"}
        currents = ["", "/w/a.png", "/nix/store/xxx-seed.png", "/w/c.gif", "/w/missing"]
        cases = [(files, cur, real_by) for cur in currents]

        def python_side(files, current, real_by):
            def real(p):
                return real_by.get(p, p)
            import eww_bar_backend.wallpaper as wp
            saved = wp._real_path
            wp._real_path = real
            try:
                return wp.wallpaper_items(
                    [Path(f) for f in files], current, thumb_fn=lambda p: "thumb:" + str(p)
                )
            finally:
                wp._real_path = saved

        self._compare("WallpaperItems", cases, python_side)

    def test_rows_from_items(self):
        def item(n):
            return {"name": f"n{n}", "path": f"/w/{n}.png", "thumb": "t",
                    "animated": "false", "active": "false"}
        cases = []
        for count in (0, 1, 2, 3, 4, 6, 7):
            for columns in (1, 2, 3, 4):
                cases.append(([item(i) for i in range(count)], columns))
        self._compare(
            "RowsFromItems", cases, lambda items, columns: wallpaper.rows_from_items(items, columns)
        )


    # -- workspaces, monitors, wallpaper query ---------------------------------

    def test_workspace_state_from_json(self):
        actives = ["", "{}", '{"id": 1}', '{"id": 3}', '{"id": 9}', '{"id": "2"}', "not json"]
        workspaces = [
            "", "[]",
            '[{"id": 1, "windows": 2}]',
            '[{"id": 1, "windows": 0}, {"id": 2, "windows": 5}]',
            '[{"id": 4, "windows": 1}, "junk", {"no": "id"}]',
            '[{"id": "3", "windows": "2"}]',
        ]
        clients = [
            "", "[]",
            '[{"urgent": true, "workspace": {"id": 2}}]',
            # urgent is compared with `is True`, so 1 must NOT count.
            '[{"urgent": 1, "workspace": {"id": 2}}]',
            '[{"urgent": true, "workspace": {"id": 1}}]',
            '[{"urgent": true}]',
            '[{"urgent": true, "workspace": "notadict"}]',
        ]
        cases = [(a, w, c) for a in actives for w in workspaces for c in clients]
        self._compare(
            "WorkspaceStateFromJSON", cases, collectors.workspace_state_from_json
        )

    def test_missing_bar_monitors(self):
        monitors = [
            "", "[]", "not json", '{"name": "eDP-1"}',
            '[{"name": "eDP-1"}]',
            '[{"name": "eDP-1"}, {"name": "HDMI-A-1"}]',
            '[{"name": ""}, {"noname": 1}, "junk", {"name": "DP-3"}]',
        ]
        windows = ["", "bar-eDP-1: bar\n", "bar-eDP-1: bar\nbar-HDMI-A-1: bar\n", "junk\n"]
        cases = [(m, w) for m in monitors for w in windows]
        self._compare("MissingBarMonitors", cases, watchers.missing_bar_monitors)

    def test_parse_awww_query(self):
        texts = [
            "",
            "eDP-1: 1920x1200, scale: 2, currently displaying: image: /w/a.png\n",
            "eDP-1: ... image: /w/a.png\nHDMI-A-1: ... image: /w/b.png\n",
            "eDP-1: ... color: #000000\n",
            "image: \n",
            "image: /w/trailing spaces   \n",
            "no match here\n",
        ]
        self._compare("ParseAwwwQuery", [(t,) for t in texts], wallpaper.parse_awww_query)

    def test_is_internal_monitor(self):
        names = ["eDP-1", "eDP-2", "LVDS-1", "HDMI-A-1", "DP-3", "", "eDP", "edp-1", "XLVDS-1"]
        self._compare("IsInternalMonitor", [(n,) for n in names], display.is_internal_monitor)

    def test_split_monitors(self):
        monitor_sets = [
            [],
            [{"name": "eDP-1"}],
            [{"name": "eDP-1"}, {"name": "HDMI-A-1"}],
            [{"name": "eDP-1", "disabled": True}, {"name": "HDMI-A-1"}],
            [{"name": "eDP-1", "disabled": False}, {"name": "HDMI-A-1", "disabled": True}],
            [{"noname": 1}, {"name": "LVDS-1"}],
            # `not m.get("disabled", False)` -- any truthy value disables.
            [{"name": "eDP-1", "disabled": 1}, {"name": "DP-3", "disabled": 0}],
        ]
        self._compare(
            "SplitMonitors",
            [(m,) for m in monitor_sets],
            lambda m: [list(part) for part in display.split_monitors(m)],
        )


    # -- the state encoder: the daemon's entire output contract ----------------

    ENCODE_VALUES = [
        None, True, False, 0, 1, -1, 42, "", "plain",
        [], {}, [1, 2, 3], ["a", "b"], [[]], [{}],
        {"a": 1}, {"a": {"b": {"c": []}}},
        # The shapes the real snapshot has.
        {"text": "", "tooltip": "", "class": ""},
        {"sinks": [], "percent": 0, "muted": "false"},
        {"groups": [{"app": "kitty", "count": 2, "items": [{"id": 1}]}]},
        # Escaping: quotes, backslashes, control characters, and the tab and
        # newline the bluetooth tooltip is built from.
        "quote\"inside", "back\\slash", "tab\there", "nl\nhere",
        "ctrl\x01\x1f", "\x7f",
        "AA:BB\tCC\n\n2 connected",
        # CPython leaves these literal; encoding/json escapes them.
        "a<b>c&d", "slash/here",
        # Non-ASCII: raw under ensure_ascii=False, escaped under True. Every
        # glyph the bar actually emits.
        "\uf001", "\uf017", "\uf10c", "\uf111", "\U000f0084", "\U000f00af",
        "\U000f075f", "\U000f0674", "\u26a0", "\u2014",
        "caf\u00e9", "\u4e2d\u6587", "\U0001f600",
        "\U000f0084 mixed \u4e2d ascii",
        # Nested, with glyphs inside lists inside dicts.
        {"battery": {"text": "\U000f0084", "capacity": 100},
         "quotas": [{"key": "claude", "windows": []}]},
    ]

    def test_encoder_matches_json_dumps(self):
        for ensure_ascii in (False, True):
            cases = [(v, ensure_ascii) for v in self.ENCODE_VALUES]
            with self.subTest(ensure_ascii=ensure_ascii):
                self._compare(
                    "EncodeJSON",
                    cases,
                    lambda v, a: json.dumps(v, separators=(",", ":"), ensure_ascii=a),
                )

    def test_encoder_matches_the_live_snapshot_shape(self):
        # The real thing: build BarState and round-trip its snapshot through the
        # Go encoder. This is the contract eww consumes.
        from eww_bar_backend.state import BarState

        snapshot = BarState().snapshot()
        decoded = json.loads(snapshot)
        answers = self._ask_go([{"fn": "EncodeJSON", "args": [decoded, False]}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        # Key order cannot survive the json.loads round trip through Go's
        # generic map, so compare decoded values; the byte-level ordering
        # guarantee is what the struct-based state model will carry.
        self.assertEqual(json.loads(answers[0]["value"]), decoded)

    def test_encoder_float_formatting(self):
        values = [
            0.0, -0.0, 1.0, 100.0, 0.5, 1.5, -2.25, 0.1, 1e-4, 1e-5, 1e-7,
            1e15, 1e16, 1e17, 1.5e20, 123456789012345.0, 1234567890123456.0,
            3.141592653589793, 2.5e-10, 1e300, 5e-324,
        ]
        self._compare(
            "EncodeFloat",
            [(v, False) for v in values],
            lambda v, _a: json.dumps(v, separators=(",", ":")),
        )

    def test_nil_slice_encodes_as_empty_array_not_null(self):
        # eww.yuck runs nine (for ...) loops and eight arraylength() calls over
        # these fields, and its . index operator hard-errors on null. Python has
        # no nil slice to get this wrong with; Go does, and encoding/json would.
        answers = self._ask_go([{"fn": "EncodeNilSlice", "args": []}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        self.assertEqual(answers[0]["value"], '{"sinks":[],"groups":[]}')


    # -- impure collectors, both sides fed the same subprocess output ----------

    SEP = "\x1f"

    def _fixture_compare(self, fn, fixtures, python_fn, extra_args=()):
        """Run fn on both sides with each fixture standing in for run_text."""
        calls = [{"fn": fn, "args": [f, *extra_args]} for f in fixtures]
        answers = self._ask_go(calls)
        mismatches = []
        for fixture, answer in zip(fixtures, answers):
            self.assertTrue(answer["ok"], f"{fn} errored in Go: {answer.get('error')}")

            def fake_run_text(command, **_kwargs):
                return fixture.get(self.SEP.join(command), "")

            # network_connection_state reads /proc/net/wireless through
            # collectors.Path, with no run_text seam. Without this the Python
            # side reads the REAL file while Go reads the fixture, and the two
            # disagree for a reason that has nothing to do with either
            # implementation -- which is exactly how it failed the first time.
            fixture_files = {
                key[len("file") + len(self.SEP):]: value
                for key, value in fixture.items()
                if key.startswith("file" + self.SEP)
            }

            class FakePath:
                def __init__(self, path):
                    self._path = str(path)

                def read_text(self, *_a, **_k):
                    if self._path in fixture_files:
                        return fixture_files[self._path]
                    raise FileNotFoundError(self._path)

            patches = [
                unittest.mock.patch.object(collectors, "run_text", side_effect=fake_run_text),
                unittest.mock.patch.object(collectors, "Path", FakePath),
            ]
            for patcher in patches:
                patcher.start()
            try:
                collectors.reset_volume_sinks_cache()
                expected = python_fn()
            finally:
                for patcher in patches:
                    patcher.stop()
                collectors.reset_volume_sinks_cache()

            if answer["value"] != expected:
                mismatches.append((fixture, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(
                f"  fixture={list(f)!r}\n    python={p!r}\n    go    ={g!r}"
                for f, p, g in mismatches[:4]
            )
            self.fail(f"{len(mismatches)}/{len(fixtures)} disagreed:\n{detail}")

    def test_collect_active_window(self):
        key = self.SEP.join(["hyprctl", "activewindow", "-j"])
        fixtures = [
            {},
            {key: ""},
            {key: "not json"},
            {key: "{}"},
            {key: '{"class": "kitty", "title": "vim"}'},
            {key: '{"class": "zen", "title": "A page"}'},
            {key: '{"class": "unknown-app", "title": "t"}'},
            {key: '{"class": "code", "title": ""}'},
            {key: '{"class": "", "title": "only title"}'},
            {key: '{"class": "kitty", "title": "' + "x" * 90 + '"}'},
        ]
        self._fixture_compare(
            "CollectActiveWindow", fixtures, collectors.active_window_state
        )

    def test_collect_workspace(self):
        a = self.SEP.join(["hyprctl", "activeworkspace", "-j"])
        w = self.SEP.join(["hyprctl", "workspaces", "-j"])
        c = self.SEP.join(["hyprctl", "clients", "-j"])
        fixtures = [
            {},
            {a: '{"id": 2}', w: '[{"id": 2, "windows": 1}]', c: "[]"},
            {a: '{"id": 1}', w: '[{"id": 3, "windows": 4}]',
             c: '[{"urgent": true, "workspace": {"id": 3}}]'},
        ]
        self._fixture_compare("CollectWorkspace", fixtures, collectors.workspace_state)

    def test_collect_media(self):
        st = self.SEP.join(["playerctl", "status"])
        md = self.SEP.join(["playerctl", "metadata", "--format", "{{artist}} - {{title}}"])
        fixtures = [
            {},
            {st: "Playing\n", md: "Artist - Song\n"},
            {st: "Paused\n", md: "Artist - Song\n"},
            {st: "Stopped\n", md: "A - B\n"},
            {st: "Weird\n", md: "A - B\n"},
            {st: "Playing\n", md: "\n"},
        ]
        self._fixture_compare("CollectMedia", fixtures, collectors.media_state)

    def test_collect_tray_count(self):
        key = self.SEP.join([
            "busctl", "--user", "get-property",
            "org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
            "org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems",
        ])
        fixtures = [{}, {key: "as 0"}, {key: 'as 3 "a" "b" "c"'}, {key: "garbage"}]
        self._fixture_compare("CollectTrayCount", fixtures, collectors.tray_count)

    def test_collect_bluetooth(self):
        show = self.SEP.join(["bluetoothctl", "show"])
        devs = self.SEP.join(["bluetoothctl", "devices", "Connected"])
        info = lambda mac: self.SEP.join(["bluetoothctl", "info", mac])
        fixtures = [
            {},
            {show: "Controller AA:BB\n\tPowered: no\n", devs: ""},
            {show: "Controller AA:BB\n\tAlias: MyBT\n\tPowered: yes\n", devs: ""},
            {
                show: "Controller AA:BB\n\tAlias: MyBT\n\tPowered: yes\n",
                devs: "Device 80:99:E7 Buds\n",
                info("80:99:E7"): "\tAlias: Buds Pro\n\tBattery Percentage: 0x55 (85)\n",
            },
            {
                show: "Controller AA:BB\n\tPowered: yes\n",
                devs: "Device 11:22 Other\nDevice 33:44 ugreen_1 Set\n",
                info("11:22"): "\tAlias: Other Thing\n",
                info("33:44"): "\tAlias: UGreen Buds\n\tBattery Percentage: 0x32 (50)\n",
            },
        ]
        self._fixture_compare("CollectBluetooth", fixtures, collectors.bluetooth_state)

    def test_collect_volume(self):
        vol = self.SEP.join(["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"])
        dflt = self.SEP.join(["pactl", "get-default-sink"])
        js = self.SEP.join(["pactl", "-f", "json", "list", "sinks"])
        short = self.SEP.join(["pactl", "list", "short", "sinks"])
        fixtures = [
            {},
            {vol: "Volume: 0.42\n", dflt: "alsa_out\n",
             js: '[{"name":"alsa_out","description":"Speakers"}]'},
            {vol: "Volume: 0.42 [MUTED]\n", dflt: "bt\n",
             js: '[{"name":"alsa_out","description":"Speakers"},{"name":"bt","description":""}]'},
            # JSON empty -> the short-format fallback path.
            {vol: "Volume: 1.00\n", dflt: "alsa_out\n", js: "",
             short: "56\talsa_out\tPipeWire\ts32le\tRUNNING\n"},
            # A description past the 30-character truncation limit. Without
            # this no fixture was long enough to notice truncation at all.
            {vol: "Volume: 0.50\n", dflt: "alsa_out\n",
             js: '[{"name":"alsa_out",'
                 '"description":"Built-in Audio Analog Stereo (HDMI 2, rear panel)"}]'},
            # Second TAB field containing spaces. pactl's short format is
            # tab-separated; splitting on whitespace would take "alsa" here and
            # no other fixture distinguishes the two.
            {vol: "Volume: 0.50\n", dflt: "x\n", js: "",
             short: "56\talsa out device\tPipeWire\ts32le\tRUNNING\n"},
            # Too few fields, and an empty name field: both must be skipped.
            {vol: "Volume: 0.50\n", dflt: "x\n", js: "",
             short: "56\n56\t\tPipeWire\n"},
        ]
        for refresh in (True, False):
            with self.subTest(refresh_sinks=refresh):
                self._fixture_compare(
                    "CollectVolume",
                    fixtures,
                    lambda r=refresh: collectors.volume_state(refresh_sinks=r),
                    extra_args=(refresh,),
                )

    def test_volume_sink_cache_survives_a_refresh_false_call(self):
        """The cache is only observable across two calls.

        Single-call testing resets it every time, so refresh=False always took
        the cold path and a mutation removing the cache entirely changed
        nothing. Here the second call gets a fixture with no pactl output: if
        the cache works the sinks survive, if not they vanish.
        """
        vol = self.SEP.join(["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"])
        dflt = self.SEP.join(["pactl", "get-default-sink"])
        js = self.SEP.join(["pactl", "-f", "json", "list", "sinks"])
        first = {vol: "Volume: 0.42\n", dflt: "alsa_out\n",
                 js: '[{"name":"alsa_out","description":"Speakers"}]'}
        second = {vol: "Volume: 0.42\n"}  # no pactl at all

        answers = self._ask_go([{"fn": "CollectVolumeSequence", "args": [first, second]}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        go_first, go_second = answers[0]["value"]

        def fake(command, **_kw):
            return self._current.get(self.SEP.join(command), "")

        with unittest.mock.patch.object(collectors, "run_text", side_effect=fake):
            collectors.reset_volume_sinks_cache()
            self._current = first
            py_first = collectors.volume_state(refresh_sinks=True)
            self._current = second
            py_second = collectors.volume_state(refresh_sinks=False)
            collectors.reset_volume_sinks_cache()

        self.assertEqual(go_first, py_first)
        self.assertEqual(go_second, py_second)
        # And the property the cache exists for, asserted directly so this
        # cannot pass by both sides being equally broken.
        self.assertEqual(py_second["sinks"], py_first["sinks"])
        self.assertNotEqual(py_second["sinks"], [])

    def test_network_radio_enabled(self):
        key = self.SEP.join(["nmcli", "radio", "wifi"])
        fixtures = [{}, {key: "enabled\n"}, {key: "disabled\n"}, {key: " enabled "}, {key: "x"}]
        self._fixture_compare(
            "NetworkRadioEnabled", fixtures, collectors.network_radio_enabled
        )

    def test_collect_link_fallback(self):
        route = self.SEP.join(["ip", "route"])
        addr = self.SEP.join(["ip", "-o", "-4", "addr", "show"])
        fixtures = [
            {},
            {route: "default via 192.168.0.1 dev eno1 metric 100\n",
             addr: "2: eno1    inet 192.168.0.88/24 scope global eno1\n"},
            {route: "default via 192.168.0.1 dev eno1 metric 100\n"},
            {route: "192.168.0.0/24 dev eno1 scope link\n"},
        ]
        self._fixture_compare(
            "CollectLinkFallback", fixtures, collectors.link_fallback_state
        )

    def test_collect_network_connection(self):
        status = self.SEP.join(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
        route = self.SEP.join(["ip", "route"])
        addr = self.SEP.join(["ip", "-o", "-4", "addr", "show"])
        wifi = self.SEP.join(["nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi"])
        wshow = lambda d: self.SEP.join(
            ["nmcli", "-t", "-f", "GENERAL.CONNECTION,IP4.ADDRESS", "dev", "show", d])
        eshow = lambda d: self.SEP.join(["nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", d])
        wireless = "file" + self.SEP + "/proc/net/wireless"
        fixtures = [
            # Nothing connected and no route -> disconnected.
            {},
            # Ethernet with an address.
            {status: "eno1:ethernet:connected:Wired\n",
             eshow("eno1"): "IP4.ADDRESS[1]:192.168.0.88/24\n"},
            # Ethernet, linked but no address.
            {status: "eno1:ethernet:connected:Wired\n", eshow("eno1"): ""},
            # Wifi with /proc/net/wireless present.
            {status: "wlan0:wifi:connected:Net\n",
             wshow("wlan0"): "GENERAL.CONNECTION:MyNet\nIP4.ADDRESS[1]:10.0.0.5/24\n",
             wireless: " wlan0: 0000   49.  -40.  -256\n"},
            # Wifi, no /proc/net/wireless -> falls back to asking nmcli.
            {status: "wlan0:wifi:connected:Net\n",
             wshow("wlan0"): "GENERAL.CONNECTION:MyNet\nIP4.ADDRESS[1]:10.0.0.5/24\n",
             wifi: "yes:MyNet:72\n"},
            # Both up, ethernet owns the default route -> wifi is dropped.
            {status: "wlan0:wifi:connected:Net\neno1:ethernet:connected:Wired\n",
             route: "default via 192.168.0.1 dev eno1 metric 100\n",
             eshow("eno1"): "IP4.ADDRESS[1]:192.168.0.88/24\n"},
            # nmcli says nothing, but the kernel still has a route: degraded,
            # not disconnected. This is the dead-NetworkManager workaround.
            {route: "default via 192.168.0.1 dev eno1 metric 100\n",
             addr: "2: eno1    inet 192.168.0.88/24 scope global eno1\n"},
        ]
        self._fixture_compare(
            "CollectNetworkConnection", fixtures, collectors.network_connection_state
        )
        self._fixture_compare("CollectNetwork", fixtures, collectors.network_state)


    # -- file-reading collectors ----------------------------------------------

    def test_cpu_state_from_samples(self):
        def stat(user, nice, system, idle, iowait, irq, softirq):
            return f"cpu  {user} {nice} {system} {idle} {iowait} {irq} {softirq} 0 0 0\n"
        pairs = [
            ("", ""),
            ("garbage\n", "garbage\n"),
            ("cpu 1 2\n", "cpu 1 2\n"),
            (stat(100, 0, 50, 800, 50, 0, 0), stat(100, 0, 50, 800, 50, 0, 0)),   # no delta
            (stat(100, 0, 50, 800, 50, 0, 0), stat(200, 0, 100, 1600, 100, 0, 0)),
            (stat(0, 0, 0, 0, 0, 0, 0), stat(100, 0, 0, 0, 0, 0, 0)),             # 100%
            (stat(0, 0, 0, 0, 0, 0, 0), stat(0, 0, 0, 100, 0, 0, 0)),             # 0%
            (stat(0, 0, 0, 0, 0, 0, 0), stat(1, 0, 0, 1, 0, 0, 0)),               # 50%
            (stat(0, 0, 0, 0, 0, 0, 0), stat(1, 0, 0, 3, 0, 0, 0)),               # 25%
            (stat(200, 0, 100, 1600, 100, 0, 0), stat(100, 0, 50, 800, 50, 0, 0)), # backwards
        ]

        def python_cpu(first, second):
            # cpu_state reads /proc/stat twice around a sleep; feed it the two
            # samples through collectors.Path and neutralise the sleep.
            samples = iter([first, second])

            class FakePath:
                def __init__(self, _p): pass
                def read_text(self, *_a, **_k): return next(samples)

            with unittest.mock.patch.object(collectors, "Path", FakePath):
                with unittest.mock.patch.object(collectors.time, "sleep", lambda _s: None):
                    return collectors.cpu_state()

        self._compare("CPUStateFromSamples", pairs, python_cpu)

    def test_temperature_state_from_readings(self):
        readings = [
            [], [0], [45000], [45400], [45500], [45600], [89000], [89500], [90000],
            [30000, 62000, 41000], [95000, 20000], [-5000], [100499], [100500],
        ]

        def python_temp(values):
            # temperature_state globs /sys/class/hwmon; drive it with a fake
            # Path whose glob yields one entry per reading.
            class FakeInput:
                def __init__(self, value): self._value = value
                def read_text(self, *_a, **_k): return f"{self._value}\n"

            class FakePath:
                def __init__(self, _p): pass
                def glob(self, _pattern): return [FakeInput(v) for v in values]

            with unittest.mock.patch.object(collectors, "Path", FakePath):
                return collectors.temperature_state()

        self._compare(
            "TemperatureStateFromReadings", [(r,) for r in readings], python_temp
        )

    def test_battery_state_from_files(self):
        base = {"capacity": "72\n", "status": "Discharging\n"}
        energy = {"energy_now": "36000000", "power_now": "9000000",
                  "energy_full": "50000000", "energy_full_design": "60000000"}
        charge = {"charge_now": "3000000", "current_now": "900000",
                  "charge_full": "4000000", "charge_full_design": "5000000",
                  "voltage_now": "11000000"}
        cases = [
            {},                                        # no capacity -> default
            {"capacity": "notanumber"},                # unparsable -> default
            base,                                      # capacity only
            {**base, **energy},
            {**base, **charge},
            {**base, "status": "Charging\n", **energy},
            {**base, "status": "Charging\n", **charge},
            {**base, "status": "Full\n", **energy},
            {**base, "capacity": "15\n"},              # critical
            {**base, "capacity": "25\n"},              # warning
            {**base, "capacity": "5\n", **energy},
            # Every icon boundary, on both sides. Without 42 here, shifting the
            # 40%% threshold to 45%% changed no result and the mutation passed.
            {**base, "capacity": "19\n"}, {**base, "capacity": "20\n"},
            {**base, "capacity": "39\n"}, {**base, "capacity": "40\n"},
            {**base, "capacity": "42\n"}, {**base, "capacity": "44\n"},
            {**base, "capacity": "59\n"}, {**base, "capacity": "60\n"},
            {**base, "capacity": "79\n"}, {**base, "capacity": "80\n"},
            {**base, "capacity": "29\n"}, {**base, "capacity": "30\n"},
            {**base, "capacity": "35\n"}, {**base, "capacity": "55\n"},
            {**base, "capacity": "75\n"}, {**base, "capacity": "95\n"},
            # BOTH families present. The original tries energy_* first and stops,
            # so which one wins is observable only here -- no other fixture has
            # more than one, and swapping the order passed.
            {**base, **energy, **charge},
            {**base, "status": "Charging\n", **energy, **charge},
            {**base, "capacity": "100\n", "status": "Charging\n", **energy},
            {**base, **energy, "power_now": "0"},      # zero rate
            {**base, **energy, "energy_full_design": "0"},
            {**base, **charge, "voltage_now": "0"},
            {**base, "energy_now": "1", "power_now": "1"},  # incomplete family
        ]

        def python_battery(files):
            class FakeFile:
                def __init__(self, name, store): self._name, self._store = name, store
                def exists(self): return self._name in self._store
                def read_text(self, *_a, **_k):
                    if self._name not in self._store:
                        raise FileNotFoundError(self._name)
                    return self._store[self._name]

            class FakeBattery:
                def __init__(self, store): self._store = store
                def __truediv__(self, name): return FakeFile(name, self._store)

            class FakeRoot:
                def glob(self, _pattern): return [FakeBattery(files)]

            return collectors.battery_state(root=FakeRoot())

        self._compare("BatteryStateFromFiles", [(c,) for c in cases], python_battery)

    # -- BarState: the bytes eww reads before any collector has run ------------

    def test_default_snapshot_is_byte_identical(self):
        from eww_bar_backend.state import BarState

        answers = self._ask_go([{"fn": "DefaultSnapshot", "args": []}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        self.assertEqual(answers[0]["value"], BarState().snapshot())

    def test_default_snapshot_matches_the_yuck_initial_literal(self):
        """The Go snapshot must satisfy the same drift guard the Python does.

        test_state_defaults.py pins eww.yuck's :initial literal to
        BarState().snapshot(); this pins the Go side to the same literal, so the
        three cannot drift apart in any pairing.
        """
        first_line = (EWW_DIR / "eww.yuck").read_text().splitlines()[0]
        match = re.search(r":initial \'(.*?)\' \"eww-bar-backend bar\"\)", first_line)
        self.assertIsNotNone(match, "could not find the deflisten :initial literal")

        answers = self._ask_go([{"fn": "DefaultSnapshot", "args": []}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        self.assertEqual(json.loads(answers[0]["value"]), json.loads(match.group(1)))


    # -- AI usage: value probing and epoch parsing ----------------------------

    def test_py_float(self):
        """float() is not strconv.ParseFloat, in four directions at once."""
        texts = [
            # ordinary
            "0", "1", "-1", "1.5", "-1.5", ".5", "5.", "+1.5", "1e5", "1E5",
            "1e-5", "-0", "-0.0", "0.1", "1e308", "1e309", "-1e309",
            # whitespace: float() strips 25 of the 29 runes str.strip() does
            " 1.5 ", "\t2\n", "\r\n3\v\f", "\x1c4", "5\x1f", "\x854", "\xa05",
            " 1", "　1", "  2", "  ", "",
            # underscores, permitted only between digits
            "1_000", "1_0.5", "1e1_0", "_1", "1_", "1__0", "1._5", "1_.5",
            # hex: Go accepts the second, CPython accepts neither
            "0x10", "0x1p-2", "-0x1p-2", "0X1P2",
            # the special values, and their spellings
            "inf", "-inf", "+inf", "Inf", "INF", "infinity", "Infinity",
            "nan", "NaN", "-nan", "infi", "in",
            # rejected
            "1,5", "1e", "e5", "--1", "1 2", "12abc", "None", "null",
            # non-ASCII decimal digits
            "٣", "١٢٣", "１２３", "๓.๕", "٣_٤",
        ]
        self._compare("PyFloat", [(t,) for t in texts], _py_float_answer)

    def test_py_float_reads_every_unicode_decimal_digit_go_knows(self):
        """What licenses unicodeDigitValue having no lookup table.

        It assumes Go's Nd range table splits on decade boundaries, so a
        digit's value is its offset from the start of its range. This checks
        that assumption against unicodedata for every Nd rune -- the only way
        it could be wrong is a range starting mid-decade.

        Runes Go's tables do not carry are skipped rather than failed: CPython
        3.14 knows ~80 digits Go 1.26 does not (Garay at U+10D40, Sunuwar at
        U+11BF0), which is toolchain Unicode skew, not a port defect. The
        assertion that matters is that Go never reads a digit as the WRONG
        value, and the floor below keeps the skip list from quietly growing
        until the test proves nothing.
        """
        import unicodedata

        digits = [
            chr(c) for c in range(0x110000)
            if unicodedata.category(chr(c)) == "Nd"
        ]
        self.assertGreater(len(digits), 700, "sanity: Nd should be a large set")

        answers = self._ask_go([{"fn": "PyFloat", "args": [d]} for d in digits])
        read = {}
        wrong, known = [], 0
        for digit, answer in zip(digits, answers):
            self.assertTrue(answer["ok"], answer.get("error"))
            parsed, bits = answer["value"]
            read[digit] = bits if parsed else None
            if not parsed:
                continue  # not in Go's Nd table at this Unicode version
            known += 1
            if bits != _float_bits(float(unicodedata.decimal(digit))):
                wrong.append(digit)
        self.assertEqual(wrong, [], "a digit read as the wrong value")
        self.assertGreater(known, 600, "Go recognised too few digits to prove anything")

        # The skip above is for Unicode skew, but on its own it also hides a
        # broken decade calculation: dropping the modulo makes a second-decade
        # digit produce ':' and REJECT, which reads as "Go doesn't know it".
        # Only runs spanning more than one decade exercise the modulo at all --
        # there are two, and U+1D7CE (mathematical digits, 5 decades) has been
        # in Unicode since 3.1 -- so for any such run Go knows the start of, it
        # must read every digit in the run. That closes the hole without
        # pinning a Unicode version.
        exercised = 0
        for run in _contiguous_runs(digits):
            if len(run) <= 10 or read.get(run[0]) is None:
                continue
            exercised += 1
            for digit in run:
                self.assertIsNotNone(
                    read[digit],
                    f"Go knows {hex(ord(run[0]))} but rejected {hex(ord(digit))} in the same run",
                )
        self.assertGreater(exercised, 0, "no multi-decade run tested; the modulo is unproven")

    def test_py_lower_handles_the_one_code_point_go_gets_wrong(self):
        """U+0130 is the reason PyLower exists rather than strings.ToLower."""
        self._compare("PyLower", [("İ",), ("AİB",), ("İİ",)],
                      str.lower)

    def test_py_lower_matches_str_lower_over_every_cased_code_point(self):
        """Exhaustive, but asserting an invariant that survives Unicode skew.

        A plain equality check here fails on ~27 code points, and none of them
        are a port defect: CPython 3.14 ships newer Unicode tables than Go
        1.26, so it knows casing for scripts (Garay at U+10D50, and a handful
        of Latin additions) that Go's tables do not. Pinning the exact set
        would just break on the next toolchain bump in either direction.

        So the assertion is the property that holds regardless of which side is
        newer: wherever the two disagree, one of them left the rune ALONE. Go
        emitting a genuinely DIFFERENT lowercase -- a real table bug, or
        PyLower's U+0130 substitution going wrong -- still fails.
        """
        cased = [chr(c) for c in range(0x110000) if chr(c).lower() != chr(c)]
        self.assertGreater(len(cased), 1000, "sanity: expected many cased runes")

        answers = self._ask_go([{"fn": "PyLower", "args": [c]} for c in cased])
        wrong = []
        for char, answer in zip(cased, answers):
            self.assertTrue(answer["ok"], answer.get("error"))
            if answer["value"] == char.lower():
                continue
            if answer["value"] == char:
                continue  # Go's tables do not know this rune is cased
            wrong.append((hex(ord(char)), char.lower(), answer["value"]))
        self.assertEqual(wrong, [], "Go produced a different lowercase, not a missing one")

    def _iso_corpus(self):
        dates = [
            "2024-01-15", "20240115", "2024-W03-1", "2024W031", "2024-W03",
            "2024W03", "2020-W53-7", "2024-W53-7", "2023-02-28", "2024-02-29",
            "2023-02-29", "2024-12-31", "2024-00-10", "2024-13-01", "2024-01-00",
            "2024-1-5", "2024-01", "2024", "2024-366",
            # Extremes, but deliberately NOT 0001-01-01 or 9999-12-31. Within
            # one UTC offset of datetime.min/max, CPython's .timestamp() raises
            # for a NAIVE value and parse_iso_epoch silently drops to its
            # strptime leg -- so the expected answer there depends on the host
            # timezone, and the case would assert different things on this
            # laptop (UTC+8) and in the flake sandbox (UTC). See isotime.go.
            "0002-01-01", "9998-12-31",
        ]
        separators = ["T", "t", " ", "X", "\t", ""]
        times = [
            "", "10", "10:30", "1030", "10:30:00", "103000", "24:00:00",
            "24:00:01", "24:30", "23:59:59", "10:30:00.5", "10:30:00.123456",
            "10:30:00.1234567891", "10:30:00,25", "10:30:00.", "10:30:60",
            "10:3", "10:30:0", "10:3000", "1030:00",
        ]
        zones = [
            "", "Z", "+00:00", "+0000", "+00", "-08:00", "+05:30", "+23:59:59",
            "+24:00", "+00:00:30", "-00:00", "+", "+1", "+123",
            "+23:59:59.999999",
        ]
        cases = []
        for date in dates:
            for sep in separators:
                for clock in times:
                    for zone in zones:
                        if not clock and zone:
                            continue  # a zone with no time is a separate axis
                        cases.append((date + sep + clock + zone,))
        return cases

    def test_parse_iso_epoch_over_the_grammar(self):
        """The generated cross product of every date, time and zone form.

        fromisoformat is far wider than RFC 3339 -- any separator character,
        week dates, T24:00:00, sub-minute offsets -- and narrower in other
        places. Hand-picked cases would only pin the forms someone remembered.

        One shape is exempt, and the exemption is deliberately narrow: a BASIC
        week date whose weekday digit is followed by another digit. CPython
        resolves that ambiguity with an undocumented separator-position rule
        that isn't derivable from the string, so parseISOWeekDate rejects it
        instead of guessing. The exemption only permits Go to answer None --
        a Go answer that differs from Python in any other way still fails, so
        Go can never invent an instant CPython would not produce.
        """
        ambiguous = re.compile(r"^\d{4}W\d{2}\d\d")
        cases = self._iso_corpus()
        answers = self._ask_go([{"fn": "ParseISOEpoch", "args": list(a)} for a in cases])

        mismatches, exempt = [], 0
        for (text,), answer in zip(cases, answers):
            self.assertTrue(answer["ok"], answer.get("error"))
            expected = collectors.parse_iso_epoch(text)
            if answer["value"] == expected:
                continue
            if answer["value"] is None and ambiguous.match(text):
                exempt += 1
                continue
            mismatches.append((text, expected, answer["value"]))

        detail = "\n".join(f"  {t!r}: python={p!r} go={g!r}" for t, p, g in mismatches[:20])
        self.assertEqual(mismatches, [], f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")
        self.assertGreater(exempt, 0, "the exemption is dead; drop it and the narrowing in isotime.go")

    def test_parse_iso_epoch_on_non_strings_and_call_site_shapes(self):
        cases = [
            (None,), (0,), (1,), (1.5,), (True,), ([],), ({},), ("",),
            # The shape the ccusage period path builds, including its failures.
            ("2024-01-15T00:00:00",), ("T00:00:00",), ("garbageT00:00:00",),
            ("2024-01-15T00:00:00Z",),
            # The strptime fallback's territory: unpadded fields fromisoformat
            # rejects outright.
            ("2024-1-5T1:2:3",), ("2024-01-15T1:2:3",), ("2024-1-5T01:02:03",),
            ("2024-02-30T00:00:00",), ("2024-01-15T10:30:61",),
            ("2024-01-15T10:30:62",), ("2024-01-15T25:00:00",),
            # Longer than 19 characters, so the fallback slices.
            ("2024-1-5T1:2:3 trailing junk",), ("2024-01-15T10:30:00extra",),
            # Z is replaced globally, not just at the end.
            ("Z2024-01-15",), ("2024Z01Z15",),
        ]
        self._compare("ParseISOEpoch", cases, collectors.parse_iso_epoch)

    def test_format_clock_time(self):
        cases = [
            (None,), (0,), (1,), (-1,), (1.9,), (-0.5,), (1705285800,),
            (1705285800.9,), (1705285800.123456,), (2**31,), (-2**31,),
        ]
        self._compare("FormatClockTime", cases, collectors.format_clock_time)

    def test_number_value(self):
        """The bool guard, the string fallback, and key precedence."""
        rows = [
            {}, {"a": 1}, {"a": 1.5}, {"a": -2}, {"a": 0},
            {"a": True}, {"a": False}, {"a": True, "b": 7},
            {"a": "12"}, {"a": "1.5"}, {"a": " 3 "}, {"a": "abc"},
            {"a": "abc", "b": 4}, {"a": None}, {"a": None, "b": 5},
            {"a": []}, {"a": {}}, {"a": [], "b": 6},
            {"a": "1e999"}, {"a": 1e308}, {"a": "0x10"}, {"a": "1_000"},
            {"b": 1}, {"a": "", "b": 2},
            {"totalTokens": 5, "total_tokens": 9},
            {"total_tokens": 9}, {"tokens": "11"},
        ]
        key_sets = [["a"], ["a", "b"], ["b", "a"], ["missing"],
                    ["totalTokens", "total_tokens", "tokens"]]
        cases = [(row, keys) for row in rows for keys in key_sets]
        self._compare(
            "NumberValue", cases,
            lambda row, keys: _float_bits(collectors.number_value(row, *keys)),
        )

    def test_list_value(self):
        rows = [
            {}, {"a": []}, {"a": [1, 2]}, {"a": "not a list"}, {"a": {}},
            {"a": None}, {"a": None, "b": [3]}, {"b": ["x"]},
            {"daily": [{"date": "2024-01-15"}]}, {"data": [1]}, {"rows": [2]},
        ]
        key_sets = [["a"], ["a", "b"], ["daily", "data", "days", "rows"]]
        cases = [(row, keys) for row in rows for keys in key_sets]
        self._compare(
            "ListValue", cases, lambda row, keys: collectors.list_value(row, *keys)
        )

    def test_daily_token_values(self):
        rows = [
            {},
            {"inputTokens": 10, "outputTokens": 20},
            {"input_tokens": 10, "output_tokens": 20},
            {"input": 1, "output": 2, "cache_creation_tokens": 3,
             "cache_read_tokens": 4},
            {"cacheCreationTokens": 5, "cacheReadTokens": 6},
            {"cacheCreationInputTokens": 5, "cacheReadInputTokens": 6},
            # totalTokens present but zero: the fallback sum must kick in.
            {"totalTokens": 0, "inputTokens": 3, "outputTokens": 4},
            {"totalTokens": -1, "inputTokens": 3},
            {"totalTokens": 99, "inputTokens": 3},
            {"totalCost": 1.25}, {"total_cost": 1.25}, {"costUSD": 1.25},
            {"cost": 1.25}, {"totalCost": 0, "cost": 9},
            {"inputTokens": "7", "outputTokens": True},
            {"tokens": 12, "totalTokens": 0},
        ]
        self._compare(
            "DailyTokenValues", [(row,) for row in rows],
            collectors.daily_token_values,
        )

    def test_agents_text(self):
        cases = [
            ([],), (["claude"],), (["codex"],), (["gemini"],),
            (["gemini", "claude"],), (["claude", "claude"],),
            (["anthropic.claude"],), (["CLAUDE"],), (["opencode"],),
            (["claude", "codex", "gemini"],), (["cod"],), (["codexx"],),
            (["", "claude"],), ([""],),
        ]
        self._compare("AgentsText", cases, collectors.agents_text)

    def test_period_agent_keys(self):
        rows = [
            {},
            {"metadata": {"agents": ["Claude", "codex"]}},
            {"metadata": {"agents": []}},
            {"metadata": {"agents": ["a", 1, None, "b"]}},
            {"metadata": {}},
            {"metadata": "not a dict"},
            {"metadata": {"agents": "abc"}},
            {"agents": [{"agent": "claude"}, {"agent": "all"}]},
            {"agents": [{"agent": "Codex"}, {"noagent": 1}, "junk"]},
            {"agents": "not a list"},
            {"metadata": {"agents": ["X"]}, "agents": [{"agent": "Y"}]},
            {"agents": [{"agent": "İstanbul"}]},
        ]
        self._compare(
            "PeriodAgentKeys", [(row,) for row in rows],
            collectors.period_agent_keys,
        )


    # -- AI usage: the provider quota cards -----------------------------------

    def test_py_title(self):
        """str.title() restarts a word at any uncased character."""
        texts = [
            "max", "pro", "max 5x", "MAX 20X", "a b", "x1y", "don't", "",
            "  ", "5x", "x5", "a1b2c3", "ǅungla", "straße", "ǆ", "ﬁne",
            "İstanbul", "ÅNGSTRÖM", "ǰ", "Ǆǅǆ",
            # U+0130 mid-word, i.e. reached by the LOWERCASE branch rather
            # than the titlecase one -- a different mapping and a different bug.
            "aİ", "xİy", "AİB", "ﬁ", "aﬁ", "ßx", "aß",
        ]
        self._compare("PyTitle", [(t,) for t in texts], str.title)

    def test_py_title_matches_str_title_over_every_cased_code_point(self):
        """Exhaustive, with the same Unicode-skew invariant as PyLower.

        This is what proves the generated fullTitleCase table is complete:
        every code point whose titlecase is longer than one character has to be
        in it, and unicode.ToTitle cannot express any of them.
        """
        cased = [chr(c) for c in range(0x110000) if chr(c).title() != chr(c)]
        self.assertGreater(len(cased), 1000, "sanity: expected many cased runes")

        answers = self._ask_go([{"fn": "PyTitle", "args": [c]} for c in cased])
        wrong = []
        for char, answer in zip(cased, answers):
            self.assertTrue(answer["ok"], answer.get("error"))
            if answer["value"] in (char.title(), char):
                continue  # equal, or a rune Go's older tables call uncased
            wrong.append((hex(ord(char)), char.title(), answer["value"]))
        self.assertEqual(wrong, [], "Go produced a different titlecase, not a missing one")

        # The skip above tolerates version skew, so assert separately that every
        # multi-character mapping is handled -- those are the table's whole
        # reason to exist, and none of them may fall through to ToTitle.
        expansions = [c for c in cased if len(c.title()) > 1]
        self.assertGreater(len(expansions), 40, "sanity: expected ~48 expansions")
        answers = self._ask_go([{"fn": "PyTitle", "args": [c]} for c in expansions])
        for char, answer in zip(expansions, answers):
            self.assertEqual(answer["value"], char.title(),
                             f"missing fullTitleCase entry for {hex(ord(char))}")

    def test_quota_window_class(self):
        cases = [(p, r) for p in (-5, 0, 1, 79, 80, 89, 90, 91, 100, 150)
                 for r in (True, False)]
        self._compare("QuotaWindowClass", cases, collectors.quota_window_class)

    def test_quota_card_class(self):
        def card(window_classes, meta_classes):
            return {
                "key": "codex", "name": "Codex", "plan": "--", "status": "live",
                "class": "", "updated": "",
                "windows": [{"label": "w", "percent": 0, "value": "0%",
                             "remaining": "--", "reset": "--", "class": c}
                            for c in window_classes],
                "meta": [{"label": "m", "value": "1", "tooltip": "", "class": c}
                         for c in meta_classes],
            }
        combos = [
            ([], []), (["active"], []), ([], ["active"]),
            (["empty"], []), (["missing"], []),
            (["active", "warning"], []), (["warning", "active"], []),
            (["critical", "warning", "active", "empty"], []),
            (["empty"], ["warning"]), (["missing"], ["critical"]),
            (["unknown"], []), (["empty", "missing"], []),
        ]
        cases = [(card(w, m),) for w, m in combos]
        self._compare("QuotaCardClass", cases, collectors.quota_card_class)

    def test_quota_default(self):
        cases = [
            ("claude", "Claude", "waiting"), ("codex", "Codex", "live"),
            ("antigravity", "Antigravity", "missing"),
            # A known key ignores the name it is handed.
            ("claude", "Not Claude", "waiting"),
            # An unknown key falls through to the generic card.
            ("gemini", "Gemini", "waiting"), ("", "", "x"),
        ]
        self._compare("QuotaDefault", cases,
                      lambda k, n, s: collectors.quota_default(k, n, s))

    _NOW = 1705285800.0  # 2024-01-15T10:30:00 local

    # Exactly midnight, i.e. exactly one week after a "2024-01-08" row starts.
    # Computed rather than hardcoded because every period comparison runs in
    # local time, and the sandbox is UTC while this laptop is not.
    _NOW_WEEK_EDGE = time.mktime(time.strptime("2024-01-15T00:00:00", "%Y-%m-%dT%H:%M:%S"))

    def _openusage_lines(self):
        return [
            {}, {"used": 50, "limit": 100},
            {"used": 0, "limit": 100}, {"used": 100, "limit": 100},
            {"used": 95, "limit": 100}, {"used": 85, "limit": 100},
            {"used": 1, "limit": 0}, {"used": "50", "limit": "100"},
            {"used": True, "limit": 100},
            {"label": "GPT-5", "used": 10, "limit": 100},
            {"label": "", "used": 10, "limit": 100},
            {"label": None, "used": 10, "limit": 100},
            {"label": 0, "used": 10, "limit": 100},
            {"label": 42, "used": 10, "limit": 100},
            {"label": "a-very-long-window-label-past-the-cap", "used": 10, "limit": 100},
            # resetsAt in the future, the past, malformed, and absent
            {"used": 10, "limit": 100, "resetsAt": "2024-01-15T12:00:00"},
            {"used": 10, "limit": 100, "resetsAt": "2024-01-15T09:00:00"},
            {"used": 10, "limit": 100, "resetsAt": "2024-01-15T10:30:00"},
            {"used": 0, "limit": 100, "resetsAt": "2024-01-16T00:00:00Z"},
            {"used": 10, "limit": 100, "resetsAt": "garbage"},
            {"used": 10, "limit": 100, "resetsAt": None},
            {"used": 10, "limit": 100, "resetsAt": 1705285800},
            {"used": 0, "limit": 0, "resetsAt": "2024-01-15T12:00:00"},
            # Epoch zero and before it: parsed, but not a usable reset time.
            {"used": 10, "limit": 100, "resetsAt": "1970-01-01T00:00:00+00:00"},
            {"used": 10, "limit": 100, "resetsAt": "1969-01-01T00:00:00+00:00"},
            {"used": 0, "limit": 100, "resetsAt": "1970-01-01T00:00:00+00:00"},
        ]

    def test_openusage_window(self):
        cases = [(line, self._NOW) for line in self._openusage_lines()]
        self._compare(
            "OpenusageWindow", cases,
            lambda line, now: collectors.openusage_window(line, now_epoch=now),
        )

    def test_openusage_meta(self):
        lines = [
            {}, {"limit": 100, "used": 40},
            {"limit": 100, "used": 100}, {"limit": 100, "used": 150},
            {"limit": 0, "used": 0},
            # int() truncates each operand before subtracting.
            {"limit": 1.9, "used": 0.9}, {"limit": 2.9, "used": 1.1},
            {"limit": 100, "used": 40, "format": {"suffix": "credits"}},
            {"limit": 100, "used": 40, "format": {"suffix": ""}},
            {"limit": 100, "used": 40, "format": {"suffix": 7}},
            {"limit": 100, "used": 40, "format": "not a dict"},
            {"limit": 100, "used": 40, "format": {}},
            {"label": "Balance", "limit": 100, "used": 40},
            {"label": "", "limit": 100, "used": 40},
            {"label": None, "limit": 100, "used": 40},
        ]
        self._compare("OpenusageMeta", [(line,) for line in lines],
                      collectors.openusage_meta)

    def _openusage_snapshots(self):
        def progress(label, used, limit, kind="percent", **extra):
            line = {"type": "progress", "label": label, "used": used,
                    "limit": limit, "format": {"kind": kind}}
            line.update(extra)
            return line

        return [
            None, "not a dict", 42, [],
            {}, {"lines": []}, {"lines": "not a list"},
            {"lines": [{"type": "text", "label": "Error", "value": "boom"}]},
            {"lines": [{"type": "text", "label": "Error", "value": ""}]},
            {"lines": [{"type": "text", "label": "Error"}]},
            {"lines": [{"type": "text", "label": "Error",
                        "value": "a failure message far longer than the forty-eight character cap"}]},
            {"lines": [{"type": "text", "label": "Note", "value": "hi"}]},
            {"lines": [progress("Session", 50, 100)]},
            {"lines": [progress("Session", 50, 100), progress("Credits", 10, 100, kind="count")]},
            {"lines": [progress("Credits", 100, 100, kind="count")]},
            {"lines": [progress("Session", 95, 100)], "plan": "Pro"},
            {"lines": [progress("Session", 95, 100)], "plan": "  Pro  "},
            {"lines": [progress("Session", 95, 100)], "plan": "   "},
            {"lines": [progress("Session", 95, 100)], "plan": 7},
            {"lines": [progress("S", 50, 100)], "plan": None},
            # More than the cap: sorted by -percent then lowercased label.
            {"lines": [progress(f"W{i}", i * 10, 100) for i in range(6)]},
            # Ties on percent, so the label tiebreak decides. The names are
            # chosen so case-folding CHANGES the order: sorted raw, "ZULU"
            # precedes "beta" because 'Z' < 'b' in code points.
            {"lines": [progress(n, 50, 100) for n in ["beta", "Alpha", "ZULU", "charlie", "delta"]]},
            {"lines": [progress(n, 50, 100) for n in ["delta", "Alpha", "charlie", "BRAVO", "echo"]]},
            # Fully tied on BOTH sort keys, so only stability decides which
            # four survive the cap. Twenty of them: Go's pdqsort falls back to
            # insertion sort (which is stable) below about a dozen elements, so
            # a smaller list would pass even with an unstable sort.
            {"lines": [progress("same", 50, 100, resetsAt=f"2024-01-{d:02d}T12:00:00")
                       for d in range(1, 21)]},
            # ...and the same idea INTERLEAVED. All-equal keys are not enough on
            # their own: pdqsort notices the slice is already ordered and
            # returns without touching it, so an unstable sort looks stable.
            # Four tie groups in rotating order force real partitioning, and the
            # top group has five members for a cap of four, so which four
            # survive -- and in what order -- is decided purely by stability.
            {"lines": [progress(f"g{i % 4}", (i % 4) * 25, 100,
                                resetsAt=f"2024-02-{i + 1:02d}T12:00:00")
                       for i in range(20)]},
            # Meta rows but no windows: the only shape that tells
            # "no windows and no meta" apart from "no windows".
            {"lines": [progress("Credits", 10, 100, kind="count")]},
            {"lines": [progress("Credits", 10, 100, kind="count"),
                       progress("Bonus", 5, 50, kind="count")]},
            # Exactly at the cap: order must be left alone, not sorted.
            {"lines": [progress(n, 10, 100) for n in ["d", "c", "b", "a"]]},
            {"lines": [progress("S", 50, 100, resetsAt="2024-01-15T12:00:00")]},
        ]

    def test_quota_from_openusage(self):
        cases = [(snap, key, name, self._NOW)
                 for snap in self._openusage_snapshots()
                 for key, name in (("codex", "Codex"), ("gemini", "Gemini"))]
        self._compare(
            "QuotaFromOpenusage", cases,
            lambda s, k, n, now: collectors.quota_from_openusage(s, k, n, now_epoch=now),
        )

    def test_format_claude_plan(self):
        values = [
            "max", "pro", "max_5x", "max_20x", "MAX_5X", "", "   ", "_",
            "a_b_c", "free", None, 7, True, [], {}, "ǅungla", "don't_stop",
        ]
        self._compare("FormatClaudePlan", [(v,) for v in values],
                      collectors.format_claude_plan)

    def test_claude_window_state(self):
        windows = [
            None, "not a dict", 42, [],
            {}, {"utilization": 50}, {"used_percent": 50}, {"percent": 50},
            {"utilization": 0}, {"utilization": 100}, {"utilization": 150},
            {"utilization": -10}, {"utilization": 79.5}, {"utilization": 80.5},
            {"utilization": "50"},
            # All three spellings present: only their order decides the answer.
            {"utilization": 10, "used_percent": 50, "percent": 90},
            {"used_percent": 50, "percent": 90},
            {"utilization": 0, "used_percent": 50},
            {"utilization": 50, "resets_at": "2024-01-15T12:00:00"},
            {"utilization": 50, "resets_at": "2024-01-15T09:00:00"},
            {"utilization": 50, "reset_at": "2024-01-15T12:00:00"},
            {"utilization": 50, "resets_at": "", "reset_at": "2024-01-15T12:00:00"},
            {"utilization": 50, "resets_at": None, "reset_at": 1705290000},
            # The old shape: a bare epoch, reached only when the ISO parse fails.
            {"utilization": 50, "resets_at": 1705290000},
            {"utilization": 50, "resets_at": 0},
            {"utilization": 50, "resets_at": -5},
            {"utilization": 0, "resets_at": "2024-01-15T12:00:00"},
        ]
        cases = [("Session", w, self._NOW) for w in windows]
        self._compare(
            "ClaudeWindowState", cases,
            lambda label, w, now: collectors.claude_window_state(label, w, now_epoch=now),
        )

    def test_claude_quota_state_from_json(self):
        bodies = [
            "", "null", "[]", "{}", "not json", "42", '{"unrelated": 1}',
            '{"five_hour": {"utilization": 50}}',
            '{"fiveHour": {"utilization": 50}}',
            # Both spellings: the snake_case one wins.
            '{"five_hour": {"utilization": 10}, "fiveHour": {"utilization": 90}}',
            '{"seven_day": {"utilization": 50}}',
            '{"seven_day_opus": {"utilization": 95}}',
            '{"seven_day_sonnet": {"utilization": 85}}',
            '{"five_hour": {"utilization": 50}, "seven_day": {"utilization": 85},'
            ' "seven_day_opus": {"utilization": 95}, "seven_day_sonnet": {"utilization": 5}}',
            '{"five_hour": "not a dict"}',
            '{"five_hour": {"utilization": 50, "resets_at": "2024-01-15T12:00:00"}}',
        ]
        plans = ["--", "max_5x", None, 7]
        cases = [(body, plan, self._NOW) for body in bodies for plan in plans]
        self._compare(
            "ClaudeQuotaStateFromJSON", cases,
            lambda body, plan, now: collectors.claude_quota_state_from_json(
                body, plan=plan, now_epoch=now),
        )


    # -- AI usage: periods and assembly ---------------------------------------

    def test_agent_display_name(self):
        keys = [
            "claude", "codex", "gemini", "opencode", "copilot", "amp", "droid",
            "unknown", "my_agent", "my_5x_agent", "MY_AGENT", "", None, 7,
            True, [], "a", "don't", "ǅungla",
        ]
        self._compare("AgentDisplayName", [(k,) for k in keys],
                      collectors.agent_display_name)

    def _period_rows(self):
        """Rows around the reference instant, plus the shapes that break parsing."""
        return [
            [],
            [{"date": "2024-01-15"}],
            [{"date": "2024-01-14"}],
            [{"period": "2024-01-15"}],
            [{"period": "2024-01"}],
            [{"period": "2023-12"}],
            [{"period": "2024-01-08"}],  # the week containing the reference day
            [{"period": "2024-01-15"}, {"period": "2024-01-08"}],
            [{"period": "2024-01-01"}, {"period": "2024-01-08"}],
            # Two starts two days apart, so they sit on different week phases:
            # picking the latest and picking the earliest step to different
            # keys, where starts a whole number of weeks apart converge.
            [{"period": "2024-01-08"}, {"period": "2024-01-10"}],
            [{"period": "2024-01-10"}, {"period": "2024-01-08"}],
            [{"period": ""}], [{"period": None}], [{"period": 7}],
            [{"date": None, "period": None}],
            ["not a dict"], ["not a dict", {"date": "2024-01-15"}],
            [{}, {"date": "2024-01-15"}],
            [{"period": "garbage"}],
            # Week starts far in the past: the stepping loop has to run.
            [{"period": "2023-01-02"}],
            [{"period": "2023-01-02"}, {"period": "2023-01-09"}],
        ]

    def test_current_period_key(self):
        cases = [(kind, [r for r in rows if isinstance(r, dict)], now)
                 for now in (self._NOW, self._NOW_WEEK_EDGE)
                 for kind in ("daily", "weekly", "monthly")
                 for rows in self._period_rows()]
        self._compare("CurrentPeriodKey", cases, collectors.current_period_key)

    def test_synthetic_period_row(self):
        cases = [(kind, [r for r in rows if isinstance(r, dict)], self._NOW)
                 for kind in ("daily", "weekly", "monthly")
                 for rows in self._period_rows()]
        self._compare("SyntheticPeriodRow", cases, collectors.synthetic_period_row)

    def test_select_period_row(self):
        cases = [(rows, kind, now)
                 for now in (self._NOW, self._NOW_WEEK_EDGE)
                 for kind in ("daily", "weekly", "monthly")
                 for rows in self._period_rows()]
        self._compare(
            "SelectPeriodRow", cases,
            lambda rows, kind, now: collectors.select_period_row(rows, kind, now_epoch=now),
        )

    def test_period_range_label(self):
        periods = [
            "", "2024-01-15", "2024-01", "2024-1", "2024-12", "2024-13",
            "2024-00", "2024-01-01", "2024-12-31", "2023-02-28", "2024-02-29",
            "garbage", "2024", "2024-01-15T00:00:00", "0", "2024-01-08",
        ]
        cases = [(kind, period, self._NOW)
                 for kind in ("daily", "weekly", "monthly")
                 for period in periods]
        self._compare(
            "PeriodRangeLabel", cases,
            lambda kind, period, now: collectors.period_range_label(kind, period, now_epoch=now),
        )

    def test_period_range_label_leaks_a_non_string_month(self):
        """Pins the one divergence PeriodRangeLabel documents.

        For kind "monthly" with a non-string period, strptime raises TypeError
        and the except branch returns the value UNCHANGED, so a number reaches
        the state as a JSON number where every other path yields a string. Go
        stringifies instead. Asserted here rather than reproduced, so the claim
        in period.go cannot quietly stop being true.
        """
        for period in (2024, 7.5, True):
            self.assertEqual(collectors.period_range_label("monthly", period), period)
            self.assertNotIsInstance(
                collectors.period_range_label("monthly", period), str,
                "if this becomes a str, drop the divergence note in period.go",
            )
        # Every other kind already stringifies, which is why only monthly differs.
        self.assertIsInstance(collectors.period_range_label("daily", 2024), str)

    def _agent_rows(self):
        def agent(key, tokens=0, cost=0.0):
            return {"agent": key, "totalTokens": tokens, "totalCost": cost}

        return [
            {}, {"agents": []}, {"agents": "not a list"},
            {"agents": [agent("claude", 100)]},
            {"agents": [agent("claude", 100), agent("codex", 200)]},
            {"agents": [agent("codex", 200), agent("claude", 100)]},
            # "all" is the rollup row and is skipped.
            {"agents": [agent("all", 300), agent("claude", 100)]},
            # Zero on both counts is dropped; either one alone keeps the row.
            {"agents": [agent("empty", 0, 0.0), agent("claude", 100)]},
            {"agents": [agent("costonly", 0, 1.5)]},
            {"agents": [agent("tokensonly", 5, 0.0)]},
            {"agents": [agent("neg", -5, -1.0)]},
            {"agents": [{"noagent": 1}, "junk", agent("claude", 100)]},
            {"agents": [{"agent": 7, "totalTokens": 5}]},
            # totalTokens absent or zero: the percentages come off the sum.
            {"agents": [agent("a", 30), agent("b", 70)]},
            {"totalTokens": 0, "agents": [agent("a", 30), agent("b", 70)]},
            {"totalTokens": 1000, "agents": [agent("a", 30), agent("b", 70)]},
            {"total_tokens": 200, "agents": [agent("a", 30)]},
            # Ties on the sort key, so only stability decides the order.
            {"agents": [agent(n, 50) for n in ["delta", "alpha", "charlie"]]},
            {"agents": [agent(f"a{i}", (i % 3) * 10 + 1) for i in range(15)]},
            # Mixed spellings and casing.
            {"agents": [{"agent": "CLAUDE", "tokens": 5, "cost": 0.5}]},
            {"agents": [{"agent": "My_Agent", "total_tokens": 5}]},
            {"agents": [agent("big", 1_500_000), agent("small", 999)]},
        ]

    def test_period_agents(self):
        self._compare("PeriodAgents", [(row,) for row in self._agent_rows()],
                      collectors.period_agents)

    def test_period_state(self):
        rowsets = self._period_rows() + [
            [{"date": "2024-01-15", "totalTokens": 1234, "totalCost": 5.5,
              "agents": [{"agent": "claude", "totalTokens": 1234, "totalCost": 5.5}]}],
            [{"date": "2024-01-15", "inputTokens": 10, "outputTokens": 20,
              "cacheCreationTokens": 5, "cacheReadTokens": 5}],
            [{"period": "2024-01", "totalTokens": 999}],
            [{"period": "2024-01-08", "totalTokens": 42}],
        ]
        cases = [(rows, kind, label, self._NOW)
                 for kind, label in (("daily", "Today"), ("weekly", "This week"),
                                     ("monthly", "This month"))
                 for rows in rowsets]
        self._compare(
            "PeriodState", cases,
            lambda rows, kind, label, now: collectors.period_state(
                rows, kind, label, now_epoch=now),
        )

    def _quota_sets(self):
        def card(key, name, status, klass, plan="--", windows=()):
            return {
                "key": key, "name": name, "plan": plan, "status": status,
                "class": klass, "updated": "",
                "windows": [{"label": lbl, "percent": pct, "value": val,
                             "remaining": "--", "reset": "--", "class": "active"}
                            for lbl, pct, val in windows],
                "meta": [],
            }

        return [
            [],
            [card("claude", "Claude", "waiting", "missing")],
            [card("claude", "Claude", "live", "active")],
            [card("claude", "Claude", "live", "warning")],
            [card("claude", "Claude", "live", "critical")],
            [card("claude", "Claude", "live", "active", plan="Max")],
            [card("claude", "Claude", "live", "active", plan="")],
            [card("claude", "Claude", "live", "active", plan="--")],
            [card("claude", "Claude", "live", "active", plan="Max",
                  windows=[("Session", 50, "50%")])],
            # More than two windows: only the first two reach the tooltip.
            [card("claude", "Claude", "live", "active", plan="Max",
                  windows=[("A", 10, "10%"), ("B", 20, "20%"), ("C", 30, "30%")])],
            # A "--" value is skipped inside the head line.
            [card("claude", "Claude", "live", "active",
                  windows=[("A", 0, "--"), ("B", 20, "20%")])],
            [card("claude", "Claude", "live", "active"),
             card("codex", "Codex", "live", "warning")],
            [card("claude", "Claude", "unavailable", "missing"),
             card("codex", "Codex", "live", "active")],
        ]

    def test_apply_quotas(self):
        import copy

        from eww_bar_backend.common import AI_USAGE_DEFAULT

        states = [
            copy.deepcopy(AI_USAGE_DEFAULT),
            collectors.ai_usage_state_from_json(
                json.dumps({"daily": [{"date": "2024-01-15", "totalTokens": 1234,
                                       "totalCost": 5.5}]}),
                now_epoch=self._NOW),
            collectors.ai_usage_state_from_json(
                json.dumps({"daily": [{"date": "2024-01-14"}]}), now_epoch=self._NOW),
        ]
        cases = [(state, quotas) for state in states for quotas in self._quota_sets()]
        self._compare(
            "ApplyQuotas", cases,
            lambda state, quotas: collectors.apply_quotas(
                copy.deepcopy(state), copy.deepcopy(quotas)),
        )

    def test_ai_usage_state_from_json(self):
        reports = [
            "", "null", "[]", "{}", "not json", "42",
            '{"daily": []}', '{"daily": [], "weekly": [], "monthly": []}',
            '{"unrelated": [1]}',
            json.dumps({"daily": [{"date": "2024-01-15", "totalTokens": 1234,
                                   "totalCost": 5.5}]}),
            json.dumps({"daily": [{"date": "2024-01-14", "totalTokens": 10}]}),
            json.dumps({"weekly": [{"period": "2024-01-08", "totalTokens": 99}]}),
            json.dumps({"monthly": [{"period": "2024-01", "totalTokens": 77}]}),
            json.dumps({
                "daily": [{"date": "2024-01-15", "totalTokens": 1_500_000,
                           "totalCost": 12.34,
                           "agents": [{"agent": "claude", "totalTokens": 1_000_000,
                                       "totalCost": 10.0},
                                      {"agent": "codex", "totalTokens": 500_000,
                                       "totalCost": 2.34}],
                           "metadata": {"agents": ["claude", "codex"]}}],
                "weekly": [{"period": "2024-01-08", "totalTokens": 3_000_000}],
                "monthly": [{"period": "2024-01", "totalTokens": 9_000_000}],
            }),
            # Rows present but none current: the synthetic row must take over.
            json.dumps({"daily": [{"date": "2020-01-01", "totalTokens": 5}]}),
        ]
        cases = [(report, self._NOW) for report in reports]
        self._compare(
            "AiUsageStateFromJSON", cases,
            lambda report, now: collectors.ai_usage_state_from_json(report, now_epoch=now),
        )

    def test_ai_usage_default_is_byte_identical(self):
        from eww_bar_backend.common import AI_USAGE_DEFAULT

        answers = self._ask_go([{"fn": "AiUsageDefault", "args": []}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        self.assertEqual(answers[0]["value"], AI_USAGE_DEFAULT)


    # -- AI usage: the impure shell -------------------------------------------
    #
    # Every seam is driven from one flat fixture map, encoded identically on
    # both sides -- see collect.InstallFixture for the key contract.

    @staticmethod
    def _run_key(argv):
        return "\x1f".join(argv)

    def _python_under_fixture(self, fixture, call, journal=None):
        """Run `call` with the Python's seams bound to the same fixture.

        journal, when given, collects the impure operations in the same
        encoding collect.InstallFixture uses, so a caller can compare the
        SEQUENCE of side effects and not just the return value.
        """
        import contextlib
        import os
        import subprocess as real_subprocess

        def note(entry):
            if journal is not None:
                journal.append(entry)

        def fake_run_text(argv, timeout=None, **_kwargs):
            note("run\x1f" + self._run_key(argv))
            return fixture.get(self._run_key(argv), "")

        class FakeCompleted:
            def __init__(self, code):
                self.returncode = code

        def fake_subprocess_run(argv, **_kwargs):
            key = self._run_key(argv)
            note("status\x1f" + key)
            return FakeCompleted(int(fixture.get("status\x1f" + key, "0")))

        class FakeSubprocess:
            DEVNULL = real_subprocess.DEVNULL
            run = staticmethod(fake_subprocess_run)

        class FakeWallPath:
            """A pathlib stand-in for the wallpaper picker, backed by fixture.

            The picker needs metadata and symlink resolution, not contents, so
            it cannot go through the FakePath used for /proc reads.
            """

            def __init__(self, path, is_file=True, mtime=0, size=0):
                self._p = str(path)
                self._is_file, self._mtime, self._size = is_file, mtime, size

            def __str__(self):
                return self._p

            __repr__ = __str__

            def __lt__(self, other):
                return self._p < str(other)

            def __eq__(self, other):
                return self._p == str(other)

            def __hash__(self):
                return hash(self._p)

            def __truediv__(self, other):
                return FakeWallPath(self._p.rstrip("/") + "/" + str(other))

            @property
            def suffix(self):
                name = self._p.rsplit("/", 1)[-1]
                return "." + name.rsplit(".", 1)[1] if "." in name[1:] else ""

            @property
            def stem(self):
                name = self._p.rsplit("/", 1)[-1]
                return name[: -len(self.suffix)] if self.suffix else name

            @property
            def parent(self):
                return FakeWallPath(self._p.rsplit("/", 1)[0] or "/")

            def is_file(self):
                return self._is_file

            def exists(self):
                return fixture.get("exists\x1f" + self._p) == "1"

            def resolve(self):
                return FakeWallPath(fixture.get("resolve\x1f" + self._p, self._p))

            def stat(self):
                class Stat:
                    st_mtime_ns = self._mtime
                    st_size = self._size
                return Stat()

            def iterdir(self):
                key = "dir\x1f" + self._p
                if key not in fixture:
                    raise FileNotFoundError(self._p)
                out = []
                for line in fixture[key].splitlines():
                    parts = line.split("|")
                    if len(parts) != 4:
                        continue
                    out.append(FakeWallPath(self._p + "/" + parts[0],
                                            parts[1] == "1", int(parts[2]), int(parts[3])))
                return out

            def mkdir(self, *_a, **_k):
                """No-op; the Go seam creates directories internally."""

        class FakeModePath:
            """Stands in for paths.display_mode_path()."""

            def __init__(self, path):
                self._path = path

            def __str__(self):
                return self._path

            @property
            def parent(self):
                class Parent:
                    def mkdir(self, *_a, **_k):
                        """No-op: the Go seam creates directories internally."""
                return Parent()

            def read_text(self, *_a, **_k):
                key = "file\x1f" + self._path
                if key not in fixture:
                    raise FileNotFoundError(self._path)
                return fixture[key]

            def write_text(self, text, *_a, **_k):
                note("write\x1f" + self._path + "\x1f" + text)

            def unlink(self, *_a, **_k):
                note("remove\x1f" + self._path)

        def fake_read_text(path):
            key = "file\x1f" + str(path)
            if key not in fixture:
                raise FileNotFoundError(path)
            return fixture[key]

        class FakePath(type(Path("/"))):
            def read_text(self, *_a, **_k):
                return fake_read_text(self)

        def fake_urlopen(request, timeout=None):
            import urllib.error

            if fixture.get("http.error"):
                raise OSError(fixture["http.error"])
            status = int(fixture.get("http.status", "200"))
            if status >= 400:
                # A real (empty) file object, not None: HTTPError falls back to
                # a tempfile when given None and then warns about cleaning it up.
                import io
                raise urllib.error.HTTPError(
                    request.full_url, status, "err", {}, io.BytesIO(b""))

            @contextlib.contextmanager
            def response():
                class R:
                    def read(self_inner):
                        return fixture.get("http.body", "").encode()
                yield R()

            return response()

        def escaped(argv, *_a, **_k):
            raise AssertionError(
                "a fixture test reached the REAL subprocess: "
                f"{argv!r}\n"
                "Some module under test holds an unpatched `subprocess`. Patch it "
                "in _python_under_fixture before running anything else -- without "
                "this guard the command executes on the developer's machine, which "
                "is how test_control_handle came to toggle wifi, bluetooth, volume "
                "and the media player before anyone noticed.")

        env = {k[len("env."):]: v for k, v in fixture.items() if k.startswith("env.")}
        mode_path = fixture.get("display.mode_path", "/run/eww-display-mode")
        patches = [
            # FIRST, and load-bearing: anything this harness forgot to patch now
            # fails loudly instead of running against the real desktop session.
            unittest.mock.patch("subprocess.run", escaped),
            unittest.mock.patch("subprocess.Popen", escaped),
            unittest.mock.patch.object(collectors, "run_text", fake_run_text),
            unittest.mock.patch.object(collectors, "Path", FakePath),
            unittest.mock.patch.object(display, "run_text", fake_run_text),
            unittest.mock.patch.object(display, "subprocess", FakeSubprocess),
            unittest.mock.patch.object(inhibitors, "subprocess", FakeSubprocess),
            unittest.mock.patch.object(control, "subprocess", FakeSubprocess),
            unittest.mock.patch.object(notifications, "run_text", fake_run_text),
            unittest.mock.patch.object(notifications, "subprocess", FakeSubprocess),
            unittest.mock.patch.object(wallpaper, "run_text", fake_run_text),
            unittest.mock.patch.object(wallpaper, "subprocess", FakeSubprocess),
            unittest.mock.patch.object(wallpaper, "Path", FakeWallPath),
            unittest.mock.patch.object(
                wallpaper, "WALLPAPER_DIR",
                FakeWallPath(fixture.get("wallpaper.dir", "/w"))),
            unittest.mock.patch.object(
                wallpaper, "THUMB_DIR",
                FakeWallPath(fixture.get("wallpaper.thumbdir", "/t"))),
            unittest.mock.patch.object(notifications, "boottime_seconds",
                                       lambda: float(fixture.get("boottime", "0"))),
            unittest.mock.patch.object(display, "display_mode_path",
                                       lambda: FakeModePath(mode_path)),
            unittest.mock.patch.dict(os.environ, env, clear=False),
        ]
        if "now" in fixture:
            patches.append(
                unittest.mock.patch.object(collectors.time, "time",
                                           lambda: float(fixture["now"])))
        patches.append(
            unittest.mock.patch("urllib.request.urlopen", fake_urlopen))

        # Every process-global cache, matching what collect.InstallFixture
        # clears on the Go side. A warm sink cache here made Python skip the two
        # pactl calls Go still made, which showed up as a journal difference
        # rather than as the harness asymmetry it was.
        collectors.reset_quota_cache()
        collectors.reset_volume_sinks_cache()
        collectors._LAST_AI_USAGE = None
        with contextlib.ExitStack() as stack:
            # HTTPError subclasses addinfourl, whose __del__ warns when it is
            # collected unclosed. The original never closes it (it only reads
            # .code), so the warning is the test's noise, not a defect.
            import warnings
            stack.enter_context(warnings.catch_warnings())
            warnings.filterwarnings("ignore", category=ResourceWarning)
            for patch in patches:
                stack.enter_context(patch)
            return call()

    def _compare_fixture(self, fn, cases, python_call, with_journal=False):
        """cases is a list of (fixture, *extra_args)."""
        answers = self._ask_go(
            [{"fn": fn, "args": [fixture, *extra]} for fixture, *extra in cases])
        mismatches = []
        for (fixture, *extra), answer in zip(cases, answers):
            self.assertTrue(answer["ok"], f"{fn} errored in Go: {answer.get('error')}")
            journal = [] if with_journal else None
            expected = self._python_under_fixture(
                fixture, lambda f=fixture, e=extra: python_call(*e), journal=journal)
            if with_journal:
                expected = {**expected, "journal": journal}
            if answer["value"] != expected:
                mismatches.append((fixture, expected, answer["value"]))
        if mismatches:
            detail = "\n".join(
                f"  fixture={f}\n    python={p!r}\n    go    ={g!r}"
                for f, p, g in mismatches[:5])
            self.fail(f"{len(mismatches)}/{len(cases)} disagreed:\n{detail}")

    def test_claude_credentials_paths(self):
        cases = [
            ({"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": ""},),
            ({"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "/etc/claude"},),
            ({"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "  /etc/claude  "},),
            ({"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "   "},),
            ({"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "~/cfg"},),
        ]
        self._compare_fixture(
            "ClaudeCredentialsPaths", cases,
            lambda: [str(p) for p in collectors.claude_credentials_paths()])

    _CREDS = "/home/tester/.claude/.credentials.json"
    _CREDS_ALT = "/home/tester/.config/claude/.credentials.json"

    def _oauth_fixtures(self):
        """Credential-file shapes. Synthetic throughout: no real file is read."""
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": ""}
        return [
            dict(base),
            dict(base, **{"file\x1f" + self._CREDS: "not json"}),
            dict(base, **{"file\x1f" + self._CREDS: "[]"}),
            dict(base, **{"file\x1f" + self._CREDS: "{}"}),
            dict(base, **{"file\x1f" + self._CREDS: '{"claudeAiOauth": {}}'}),
            dict(base, **{"file\x1f" + self._CREDS:
                          '{"claudeAiOauth": {"accessToken": ""}}'}),
            dict(base, **{"file\x1f" + self._CREDS:
                          '{"claudeAiOauth": "not a dict"}'}),
            dict(base, **{"file\x1f" + self._CREDS:
                          '{"claudeAiOauth": {"accessToken": "tok"}}'}),
            dict(base, **{"file\x1f" + self._CREDS:
                          '{"claudeAiOauth": {"accessToken": "tok",'
                          ' "subscriptionType": "max_5x", "expiresAt": 4102444800}}'}),
            # First path present but unusable, second good.
            dict(base, **{"file\x1f" + self._CREDS: "{}",
                          "file\x1f" + self._CREDS_ALT:
                          '{"claudeAiOauth": {"accessToken": "alt"}}'}),
            # First path ABSENT, second good. A different branch from the one
            # above: this is the unreadable-file path, and only it exercises
            # whether the loop keeps going or stops at the first miss.
            dict(base, **{"file\x1f" + self._CREDS_ALT:
                          '{"claudeAiOauth": {"accessToken": "alt"}}'}),
        ]

    def test_claude_load_oauth(self):
        self._compare_fixture(
            "ClaudeLoadOAuth", [(f,) for f in self._oauth_fixtures()],
            collectors.claude_load_oauth)

    def test_claude_quota_state(self):
        good = ('{"claudeAiOauth": {"accessToken": "tok",'
                ' "subscriptionType": "max_5x", "expiresAt": %s}}')
        body = json.dumps({"five_hour": {"utilization": 50},
                           "seven_day": {"utilization": 85}})
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW)}

        def creds(expires):
            return {"file\x1f" + self._CREDS: good % expires}

        cases = [
            # No credentials at all.
            (dict(base),),
            # Expired, in seconds and in milliseconds.
            (dict(base, **creds(int(self._NOW) - 10)),),
            (dict(base, **creds(int(self._NOW * 1000) - 10000)),),
            # expiresAt of 0 means "unknown", not "expired".
            (dict(base, **creds(0)),),
            (dict(base, **creds(int(self._NOW) + 3600), **{"http.body": body}),),
            (dict(base, **creds(int(self._NOW * 1000) + 3600000),
                  **{"http.body": body}),),
            # Transport failure, then each HTTP status the handler splits on.
            (dict(base, **creds(0), **{"http.error": "boom"}),),
            (dict(base, **creds(0), **{"http.status": "401"}),),
            (dict(base, **creds(0), **{"http.status": "403"}),),
            (dict(base, **creds(0), **{"http.status": "500"}),),
            (dict(base, **creds(0), **{"http.status": "404"}),),
            (dict(base, **creds(0), **{"http.status": "200", "http.body": "{}"}),),
            (dict(base, **creds(0), **{"http.status": "200", "http.body": body}),),
        ]
        self._compare_fixture("ClaudeQuotaState", cases, collectors.claude_quota_state)

    _PROBE_KEY = "openusage-cli\x1fprobe\x1fcodex\x1fantigravity"

    def _probe_report(self, *providers):
        return json.dumps([
            {"providerId": key,
             "plan": "Pro",
             "lines": [{"type": "progress", "label": "Session", "used": used,
                        "limit": 100, "format": {"kind": "percent"}}]}
            for key, used in providers
        ])

    def test_openusage_quota_states(self):
        base = {"now": str(self._NOW)}
        cases = [
            (dict(base), self._NOW),
            (dict(base, **{self._PROBE_KEY: "not json"}), self._NOW),
            (dict(base, **{self._PROBE_KEY: "{}"}), self._NOW),
            (dict(base, **{self._PROBE_KEY: "[]"}), self._NOW),
            (dict(base, **{self._PROBE_KEY: '["not a dict"]'}), self._NOW),
            (dict(base, **{self._PROBE_KEY: self._probe_report(("codex", 50))}),
             self._NOW),
            (dict(base, **{self._PROBE_KEY: self._probe_report(
                ("codex", 50), ("antigravity", 95))}), self._NOW),
            # An unrelated provider id: neither card should pick it up.
            (dict(base, **{self._PROBE_KEY: self._probe_report(("gemini", 50))}),
             self._NOW),
            (dict(base, **{self._PROBE_KEY: json.dumps(
                [{"providerId": "codex",
                  "lines": [{"type": "text", "label": "Error", "value": "nope"}]}])}),
             self._NOW),
        ]
        self._compare_fixture(
            "OpenusageQuotaStates", cases,
            lambda now: collectors.openusage_quota_states(now_epoch=now))

    def test_quota_states_caches_across_calls(self):
        """The cache is only observable across two calls, so make two."""
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50))}
        cases = [(dict(base),), (dict(base, **{"http.status": "401"}),)]

        def python_call():
            first = collectors.quota_states(True)
            second = collectors.quota_states(False)
            return [first, second]

        self._compare_fixture("QuotaStatesSequence", cases, python_call)

    _CCUSAGE_KEY = ("ccusage\x1fdaily\x1f--json\x1f--offline\x1f--sections"
                    "\x1fdaily,weekly,monthly\x1f--by-agent\x1f--since\x1f")

    def _ccusage_key(self, now):
        since = time.strftime("%Y%m%d", time.localtime(now - 45 * 86400))
        return self._CCUSAGE_KEY + since

    def _ccusage_report(self):
        return json.dumps({
            "daily": [{"date": time.strftime("%F", time.localtime(self._NOW)),
                       "totalTokens": 1234, "totalCost": 5.5}],
            "weekly": [], "monthly": [],
        })

    def test_ai_usage_state(self):
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50))}
        good = {self._ccusage_key(self._NOW): self._ccusage_report()}
        cases = [
            (dict(base), True),
            (dict(base), False),
            (dict(base, **good), True),
            (dict(base, **good), False),
            (dict(base, **{self._ccusage_key(self._NOW): "not json"}), True),
            (dict(base, **{self._ccusage_key(self._NOW): '{"daily": []}'}), True),
        ]
        self._compare_fixture(
            "AiUsageState", cases,
            lambda refresh: collectors.ai_usage_state(refresh_quotas=refresh))

    def test_ai_usage_state_serves_stale_after_a_good_report(self):
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50)),
                self._ccusage_key(self._NOW): self._ccusage_report()}

        def python_call():
            good = collectors.ai_usage_state(refresh_quotas=True)
            with unittest.mock.patch.object(collectors, "run_text",
                                            lambda *a, **k: ""):
                stale = collectors.ai_usage_state(refresh_quotas=True)
            return [good, stale]

        self._compare_fixture("AiUsageStateStaleSequence", [(dict(base),)], python_call)

    def test_ai_usage_state_does_not_go_stale_on_two_good_reports(self):
        """The stale branch's source guard, which needs two GOOD calls to see.

        With one call the remembered state is still empty, and with a
        good-then-broken pair both a guarded and an unguarded stale branch
        behave identically -- so neither shape can tell whether the guard is
        there. Two good calls can: without the guard the second would come back
        marked stale.
        """
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50)),
                self._ccusage_key(self._NOW): self._ccusage_report()}

        def python_call():
            return [collectors.ai_usage_state(refresh_quotas=True),
                    collectors.ai_usage_state(refresh_quotas=True)]

        self._compare_fixture("AiUsageStateGoodTwice", [(dict(base),)], python_call)

    def test_refresh_ai_usage(self):
        from eww_bar_backend.common import AI_USAGE_DEFAULT

        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50)),
                self._ccusage_key(self._NOW): self._ccusage_report()}

        currents = [
            copy_default := json.loads(json.dumps(AI_USAGE_DEFAULT)),
            {**json.loads(json.dumps(AI_USAGE_DEFAULT)),
             "meta": {**AI_USAGE_DEFAULT["meta"], "refreshing": "true"}},
        ]
        cases = [(dict(base), current, refresh)
                 for current in currents for refresh in (True, False)]

        class FakeState:
            def __init__(self, current):
                self.current = current
                self.published = []

            def get(self, key, default=None):
                return self.current if key == "ai_usage" else default

            def update(self, **items):
                self.published.append(items["ai_usage"])
                self.current = items["ai_usage"]

        def python_call(current, refresh):
            state = FakeState(json.loads(json.dumps(current)))
            result = collectors.refresh_ai_usage(state, refresh_quotas=refresh)
            return {"published": state.published, "result": result}

        self._compare_fixture("RefreshAiUsage", cases, python_call)

    def test_ai_refresh_cycle_probe_cadence(self):
        """Which ticks pay for the openusage probe, over a full period."""
        base = {"env.HOME": "/home/tester", "env.CLAUDE_CONFIG_DIR": "",
                "now": str(self._NOW),
                self._PROBE_KEY: self._probe_report(("codex", 50)),
                self._ccusage_key(self._NOW): self._ccusage_report()}
        cases = [(dict(base), every, 14) for every in (1, 2, 6)]

        class FakeState:
            def __init__(self):
                self.current = None

            def get(self, key, default=None):
                return self.current if key == "ai_usage" else default

            def update(self, **items):
                self.current = items["ai_usage"]

        def python_call(quota_every, ticks):
            probed = []
            real = collectors.run_text

            def counting(argv, *a, **k):
                if argv and argv[0] == "openusage-cli":
                    probed[-1] = True
                return real(argv, *a, **k)

            refresh = collectors.ai_refresh_cycle(quota_every=quota_every)
            state = FakeState()
            with unittest.mock.patch.object(collectors, "run_text", counting):
                for _ in range(ticks):
                    probed.append(False)
                    refresh(state)
            return probed

        self._compare_fixture("AiRefreshCycleProbeTicks", cases, python_call)


    # -- display and inhibitors -----------------------------------------------

    _MODE_PATH = "/run/eww-display-mode"
    _IDLE_SVC = "systemctl\x1f--user\x1fis-active\x1f--quiet\x1feww-hypridle-inhibit.service"
    _LID_SVC = "systemctl\x1f--user\x1fis-active\x1f--quiet\x1feww-lid-inhibit.service"

    def _display_base(self, **extra):
        # Go builds the path from XDG_RUNTIME_DIR at call time, so the fixture
        # sets it; the Python side has display_mode_path patched to match.
        base = {"display.mode_path": self._MODE_PATH, "env.XDG_RUNTIME_DIR": "/run"}
        base.update(extra)
        return base

    def test_inhibitor_states(self):
        cases = [
            (self._display_base(),),
            (self._display_base(**{"status\x1f" + self._IDLE_SVC: "0"}),),
            (self._display_base(**{"status\x1f" + self._IDLE_SVC: "1"}),),
            (self._display_base(**{"status\x1f" + self._IDLE_SVC: "3"}),),
            (self._display_base(**{"status\x1f" + self._LID_SVC: "1"}),),
        ]
        self._compare_fixture(
            "IdleInhibitedState", cases,
            lambda: {"result": inhibitors.idle_inhibited_state()}, with_journal=True)
        self._compare_fixture(
            "LidInhibitedState", cases,
            lambda: {"result": inhibitors.lid_inhibited_state()}, with_journal=True)

    def _setter_cases(self):
        start = "systemctl\x1f--user\x1fstart\x1feww-hypridle-inhibit.service"
        stop = "systemctl\x1f--user\x1fstop\x1feww-hypridle-inhibit.service"
        return [
            # The read-back disagreeing with the request is the interesting
            # case: systemctl can exit zero for a unit that did not come up.
            (self._display_base(), True),
            (self._display_base(), False),
            (self._display_base(**{"status\x1f" + self._IDLE_SVC: "1"}), True),
            (self._display_base(**{"status\x1f" + start: "1"}), True),
            (self._display_base(**{"status\x1f" + stop: "1"}), False),
            (self._display_base(**{"status\x1f" + start: "0",
                                   "status\x1f" + self._IDLE_SVC: "1"}), True),
        ]

    def _catching(self, call):
        """Run a Python call that may raise, in the shape diffgen reports."""
        try:
            return {"result": call(), "error": None}
        except Exception as exc:  # noqa: BLE001 -- mirrors the Go error return
            return {"result": None, "error": str(exc)}

    def test_set_idle_inhibited(self):
        self._compare_fixture(
            "SetIdleInhibited", self._setter_cases(),
            lambda enabled: self._catching(
                lambda: inhibitors.set_idle_inhibited(enabled)),
            with_journal=True)

    def test_toggle_idle_inhibited(self):
        cases = [
            (self._display_base(),),
            (self._display_base(**{"status\x1f" + self._IDLE_SVC: "1"}),),
        ]
        self._compare_fixture(
            "ToggleIdleInhibited", cases,
            lambda: self._catching(inhibitors.toggle_idle_inhibited),
            with_journal=True)

    def test_read_display_mode(self):
        cases = [
            (self._display_base(),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "external"}),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "headless\n"}),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "  normal  "}),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "bogus"}),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: ""}),),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "EXTERNAL"}),),
        ]
        self._compare_fixture("ReadDisplayMode", cases, display.read_display_mode)

    def test_write_display_mode(self):
        """"normal" is the file's ABSENCE, so it removes rather than writes."""
        cases = [(self._display_base(), mode)
                 for mode in ("normal", "external", "headless", "bogus")]
        self._compare_fixture(
            "WriteDisplayMode", cases,
            lambda mode: {"result": display.write_display_mode(mode)},
            with_journal=True)

    def test_display_state(self):
        cases = [(self._display_base(**{"file\x1f" + self._MODE_PATH: mode}), status)
                 for mode in ("external", "headless", "bogus")
                 for status in ("", "Custom status")]
        cases += [(self._display_base(), ""), (self._display_base(), "Custom")]
        self._compare_fixture("DisplayState", cases, display.display_state)

    def test_monitor_state(self):
        key = "hyprctl\x1fmonitors\x1f-j"
        cases = [
            (self._display_base(),),
            (self._display_base(**{key: "not json"}),),
            (self._display_base(**{key: "[]"}),),
            (self._display_base(**{key: '[{"name": "eDP-1"}]'}),),
            (self._display_base(**{key: '[{"name": "eDP-1"}, {"name": "DP-2"}]'}),),
        ]
        self._compare_fixture("MonitorState", cases, display.monitor_state)

    def test_monitor_state_hardens_what_the_original_would_crash_on(self):
        """Pins the divergence MonitorState documents.

        monitor_state passes hyprctl's output straight through, so a report
        that is not a list of objects reaches split_monitors, which calls .get
        on each element. A JSON object yields its KEYS -- strings -- and a
        stray non-object element is itself a string, and both raise
        AttributeError inside the display switch. Go normalises to the objects
        instead. Asserted here rather than reproduced, so the note in
        display.go cannot quietly stop being true.
        """
        for payload in ('{"a": 1}', '[{"name": "eDP-1"}, "junk"]'):
            monitors = collectors.parse_json(payload, [])
            with self.assertRaises(AttributeError):
                display.split_monitors(monitors)
        # An EMPTY object is the one non-list that survives, because iterating
        # it yields nothing -- which is why the guard cannot just be "is a list".
        self.assertEqual(display.split_monitors({}), ([], []))

    def test_set_display_mode(self):
        """The side-effect SEQUENCE, which is this function's whole content."""
        monitors = "hyprctl\x1fmonitors\x1f-j"
        both = json.dumps([{"name": "eDP-1"}, {"name": "DP-2"}])
        internal_only = json.dumps([{"name": "eDP-1"}])
        two_internal = json.dumps([{"name": "eDP-1"}, {"name": "LVDS-1"},
                                   {"name": "DP-2"}])
        disabled_ext = json.dumps([{"name": "eDP-1"},
                                   {"name": "DP-2", "disabled": True}])
        unnamed = json.dumps([{"name": ""}, {"name": "DP-2"}])

        cases = []
        for action in ("status", "restore", "normal", "external", "headless",
                       "toggle", "bogus", ""):
            cases.append((self._display_base(**{monitors: both}), action))
        cases += [
            # toggle depends on the mode already recorded.
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "headless",
                                   monitors: both}), "toggle"),
            (self._display_base(**{"file\x1f" + self._MODE_PATH: "external",
                                   monitors: both}), "toggle"),
            # external with nothing to switch to, and with a disabled external.
            (self._display_base(**{monitors: internal_only}), "external"),
            (self._display_base(**{monitors: disabled_ext}), "external"),
            (self._display_base(**{monitors: "not json"}), "external"),
            # More than one internal panel: each gets its own keyword call.
            (self._display_base(**{monitors: two_internal}), "external"),
            # An unnamed monitor is skipped rather than disabled as "".
            (self._display_base(**{monitors: unnamed}), "external"),
            # A failing hyprctl must abort before the mode file is written.
            (self._display_base(**{monitors: both,
                                   "status\x1fhyprctl\x1freload": "1"}), "normal"),
            (self._display_base(**{monitors: both,
                                   "status\x1fhyprctl\x1fdispatch\x1fdpms\x1foff": "1"}),
             "headless"),
            (self._display_base(**{monitors: both,
                                   "status\x1fhyprctl\x1fkeyword\x1fmonitor\x1feDP-1,disable": "1"}),
             "external"),
            # A failing systemctl must abort too.
            (self._display_base(**{monitors: both,
                                   "status\x1fsystemctl\x1f--user\x1fstart\x1feww-lid-inhibit.service": "1"}),
             "headless"),
        ]
        self._compare_fixture(
            "SetDisplayMode", cases,
            lambda action: self._catching(lambda: display.set_display_mode(action)),
            with_journal=True)


    # -- notification actions and the wallpaper picker ------------------------

    _HIST = "dunstctl\x1fhistory"
    _PAUSED = "dunstctl\x1fis-paused"

    def _history(self, *items):
        """dunstctl history's nested {"type":..,"data":..} envelope."""
        def field(value):
            return {"type": "s", "data": value}
        return json.dumps({"type": "aa{sv}", "data": [[
            {"id": field(i), "appname": field(app), "summary": field(sm),
             "body": field(""), "urgency": field("NORMAL"),
             "timestamp": field(ts)}
            for i, app, sm, ts in items
        ]]})

    def _notif_base(self, **extra):
        base = {"boottime": "1000", self._PAUSED: "false"}
        base.update(extra)
        return base

    def _notif_fixtures(self):
        two = self._history((1, "Firefox", "Hello", 900_000_000),
                            (2, "Spotify", "Track", 950_000_000))
        return [
            self._notif_base(),
            self._notif_base(**{self._HIST: "not json"}),
            self._notif_base(**{self._HIST: self._history()}),
            self._notif_base(**{self._HIST: two}),
            self._notif_base(**{self._HIST: two, self._PAUSED: "true"}),
            # An app name past the 20-char group cap, so the group name is
            # truncated and clear-group has to truncate to match.
            self._notif_base(**{self._HIST: self._history(
                (3, "an-extremely-long-application-name", "S", 900_000_000))}),
        ]

    def test_collect_notifications(self):
        self._compare_fixture(
            "CollectNotifications", [(f,) for f in self._notif_fixtures()],
            notifications.notifications_state)

    def _notif_python(self, action, arg=""):
        actions = {
            "toggle-group": lambda: notifications.toggle_group(arg),
            "dismiss": lambda: notifications.dismiss_notification(arg),
            "clear-group": lambda: notifications.clear_group(arg),
            "clear-all": notifications.clear_all_notifications,
            "dnd-toggle": notifications.toggle_dnd,
            "mark-seen": notifications.mark_seen,
        }
        notifications._COLLAPSED.clear()
        notifications._LAST_SEEN_US = 0
        return self._catching(actions[action])

    def test_notif_actions(self):
        two = self._history((1, "Firefox", "Hello", 900_000_000),
                            (2, "Spotify", "Track", 950_000_000))
        long_app = self._history((3, "an-extremely-long-application-name", "S", 900_000_000))
        cases = []
        for fixture in self._notif_fixtures():
            for action, arg in (("clear-all", ""), ("dnd-toggle", ""),
                                ("mark-seen", ""), ("toggle-group", "Firefox"),
                                ("toggle-group", ""), ("dismiss", "7"),
                                ("dismiss", ""), ("dismiss", "notanumber"),
                                ("dismiss", " 8 "), ("clear-group", "Firefox"),
                                ("clear-group", "")):
                cases.append((fixture, action, arg))
        # clear-group against a truncated group name: the id must still be found.
        cases.append((self._notif_base(**{self._HIST: long_app}),
                      "clear-group", "an-extremely-long..."))
        # ...and the untruncated name, which must NOT match.
        cases.append((self._notif_base(**{self._HIST: long_app}),
                      "clear-group", "an-extremely-long-application-name"))
        cases.append((self._notif_base(**{self._HIST: two}), "clear-group", "Firefox"))
        self._compare_fixture(
            "NotifAction", cases,
            lambda action, arg: self._notif_python(action, arg), with_journal=True)

    def test_notif_toggle_group_is_a_toggle(self):
        """Collapse state only shows across two calls."""
        two = self._history((1, "Firefox", "Hello", 900_000_000))
        cases = [(self._notif_base(**{self._HIST: two}), "Firefox")]

        def python_call(app):
            notifications._COLLAPSED.clear()
            notifications._LAST_SEEN_US = 0
            first = self._catching(lambda: notifications.toggle_group(app))
            second = self._catching(lambda: notifications.toggle_group(app))
            return [first, second]

        answers = self._ask_go([{"fn": "NotifToggleGroupTwice", "args": [f, a]}
                                for f, a in cases])
        for (fixture, app), answer in zip(cases, answers):
            self.assertTrue(answer["ok"], answer.get("error"))
            journal = []
            expected = self._python_under_fixture(
                fixture, lambda a=app: python_call(a), journal=journal)
            # Both halves carry the running journal, so compare the results and
            # assert the collapse actually flipped.
            self.assertEqual([e["result"] for e in answer["value"]],
                             [e["result"] for e in expected])
            self.assertNotEqual(answer["value"][0]["result"],
                                answer["value"][1]["result"],
                                "toggling twice should not be a no-op")

    _WDIR = "/home/tester/Pictures/wallpapers"
    _TDIR = "/home/tester/.cache/eww-bar/wallpaper-thumbs"

    def _wall_base(self, listing=(), **extra):
        base = {"env.HOME": "/home/tester",
                "wallpaper.dir": self._WDIR, "wallpaper.thumbdir": self._TDIR,
                "dir\x1f" + self._WDIR: "\n".join(listing)}
        base.update(extra)
        return base

    def test_scan_wallpaper_files(self):
        cases = [
            (self._wall_base(),),
            ({"env.HOME": "/home/tester", "wallpaper.dir": self._WDIR,
              "wallpaper.thumbdir": self._TDIR},),  # unreadable directory
            (self._wall_base(["a.png|1|100|10", "b.jpg|1|200|20"]),),
            # Out of order on disk: the scan has to sort.
            (self._wall_base(["z.png|1|100|10", "a.png|1|200|20"]),),
            # Extensions: case-insensitive, and .jxl is excluded on purpose.
            (self._wall_base(["a.PNG|1|1|1", "b.JPEG|1|1|1", "c.jxl|1|1|1",
                              "d.txt|1|1|1", "e.tiff|1|1|1"]),),
            # A directory entry, not a file -- and one NAMED like a wallpaper,
            # since a bare "sub" is already excluded by the extension filter and
            # so proves nothing about the is-file check.
            (self._wall_base(["sub|0|1|1", "a.png|1|1|1"]),),
            (self._wall_base(["album.png|0|1|1", "a.png|1|1|1"]),),
            (self._wall_base(["noext|1|1|1"]),),
        ]
        self._compare_fixture(
            "ScanWallpaperFiles", [(f, self._WDIR) for f, in cases],
            lambda d: [str(p) for p in wallpaper.scan_wallpaper_files(d)])

    def test_thumb_cache_path(self):
        """The digest input, byte for byte.

        A mismatch here does not fail loudly -- it silently invalidates every
        cached thumbnail and re-runs magick once per wallpaper at up to 15 s
        each, which reads as "the picker is slow now".
        """
        import hashlib

        cases = [
            (self._wall_base(), "/w/a.png", 12345, 678),
            (self._wall_base(), "/w/a.png", 0, 0),
            (self._wall_base(), "/w/spaced name.png", 1, 2),
            (self._wall_base(), "/w/uni\u00e9.png", 999999999999, 4),
            (self._wall_base(), "", 1, 1),
        ]

        def python_call(path, mtime, size):
            digest = hashlib.sha1(f"{path}:{mtime}:{size}".encode()).hexdigest()
            return str(wallpaper.THUMB_DIR / f"{digest}.png")


        self._compare_fixture("ThumbCachePath", cases, python_call)

    def test_wallpaper_state(self):
        listing = ["a.png|1|100|10", "b.gif|1|200|20"]
        query = "awww\x1fquery"
        line = "eDP-1: 1920x1200, scale: 2, currently displaying: image: "
        cases = [
            (self._wall_base(listing),),
            (self._wall_base(listing, **{query: line + self._WDIR + "/a.png"}),),
            (self._wall_base(listing, **{query: "no image here"}),),
            (self._wall_base(),),
            # A symlinked current wallpaper, which is why realPath exists.
            (self._wall_base(listing, **{
                query: line + "/link.png",
                "resolve\x1f/link.png": self._WDIR + "/b.gif"}),),
        ]

        self._compare_fixture("CollectWallpaper", cases, wallpaper.wallpaper_state)

    def test_set_wallpaper(self):
        listing = ["a.png|1|100|10"]
        cases = [(self._wall_base(listing), path)
                 for path in ("", self._WDIR + "/a.png", "/elsewhere/x.png")]
        self._compare_fixture(
            "SetWallpaper", cases,
            lambda path: self._catching(lambda: wallpaper.set_wallpaper(path)),
            with_journal=True)

    def test_mark_seen_never_moves_the_badge_backwards(self):
        """Needs THREE calls, not two.

        With an empty or older second history the returned state is the same
        either way, because `new` counts items against lastSeen and there are
        none to count. And the third step has to be a plain READ rather than a
        third mark, because mark_seen updates the high-water mark before
        reading state back and would repair the rewind before anyone saw it.
        """
        recent = self._history((1, "Firefox", "Hello", 900_000_000))
        older = self._history((2, "Spotify", "Old", 100_000_000))
        histories = [recent, older, recent]
        fixture = self._notif_base(**{self._HIST: recent, "boottime": "0.5"})

        def python_call():
            notifications._COLLAPSED.clear()
            notifications._LAST_SEEN_US = 0
            out = []
            for i, body in enumerate(histories):
                with unittest.mock.patch.object(
                        notifications, "run_text",
                        lambda argv, b=body, **k: b
                        if argv[:2] == ["dunstctl", "history"]
                        else fixture.get("\x1f".join(argv), "")):
                    # The last step is a plain READ: mark_seen updates the
                    # high-water mark before reading state back, so a third
                    # mark would repair the rewind before it could be seen.
                    if i == len(histories) - 1:
                        out.append({"result": notifications.notifications_state(),
                                    "error": None})
                    else:
                        out.append(self._catching(notifications.mark_seen))
            return out

        answers = self._ask_go(
            [{"fn": "NotifMarkSeenRewind", "args": [fixture, histories]}])
        self.assertTrue(answers[0]["ok"], answers[0].get("error"))
        expected = self._python_under_fixture(fixture, python_call, journal=[])
        self.assertEqual([e["result"] for e in answers[0]["value"]],
                         [e["result"] for e in expected])
        # The third read must show nothing new: the 900 ms item was already
        # marked seen by the first call and the second must not have undone it.
        self.assertEqual(answers[0]["value"][2]["result"]["new"], 0)


    # -- the control command handler ------------------------------------------

    def _control_base(self, **extra):
        """Everything the eight command branches touch, in one fixture."""
        base = self._notif_base()
        base.update(self._wall_base(["a.png|1|100|10"]))
        base.update(self._display_base())
        base.update({
            "wpctl\x1fget-volume\x1f@DEFAULT_AUDIO_SINK@": "Volume: 0.50",
            "pactl\x1fget-default-sink": "sink-a",
            "pactl\x1f-f\x1fjson\x1flist\x1fsinks":
                '[{"name": "sink-a", "description": "Speakers"}]',
            "playerctl\x1fstatus": "Playing",
            "playerctl\x1fmetadata\x1f--format\x1f{{artist}} - {{title}}": "A - B",
            "bluetoothctl\x1fshow": "Alias: host\nPowered: yes",
            "bluetoothctl\x1fdevices\x1fConnected": "",
            "nmcli\x1fradio\x1fwifi": "enabled",
            "nmcli\x1f-t\x1f-f\x1fDEVICE,TYPE,STATE\x1fdev\x1fstatus": "",
            "hyprctl\x1fmonitors\x1f-j": json.dumps(
                [{"name": "eDP-1"}, {"name": "DP-2"}]),
        })
        base.update(extra)
        return base

    def test_control_handle(self):
        """Every branch, its reply shape, its state write and its side effects."""
        payloads = [
            {}, {"command": "nope"}, {"command": "ping"},
            # volume
            {"command": "volume", "action": "up"},
            {"command": "volume", "action": "down"},
            {"command": "volume", "action": "mute"},
            {"command": "volume", "action": "set", "value": "42"},
            {"command": "volume", "action": "set", "value": "42.6"},
            {"command": "volume", "action": "set", "value": "150"},
            {"command": "volume", "action": "set", "value": "-5"},
            {"command": "volume", "action": "set", "value": "abc"},
            {"command": "volume", "action": "set"},
            {"command": "volume", "action": "sink", "sink": "sink-a"},
            {"command": "volume", "action": "sink"},
            {"command": "volume", "action": "bogus"},
            {"command": "volume"},
            # media
            {"command": "media", "action": "play-pause"},
            {"command": "media", "action": "next"},
            {"command": "media", "action": "bogus"},
            # bluetooth
            {"command": "bluetooth", "action": "power-toggle"},
            {"command": "bluetooth", "action": "disconnect", "mac": "AA:BB"},
            {"command": "bluetooth", "action": "disconnect"},
            {"command": "bluetooth", "action": "bogus"},
            # network
            {"command": "network", "action": "wifi-toggle"},
            {"command": "network", "action": "bogus"},
            # idle -- note the reply carries no "action" key
            {"command": "idle"}, {"command": "idle", "action": "on"},
            {"command": "idle", "action": "off"},
            {"command": "idle", "action": "status"},
            {"command": "idle", "action": "bogus"},
            # display
            {"command": "display"}, {"command": "display", "action": "headless"},
            {"command": "display", "action": "bogus"},
            # notif
            {"command": "notif", "action": "clear-all"},
            {"command": "notif", "action": "dnd-toggle"},
            {"command": "notif", "action": "mark-seen"},
            {"command": "notif", "action": "toggle-group", "app": "Firefox"},
            {"command": "notif", "action": "toggle-group"},
            {"command": "notif", "action": "dismiss", "id": "3"},
            {"command": "notif", "action": "dismiss", "id": "x"},
            {"command": "notif", "action": "clear-group", "app": "Firefox"},
            {"command": "notif", "action": "bogus"},
            # wallpaper
            {"command": "wallpaper", "action": "rescan"},
            {"command": "wallpaper", "action": "set", "path": "/w/a.png"},
            {"command": "wallpaper", "action": "set"},
            {"command": "wallpaper", "action": "bogus"},
        ]
        cases = [(self._control_base(), payload) for payload in payloads]

        class FakeState:
            def __init__(self, initial):
                self.state = initial

            def get(self, key, default=None):
                return self.state.get(key, default)

            def update(self, **items):
                self.state.update(items)

            def snapshot(self):
                return json.dumps(self.state, separators=(",", ":"),
                                  ensure_ascii=False)

        def python_call(payload):
            from eww_bar_backend.state import BarState

            store = FakeState(json.loads(BarState().snapshot()))
            try:
                reply = control.handle_control_command(store, payload)
            except Exception as exc:  # noqa: BLE001 -- serve_control_connection does this
                reply = {"ok": False, "error": str(exc)}
            return {"reply": json.dumps(reply, separators=(",", ":")),
                    "snapshot": store.snapshot()}

        self._compare_fixture("ControlHandle", cases, python_call, with_journal=True)


def _contiguous_runs(chars):
    """Split a sorted character list into runs of consecutive code points."""
    runs, current = [], [chars[0]]
    for char in chars[1:]:
        if ord(char) == ord(current[-1]) + 1:
            current.append(char)
        else:
            runs.append(current)
            current = [char]
    runs.append(current)
    return runs


def _float_bits(value):
    """The IEEE bits of a float, with NaN canonicalised.

    Mirrors diffgen's floatBits. Bits rather than the number itself because
    Go's encoding/json refuses Inf and NaN, and because they distinguish -0.0.
    """
    import math
    import struct

    if isinstance(value, float) and math.isnan(value):
        return 0x7FF8000000000000
    return struct.unpack("<Q", struct.pack("<d", float(value)))[0]


def _py_float_answer(text):
    try:
        return [True, _float_bits(float(text))]
    except ValueError:
        return [False, _float_bits(0.0)]


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
