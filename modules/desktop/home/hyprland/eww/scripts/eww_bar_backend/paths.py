"""Runtime path helpers shared by the daemon and the eww-barctl click path.

Deliberately dependency-free — only os and pathlib — so ctl.py can import
control_socket_path from here without pulling in common.py (and through it
subprocess).
"""

import os
from pathlib import Path


def runtime_file(name):
    runtime_dir = os.environ.get("XDG_RUNTIME_DIR", "/tmp")
    return Path(runtime_dir) / name


def display_mode_path():
    return runtime_file("eww-display-mode")


def backend_pidfile_path():
    return runtime_file("eww-backend.pid")


def control_socket_path():
    return runtime_file("eww-backend.sock")
