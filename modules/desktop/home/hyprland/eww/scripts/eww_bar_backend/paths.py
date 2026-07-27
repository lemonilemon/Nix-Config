"""Runtime path helpers for the daemon.

The control socket path is duplicated in the Go client (client/ctl.go); these
two are the only definitions, and client/ctl_test.go pins the value. Keep them
in step or eww-barctl talks to a socket nobody is listening on.
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
