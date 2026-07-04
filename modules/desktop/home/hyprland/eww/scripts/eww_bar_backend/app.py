import signal
import sys
import threading
from pathlib import Path

from .collectors import (
    battery_state,
    bluetooth_state,
    clock_state,
    cpu_state,
    idle_inhibited_state,
    media_state,
    memory_state,
    network_state,
    temperature_state,
    volume_state,
    workspace_state,
)
from .control import control_server, run_ctl, write_backend_pidfile
from .state import BarState
from .watchers import (
    emit_loop,
    periodic,
    watch_bluetooth,
    watch_command,
    watch_hyprland,
    watch_idle_inhibitor,
)


signal.signal(signal.SIGPIPE, signal.SIG_DFL)


def run_bar():
    idle_refresh = threading.Event()
    signal.signal(signal.SIGUSR1, lambda _signum, _frame: idle_refresh.set())
    write_backend_pidfile()

    state = BarState()
    state.update(
        clock=clock_state(),
        media=media_state(),
        cpu=cpu_state(),
        memory=memory_state(),
        temperature=temperature_state(),
        network=network_state(),
        volume=volume_state(),
        battery=battery_state(),
        bluetooth=bluetooth_state(),
        idle_inhibited=idle_inhibited_state(),
        workspace_state=workspace_state(),
    )

    threads = [
        threading.Thread(target=watch_hyprland, args=(state,), daemon=True),
        threading.Thread(target=periodic, args=(state, 60), kwargs={"clock": clock_state}, daemon=True),
        threading.Thread(
            target=periodic,
            args=(state, 7),
            kwargs={"cpu": cpu_state, "memory": memory_state},
            daemon=True,
        ),
        threading.Thread(
            target=periodic,
            args=(state, 15),
            kwargs={"temperature": temperature_state},
            daemon=True,
        ),
        threading.Thread(
            target=periodic,
            args=(state, 30),
            kwargs={"battery": battery_state},
            daemon=True,
        ),
        threading.Thread(target=watch_idle_inhibitor, args=(state, idle_refresh), daemon=True),
        threading.Thread(target=control_server, args=(state,), daemon=True),
        threading.Thread(
            target=periodic,
            args=(state, 60),
            kwargs={"network": network_state},
            daemon=True,
        ),
        threading.Thread(
            target=watch_command,
            args=(state, "network", network_state, ["nmcli", "monitor"]),
            daemon=True,
        ),
        threading.Thread(
            target=watch_command,
            args=(state, "volume", volume_state, ["pactl", "subscribe"]),
            daemon=True,
        ),
        threading.Thread(
            target=watch_command,
            args=(
                state,
                "media",
                media_state,
                ["playerctl", "-F", "metadata", "--format", "{{playerName}} {{status}} {{artist}} {{title}}"],
            ),
            daemon=True,
        ),
        threading.Thread(target=watch_bluetooth, args=(state,), daemon=True),
    ]
    for thread in threads:
        thread.start()

    emit_loop(state)


def main():
    if Path(sys.argv[0]).name == "eww-barctl":
        return run_ctl(sys.argv[1:])

    mode = sys.argv[1] if len(sys.argv) > 1 else "bar"
    if mode == "bar":
        run_bar()
    elif mode == "ctl":
        return run_ctl(sys.argv[2:])
    else:
        print(f"unknown backend mode: {mode}", file=sys.stderr)
        return 1
    return 0
