import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import watchers  # noqa: E402
from eww_bar_backend.watchers import (  # noqa: E402
    bar_window_command,
    missing_bar_monitors,
    monitor_event,
    open_bar_names,
)


class MonitorEventTests(unittest.TestCase):
    def test_added_parses_name(self):
        self.assertEqual(monitor_event("monitoradded>>HDMI-A-1"), ("added", "HDMI-A-1"))

    def test_removed_parses_name(self):
        self.assertEqual(monitor_event("monitorremoved>>HDMI-A-1"), ("removed", "HDMI-A-1"))

    def test_v2_events_ignored_to_avoid_double_fire(self):
        self.assertIsNone(monitor_event("monitoraddedv2>>1,HDMI-A-1,Dell U2723QE"))
        self.assertIsNone(monitor_event("monitorremovedv2>>1,HDMI-A-1,Dell U2723QE"))

    def test_unrelated_and_empty_events_ignored(self):
        self.assertIsNone(monitor_event("workspace>>3"))
        self.assertIsNone(monitor_event("monitoradded>>"))
        self.assertIsNone(monitor_event(""))


class BarWindowCommandTests(unittest.TestCase):
    def test_added_matches_startup_script_shape(self):
        self.assertEqual(
            bar_window_command("added", "HDMI-A-1"),
            [
                "eww", "open", "bar",
                "--id", "bar-HDMI-A-1",
                "--screen", "HDMI-A-1",
                "--arg", "output=HDMI-A-1",
            ],
        )

    def test_removed_closes_bar_window(self):
        self.assertEqual(
            bar_window_command("removed", "HDMI-A-1"),
            ["eww", "close", "bar-HDMI-A-1"],
        )


class OpenBarNamesTests(unittest.TestCase):
    def test_extracts_monitor_names_from_bar_ids(self):
        text = "bar-eDP-1: bar\nbar-HDMI-A-1: bar\nvolume_popup: volume_popup\n"
        self.assertEqual(open_bar_names(text), {"eDP-1", "HDMI-A-1"})

    def test_empty_output(self):
        self.assertEqual(open_bar_names(""), set())


class MissingBarMonitorsTests(unittest.TestCase):
    MONITORS = '[{"name": "eDP-1"}, {"name": "HDMI-A-1"}]'

    def test_reports_monitor_without_bar(self):
        self.assertEqual(
            missing_bar_monitors(self.MONITORS, "bar-eDP-1: bar\n"),
            ["HDMI-A-1"],
        )

    def test_nothing_missing_when_all_bars_open(self):
        self.assertEqual(
            missing_bar_monitors(self.MONITORS, "bar-eDP-1: bar\nbar-HDMI-A-1: bar\n"),
            [],
        )

    def test_malformed_inputs_are_safe(self):
        self.assertEqual(missing_bar_monitors("not json", ""), [])
        self.assertEqual(missing_bar_monitors('{"name": "x"}', ""), [])
        self.assertEqual(missing_bar_monitors('[{"no_name": 1}, 42]', ""), [])


if __name__ == "__main__":
    unittest.main()


class _Result:
    def __init__(self, returncode=0):
        self.returncode = returncode


