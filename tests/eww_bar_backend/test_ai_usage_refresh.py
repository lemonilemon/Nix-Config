"""The quota probe must stay off the background poll.

`openusage-cli probe` is by far the most expensive thing the daemon does:
measured on this host, 43.6 CPU-seconds and 41.6 s of wall clock per call,
against 0.85 for the ccusage report beside it. None of what it produces reaches
the bar face -- eww.yuck:80-83 renders only ai_usage.text, which comes from
ccusage -- and eww.yuck:81 already fires `eww-barctl ai refresh` when the popup
that does show quotas is opened. Running it on the 5-minute poll spent most of
the daemon's CPU budget refreshing something nobody was looking at.
"""

import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors  # noqa: E402


def fake_quota(key, name):
    return {"key": key, "name": name, "status": "live", "class": "", "windows": [], "meta": []}


class QuotaCacheTests(unittest.TestCase):
    def setUp(self):
        collectors.reset_quota_cache()
        self.addCleanup(collectors.reset_quota_cache)
        self.claude = unittest.mock.patch.object(
            collectors, "claude_quota_state", side_effect=lambda: fake_quota("claude", "Claude")
        )
        self.openusage = unittest.mock.patch.object(
            collectors,
            "openusage_quota_states",
            side_effect=lambda: [fake_quota("codex", "Codex")],
        )
        self.claude_mock = self.claude.start()
        self.openusage_mock = self.openusage.start()
        self.addCleanup(self.claude.stop)
        self.addCleanup(self.openusage.stop)

    def test_first_call_probes(self):
        quotas = collectors.quota_states()
        self.assertEqual([q["key"] for q in quotas], ["claude", "codex"])
        self.assertEqual(self.openusage_mock.call_count, 1)

    def test_refresh_false_reuses_the_cache_without_probing(self):
        first = collectors.quota_states()
        self.openusage_mock.reset_mock()
        self.claude_mock.reset_mock()
        second = collectors.quota_states(refresh=False)
        self.assertEqual(second, first)
        self.assertEqual(self.openusage_mock.call_count, 0)
        self.assertEqual(self.claude_mock.call_count, 0)

    def test_cold_cache_probes_even_when_not_asked_to(self):
        # Startup must not leave the popup empty until the first explicit
        # refresh 30 minutes later.
        collectors.quota_states(refresh=False)
        self.assertEqual(self.openusage_mock.call_count, 1)

    def test_cached_quotas_are_copies(self):
        first = collectors.quota_states()
        first[0]["status"] = "mutated"
        second = collectors.quota_states(refresh=False)
        self.assertEqual(second[0]["status"], "live")


class AiUsageStateTests(unittest.TestCase):
    def setUp(self):
        collectors.reset_quota_cache()
        self.addCleanup(collectors.reset_quota_cache)

    def test_refresh_quotas_false_does_not_run_openusage(self):
        commands = []

        def fake_run_text(command, **_kwargs):
            commands.append(command[0])
            return ""

        with unittest.mock.patch.object(collectors, "run_text", side_effect=fake_run_text):
            with unittest.mock.patch.object(
                collectors, "claude_quota_state", return_value=fake_quota("claude", "Claude")
            ):
                collectors.quota_states()  # warm the cache
                commands.clear()
                collectors.ai_usage_state(refresh_quotas=False)

        self.assertIn("ccusage", commands)
        self.assertNotIn("openusage-cli", commands)


class RefreshCycleTests(unittest.TestCase):
    """The background poll keeps the cheap half and skips the expensive one."""

    def _record(self, cycles, quota_every):
        seen = []
        refresh = collectors.ai_refresh_cycle(quota_every=quota_every)
        with unittest.mock.patch.object(
            collectors,
            "refresh_ai_usage",
            side_effect=lambda _state, refresh_quotas=True: seen.append(refresh_quotas),
        ):
            for _ in range(cycles):
                refresh(object())
        return seen

    def test_first_cycle_refreshes_quotas(self):
        self.assertTrue(self._record(1, 6)[0])

    def test_quotas_refresh_only_every_nth_cycle(self):
        seen = self._record(13, 6)
        self.assertEqual(seen, [True] + [False] * 5 + [True] + [False] * 5 + [True])
        self.assertEqual(sum(seen), 3)

    def test_every_cycle_still_refreshes_the_cheap_half(self):
        # 12 cycles must be 12 ccusage reports, whatever the quota cadence.
        self.assertEqual(len(self._record(12, 6)), 12)

    def test_quota_every_one_refreshes_always(self):
        self.assertEqual(self._record(4, 1), [True] * 4)


class ControlRefreshTests(unittest.TestCase):
    def test_the_popups_explicit_refresh_still_probes(self):
        # eww.yuck:81 and :491 both send `ai refresh`; that path is the whole
        # reason the background poll can stop probing, so it must not inherit
        # the cheap default.
        import inspect

        signature = inspect.signature(collectors.refresh_ai_usage)
        self.assertIs(signature.parameters["refresh_quotas"].default, True)


if __name__ == "__main__":
    unittest.main()
