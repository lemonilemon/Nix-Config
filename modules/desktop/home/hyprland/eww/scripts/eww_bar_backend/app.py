import signal
import threading

from .collectors import (
    active_window_state,
    battery_state,
    bluetooth_state,
    clock_state,
    cpu_state,
    media_state,
    memory_state,
    network_state,
    refresh_ai_usage,
    temperature_state,
    tray_count,
    volume_state,
    workspace_state,
)
from .control import control_server, write_backend_pidfile
from .display import display_state
from .inhibitors import idle_inhibited_state
from .notifications import notifications_state
from .state import BarState
from .wallpaper import wallpaper_state
from .watchers import (
    emit_loop,
    periodic,
    periodic_refresh,
    watch_bluetooth,
    watch_command,
    watch_hyprland,
    watch_idle_inhibitor,
)


signal.signal(signal.SIGPIPE, signal.SIG_IGN)


def run_bar():
    idle_refresh = threading.Event()
    signal.signal(signal.SIGUSR1, lambda _signum, _frame: idle_refresh.set())
    write_backend_pidfile()

    state = BarState()
    threading.Thread(target=control_server, args=(state,), daemon=True).start()

    state.update(
        active_window=active_window_state(),
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
        display=display_state(),
        workspace_state=workspace_state(),
        notifications=notifications_state(),
    )

    threads = [
        threading.Thread(target=watch_hyprland, args=(state,), daemon=True),
        threading.Thread(target=periodic, args=(state, 60), kwargs={"clock": clock_state}, daemon=True),
        threading.Thread(
            target=periodic_refresh,
            args=(state, 300, refresh_ai_usage),
            daemon=True,
        ),
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
            kwargs={"battery": battery_state, "notifications": notifications_state},
            daemon=True,
        ),
        threading.Thread(target=watch_idle_inhibitor, args=(state, idle_refresh), daemon=True),
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
        threading.Thread(
            target=periodic,
            args=(state, 5),
            kwargs={"tray_count": tray_count},
            daemon=True,
        ),
        threading.Thread(
            target=watch_command,
            args=(
                state,
                "notifications",
                notifications_state,
                ["dbus-monitor", "--profile", "interface='org.freedesktop.Notifications'"],
            ),
            daemon=True,
        ),
        threading.Thread(
            target=lambda: state.update(wallpaper=wallpaper_state()),
            daemon=True,
        ),
    ]
    for thread in threads:
        thread.start()

    emit_loop(state)
