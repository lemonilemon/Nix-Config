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

from eww_bar_backend import collectors  # noqa: E402
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
