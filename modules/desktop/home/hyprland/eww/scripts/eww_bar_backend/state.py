import json
import threading

from .common import (
    ACTIVE_WINDOW_DEFAULT,
    AI_USAGE_DEFAULT,
    BATTERY_DEFAULT,
    BLUETOOTH_DEFAULT,
    CLOCK_DEFAULT,
    EMPTY_MODULE,
    MEDIA_DEFAULT,
    NETWORK_DEFAULT,
    VOLUME_DEFAULT,
    WORKSPACE_DEFAULT,
)
from .notifications import NOTIFICATIONS_DEFAULT
from .wallpaper import WALLPAPER_DEFAULT


class BarState:
    def __init__(self):
        self.lock = threading.Lock()
        self.state = {
            "active_window": ACTIVE_WINDOW_DEFAULT.copy(),
            "workspace_state": WORKSPACE_DEFAULT.copy(),
            "submap": "",
            # Placeholder, not a live reading: app.run_bar overwrites this via
            # its first state.update within milliseconds, and a time-dependent
            # constructor would make the eww.yuck :initial literal unassertable.
            "clock": CLOCK_DEFAULT.copy(),
            "media": MEDIA_DEFAULT.copy(),
            "ai_usage": AI_USAGE_DEFAULT.copy(),
            "cpu": " --%",
            "memory": EMPTY_MODULE.copy(),
            "temperature": {"text": " --°C", "class": ""},
            "network": NETWORK_DEFAULT.copy(),
            "volume": VOLUME_DEFAULT.copy(),
            "battery": BATTERY_DEFAULT.copy(),
            "bluetooth": BLUETOOTH_DEFAULT.copy(),
            "tray_count": 0,
            "notifications": dict(NOTIFICATIONS_DEFAULT, groups=[]),
            "wallpaper": dict(WALLPAPER_DEFAULT, rows=[]),
            "idle_inhibited": "false",
            "display": {
                "mode": "normal",
                "lid_inhibited": "false",
                "status": "Normal desktop mode",
                "class": "normal",
            },
        }
        self.changed = threading.Event()

    def get(self, key, default=None):
        with self.lock:
            return self.state.get(key, default)

    def update(self, **items):
        with self.lock:
            changed = False
            for key, value in items.items():
                if self.state.get(key) != value:
                    self.state[key] = value
                    changed = True
        if changed:
            self.changed.set()

    def snapshot(self):
        with self.lock:
            return json.dumps(self.state, separators=(",", ":"), ensure_ascii=False)
