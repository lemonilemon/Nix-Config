"""The control server must not drop commands under a burst.

`eww.yuck` binds `eww-barctl` to :onscroll, and a scroll wheel delivers ticks
faster than one collector round trip. The original loop accepted one connection,
served it to completion, and only then accepted the next -- so with listen(8),
any burst that outran the handler filled the backlog and the kernel refused the
rest. Measured against a faithful replica at a 34 ms service time: a 30-client
burst got 10 through and refused 20; at 60, it refused 50.

That never bit in production only because the Python client took ~45 ms to reach
connect(). A compiled client reaches it in ~3 ms, which is exactly the change
this backend is heading for -- so the fix has to land first.
"""

import concurrent.futures
import json
import shutil
import socket
import sys
import tempfile
import threading
import time
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import control  # noqa: E402


class _SlowState:
    """Stands in for BarState, but makes every command take a collector's worth
    of time so the burst actually contends."""

    SERVICE_SECONDS = 0.05

    def __init__(self):
        self.updates = []
        self.lock = threading.Lock()

    def update(self, **items):
        time.sleep(self.SERVICE_SECONDS)
        with self.lock:
            self.updates.append(items)


class ControlServerBurstTests(unittest.TestCase):
    def setUp(self):
        # A short path: AF_UNIX addresses cap out around 108 bytes.
        self.tmpdir = Path(tempfile.mkdtemp(prefix="ewwctl-"))
        self.socket_path = self.tmpdir / "burst.sock"
        self.addCleanup(shutil.rmtree, self.tmpdir, ignore_errors=True)

        self.state = _SlowState()
        patcher = unittest.mock.patch.object(
            control, "control_socket_path", return_value=self.socket_path
        )
        patcher.start()
        self.addCleanup(patcher.stop)

        # A slow command: handle_control_command -> state.update -> 50 ms.
        handler = unittest.mock.patch.object(
            control,
            "handle_control_command",
            side_effect=lambda state, payload: (state.update(**payload), {"ok": True})[1],
        )
        handler.start()
        self.addCleanup(handler.stop)

        self.thread = threading.Thread(
            target=control.control_server, args=(self.state,), daemon=True
        )
        self.thread.start()
        for _ in range(200):
            if self.socket_path.exists():
                break
            time.sleep(0.01)
        self.assertTrue(self.socket_path.exists(), "control server never bound its socket")

    def _one_command(self, index):
        try:
            client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            client.settimeout(10.0)
            client.connect(str(self.socket_path))
            client.sendall(json.dumps({"command": "ping", "n": index}).encode() + b"\n")
            client.shutdown(socket.SHUT_WR)
            chunks = []
            while True:
                chunk = client.recv(4096)
                if not chunk:
                    break
                chunks.append(chunk)
            client.close()
            return json.loads(b"".join(chunks).decode().strip())
        except Exception as exc:  # noqa: BLE001 - the failure mode is the assertion
            return {"ok": False, "error": f"{type(exc).__name__}: {exc}"}

    def test_a_scroll_sized_burst_loses_nothing(self):
        burst = 40
        with concurrent.futures.ThreadPoolExecutor(max_workers=burst) as pool:
            responses = list(pool.map(self._one_command, range(burst)))

        failures = [r for r in responses if not r.get("ok")]
        self.assertEqual(failures, [], f"{len(failures)}/{burst} commands were dropped")
        self.assertEqual(len(self.state.updates), burst)

    def test_the_burst_is_served_concurrently_not_serially(self):
        # 20 commands x 50 ms each is 1.0 s serially. Concurrency should beat
        # that comfortably; the generous bound keeps this from flaking on a
        # loaded machine while still failing outright on a serial loop.
        burst = 20
        started = time.perf_counter()
        with concurrent.futures.ThreadPoolExecutor(max_workers=burst) as pool:
            responses = list(pool.map(self._one_command, range(burst)))
        elapsed = time.perf_counter() - started

        self.assertTrue(all(r.get("ok") for r in responses))
        serial = burst * _SlowState.SERVICE_SECONDS
        self.assertLess(
            elapsed,
            serial * 0.6,
            f"{burst} commands took {elapsed:.2f}s; serial would be {serial:.2f}s",
        )


if __name__ == "__main__":
    unittest.main()
