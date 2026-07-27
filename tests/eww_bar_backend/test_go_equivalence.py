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
import unittest.mock
from decimal import ROUND_HALF_UP, Decimal
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
GO_DIR = EWW_DIR / "go"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, display, notifications, wallpaper, watchers  # noqa: E402
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
