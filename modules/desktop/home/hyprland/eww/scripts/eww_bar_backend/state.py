import json
import threading

from .collectors import clock_state
from .common import BATTERY_DEFAULT, EMPTY_MODULE, MEDIA_DEFAULT, WORKSPACE_DEFAULT


class BarState:
    def __init__(self):
        self.lock = threading.Lock()
        self.state = {
            "workspace_state": WORKSPACE_DEFAULT.copy(),
            "submap": "",
            "clock": clock_state(),
            "media": MEDIA_DEFAULT.copy(),
            "cpu": " --%",
            "memory": EMPTY_MODULE.copy(),
            "temperature": {"text": " --°C", "class": ""},
            "network": EMPTY_MODULE.copy(),
            "volume": "",
            "battery": BATTERY_DEFAULT.copy(),
            "bluetooth": EMPTY_MODULE.copy(),
            "idle_inhibited": "false",
        }
        self.changed = threading.Event()

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
