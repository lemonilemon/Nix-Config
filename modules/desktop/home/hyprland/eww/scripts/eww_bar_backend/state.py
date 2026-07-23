import json
import threading

from .collectors import clock_state
from .common import (
    AI_USAGE_DEFAULT,
    BATTERY_DEFAULT,
    BLUETOOTH_DEFAULT,
    EMPTY_MODULE,
    MEDIA_DEFAULT,
    NETWORK_DEFAULT,
    VOLUME_DEFAULT,
    WORKSPACE_DEFAULT,
)
from .notifications import NOTIFICATIONS_DEFAULT


class BarState:
    def __init__(self):
        self.lock = threading.Lock()
        self.state = {
            "workspace_state": WORKSPACE_DEFAULT.copy(),
            "submap": "",
            "clock": clock_state(),
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