class ApplyMonitorEventTests(unittest.TestCase):
    """watchers.py:78-79 and :83-84 -- two separate hotplug workarounds.

    Neither is guessable from the code's shape, and both cost a visibly broken
    bar if a rewrite "simplifies" them away:

      * GDK learns about a hotplugged output a beat after Hyprland announces
        it, so the first `eww open` can fail on a monitor that genuinely
        exists. Dropping the retry means a hotplugged screen gets no bar.
      * Something else may have opened the bar meanwhile (the startup script,
        or eww's own config reload). Re-opening an already-open id makes the
        bar visibly flicker.
    """

    def _patched(self, active_windows="", returncodes=(0,)):
        sleeps = []
        calls = []
        codes = iter(returncodes)

        def fake_run(command, **_kwargs):
            calls.append(command)
            try:
                return _Result(next(codes))
            except StopIteration:
                return _Result(1)

        return sleeps, calls, unittest.mock.patch.multiple(
            watchers,
            time=unittest.mock.Mock(sleep=lambda s: sleeps.append(s)),
            run_text=unittest.mock.Mock(return_value=active_windows),
            subprocess=unittest.mock.Mock(
                run=fake_run, DEVNULL=None, Popen=unittest.mock.Mock()
            ),
        )

    def test_added_retries_until_gdk_catches_up(self):
        sleeps, calls, patcher = self._patched(returncodes=(1, 1, 0))
        with patcher:
            watchers.apply_monitor_event("added", "HDMI-A-1")
        self.assertEqual(len(calls), 3, "must retry past GDK lag, not give up on the first failure")
        self.assertTrue(all(c == bar_window_command("added", "HDMI-A-1") for c in calls))

    def test_added_gives_up_after_five_attempts(self):
        sleeps, calls, patcher = self._patched(returncodes=(1, 1, 1, 1, 1, 1, 1))
        with patcher:
            watchers.apply_monitor_event("added", "HDMI-A-1")
        self.assertEqual(len(calls), 5, "must not retry forever")

    def test_added_stops_on_first_success(self):
        sleeps, calls, patcher = self._patched(returncodes=(0,))
        with patcher:
            watchers.apply_monitor_event("added", "HDMI-A-1")
        self.assertEqual(len(calls), 1)

    def test_added_skips_entirely_when_the_bar_is_already_open(self):
        # The flicker guard. Another opener won the race, so touching it at all
        # is a visible regression.
        sleeps, calls, patcher = self._patched(active_windows="bar-HDMI-A-1: bar\n")
        with patcher:
            watchers.apply_monitor_event("added", "HDMI-A-1")
        self.assertEqual(calls, [], "re-opening an open bar id makes it flicker")

    def test_it_waits_before_the_first_attempt(self):
        # The lag is before the first try, not between retries -- an
        # implementation that slept only on retry would still lose the race.
        sleeps, calls, patcher = self._patched(returncodes=(0,))
        with patcher:
            watchers.apply_monitor_event("added", "HDMI-A-1")
        self.assertEqual(sleeps, [1])

    def test_removed_does_not_retry_or_consult_open_windows(self):
        sleeps, calls, patcher = self._patched(active_windows="bar-HDMI-A-1: bar\n", returncodes=(1,))
        with patcher:
            watchers.apply_monitor_event("removed", "HDMI-A-1")
        self.assertEqual(calls, [bar_window_command("removed", "HDMI-A-1")])

    def test_a_failing_eww_never_propagates(self):
        # This runs on its own thread off the Hyprland event loop; an exception
        # here would kill hotplug handling for the rest of the session.
        def boom(*_a, **_k):
            raise OSError("eww is gone")

        with unittest.mock.patch.multiple(
            watchers,
            time=unittest.mock.Mock(sleep=lambda _s: None),
            run_text=unittest.mock.Mock(return_value=""),
            subprocess=unittest.mock.Mock(run=boom, DEVNULL=None),
        ):
            watchers.apply_monitor_event("added", "HDMI-A-1")


class ReconcileBarWindowsTests(unittest.TestCase):
    """watchers.py:118-122 -- the event that can never be observed.

    Connecting a monitor makes eww reload its whole configuration, which kills
    and respawns this backend. The monitoradded event therefore fires while no
    listener is alive. Nothing in the code hints that the startup reconcile is
    load-bearing rather than belt-and-braces; delete it and a hotplug that
    triggers a reload leaves the new screen bare.
    """

    def test_opens_bars_for_monitors_that_have_none(self):
        opened = []
        responses = {
            ("hyprctl", "monitors", "-j"): '[{"name":"eDP-1"},{"name":"HDMI-A-1"}]',
            ("eww", "active-windows"): "bar-eDP-1: bar\n",
        }
        with unittest.mock.patch.object(
            watchers, "run_text", side_effect=lambda cmd, **_k: responses[tuple(cmd)]
        ):
            with unittest.mock.patch.object(
                watchers, "apply_monitor_event", side_effect=lambda a, n: opened.append((a, n))
            ):
                watchers.reconcile_bar_windows()
        self.assertEqual(opened, [("added", "HDMI-A-1")])

    def test_does_nothing_when_every_monitor_has_a_bar(self):
        opened = []
        responses = {
            ("hyprctl", "monitors", "-j"): '[{"name":"eDP-1"}]',
            ("eww", "active-windows"): "bar-eDP-1: bar\n",
        }
        with unittest.mock.patch.object(
            watchers, "run_text", side_effect=lambda cmd, **_k: responses[tuple(cmd)]
        ):
            with unittest.mock.patch.object(
                watchers, "apply_monitor_event", side_effect=lambda a, n: opened.append((a, n))
            ):
                watchers.reconcile_bar_windows()
        self.assertEqual(opened, [])


class _FakeStdout:
    """A subprocess pipe that yields a fixed set of lines, then EOF."""

    def __init__(self, lines):
        self._lines = list(lines)

    def __iter__(self):
        return iter(self._lines)

    def readline(self):
        return ""


class _StopLoop(Exception):
    pass


class _FakeState:
    def __init__(self):
        self.updates = []

    def update(self, **items):
        self.updates.append(items)


class WatchCommandTests(unittest.TestCase):
    """The watcher loop, which has never had a test and is where a port breaks.

    In particular the line filter: without it a collector that shells out to the
    same subsystem it is watching becomes its own event source. Measured on this
    host before the filter landed, `pactl subscribe` fed the volume watcher 1248
    events a minute, all of them provoked by the watcher's own children.
    """

    def _run(self, lines, line_filter=None, collector=None):
        state = _FakeState()
        collector = collector or (lambda: "collected")

        def fake_sleep(seconds):
            if seconds == 2:  # the outer reconnect sleep: one pass is enough
                raise _StopLoop

        proc = unittest.mock.Mock(stdout=_FakeStdout(lines), wait=lambda timeout=None: 0)
        with unittest.mock.patch.multiple(
            watchers,
            time=unittest.mock.Mock(sleep=fake_sleep),
            select=unittest.mock.Mock(select=lambda *a, **k: ([], [], [])),
            subprocess=unittest.mock.Mock(
                Popen=unittest.mock.Mock(return_value=proc), PIPE=-1, DEVNULL=-3
            ),
        ):
            try:
                watchers.watch_command(state, "volume", collector, ["fake"], line_filter=line_filter)
            except _StopLoop:
                pass
        return state.updates

    def test_collects_once_before_any_event(self):
        # The bar must not sit on its :initial literal until the first event.
        updates = self._run([])
        self.assertEqual(updates, [{"volume": "collected"}])

    def test_every_line_collects_when_unfiltered(self):
        updates = self._run(["a\n", "b\n", "c\n"])
        self.assertEqual(len(updates), 4, "one initial collect plus one per line")

    def test_filtered_lines_do_not_collect(self):
        updates = self._run(
            ["Event 'change' on client #1\n", "Event 'new' on client #2\n"],
            line_filter=collectors_volume_filter(),
        )
        self.assertEqual(updates, [{"volume": "collected"}], "client events must not re-trigger")

    def test_relevant_lines_still_collect_through_the_filter(self):
        updates = self._run(
            ["Event 'change' on client #1\n", "Event 'change' on sink #0\n"],
            line_filter=collectors_volume_filter(),
        )
        self.assertEqual(len(updates), 2, "the sink event must get through")

    def test_the_real_captured_client_storm_produces_no_collects(self):
        # Shape taken from a live 90 s capture on this host: 634 client
        # lifecycles, each new/change/remove, and nothing else.
        storm = []
        for n in range(50):
            for verb in ("new", "change", "remove"):
                storm.append(f"Event '{verb}' on client #{n}\n")
        updates = self._run(storm, line_filter=collectors_volume_filter())
        self.assertEqual(updates, [{"volume": "collected"}])

    def test_a_collector_that_raises_does_not_kill_the_watcher(self):
        # Every collector shells out; a transient failure must not end the
        # thread and freeze that module for the rest of the session.
        calls = []

        def flaky():
            calls.append(1)
            raise OSError("nmcli went away")

        self._run(["a\n"], collector=flaky)
        self.assertTrue(calls, "collector should have been attempted")


def collectors_volume_filter():
    from eww_bar_backend import collectors

    return collectors.volume_event_is_relevant


class EmitLoopTests(unittest.TestCase):
    """watchers.py:242-254 -- the daemon's entire output contract.

    eww reads this process's stdout as a deflisten. Two properties matter and
    neither is obvious: the first snapshot goes out before anything waits (so
    the bar leaves its :initial literal immediately), and a closed pipe is a
    clean exit rather than a traceback -- eww closes it on every config reload.
    """

    class _Signal:
        def __init__(self, waits):
            self._waits = waits

        def wait(self, timeout=None):
            if not self._waits:
                raise _StopLoop
            self._waits.pop()

        def clear(self):
            pass

    class _State:
        def __init__(self, changed, snapshots=None):
            self.changed = changed
            self._snapshots = snapshots or []
            self.calls = 0

        def snapshot(self):
            self.calls += 1
            return f"snapshot-{self.calls}"

    def _capture(self, state):
        printed = []
        with unittest.mock.patch("builtins.print", side_effect=lambda *a, **k: printed.append(a[0])):
            try:
                watchers.emit_loop(state)
            except _StopLoop:
                pass
        return printed

    def test_emits_immediately_without_waiting(self):
        state = self._State(self._Signal([]))
        self.assertEqual(self._capture(state), ["snapshot-1"])

    def test_emits_once_per_change(self):
        state = self._State(self._Signal([None, None]))
        self.assertEqual(self._capture(state), ["snapshot-1", "snapshot-2", "snapshot-3"])

    def test_a_closed_pipe_exits_cleanly_on_the_first_emit(self):
        state = self._State(self._Signal([None]))
        with unittest.mock.patch("builtins.print", side_effect=BrokenPipeError):
            watchers.emit_loop(state)  # must return, not raise

    def test_a_closed_pipe_exits_cleanly_mid_loop(self):
        state = self._State(self._Signal([None, None]))
        calls = {"n": 0}

        def flaky_print(*_a, **_k):
            calls["n"] += 1
            if calls["n"] > 1:
                raise BrokenPipeError

        with unittest.mock.patch("builtins.print", side_effect=flaky_print):
            watchers.emit_loop(state)
        self.assertEqual(calls["n"], 2)
