"""Process-wide safety rails for the eww backend tests.

These run at package import, before any test module is loaded, and they exist
because the previous arrangement was a CONVENTION rather than an invariant.
Every test file did isolate itself correctly -- temp runtime dirs, patched
subprocess -- and it still failed, because test_go_equivalence patched the
subprocess seam for four modules under test and missed the fifth. The result
was a test run that executed `nmcli radio wifi off`, `bluetoothctl power off`,
`wpctl set-mute toggle` and `playerctl next` against a live desktop session.

A convention that has to be re-applied for every module will eventually miss
one. These two rails close the class instead:

  1. The only program the suite may execute is the Go toolchain. Anything else
     raises, naming the argv, on the first call rather than running.
  2. XDG_RUNTIME_DIR points at a throwaway directory for the whole process, so
     even a test that forgets to isolate cannot reach the running daemon's
     control socket or pidfile.

Rail 2 matters more than it looks. control_server() opens by unlinking the
socket path and then binding it, so a test that reached the real path would
delete the live daemon's socket and silently take over the bar's control
channel -- every eww-barctl call from the bar would hit the test.
"""

import atexit
import os
import shutil
import subprocess
import tempfile
from contextlib import contextmanager

# -- rail 1: no subprocess except the Go toolchain ---------------------------

_real_run = subprocess.run
_real_popen = subprocess.Popen

# test_go_equivalence drives the ported code through `go run`, which is the one
# legitimate child process in the suite.
_ALLOWED_BASENAMES = frozenset({"go"})

_escape_allowed = False


class SubprocessEscape(BaseException):
    """Raised when the suite tries to execute a disallowed program.

    A BaseException, NOT an Exception, and that is the whole point. The modules
    under test swallow exceptions on purpose -- wallpaper.set_wallpaper wraps
    its subprocess.run in `except Exception: pass`, and
    serve_control_connection turns any Exception into an error reply. An
    Exception-based guard would be absorbed by the very code it is meant to
    police, leaving the test green or merely puzzling. This one propagates
    through those handlers and names the command.
    """



def _argv0(args):
    if isinstance(args, (list, tuple)) and args:
        return os.path.basename(str(args[0]))
    if isinstance(args, (str, bytes, os.PathLike)):
        return os.path.basename(os.fsdecode(args))
    return ""


def _check(args):
    if _escape_allowed or _argv0(args) in _ALLOWED_BASENAMES:
        return
    raise SubprocessEscape(
        f"the test suite tried to execute {args!r}.\n"
        "Only the Go toolchain may be run. A module under test is holding an "
        "unpatched `subprocess` -- patch it in the test's fixture rather than "
        "letting the command reach the developer's session. If a real command "
        "is genuinely required, wrap the call in "
        "tests.eww_bar_backend.allow_real_subprocess() so the exception is "
        "visible in the test that wants it."
    )


def _guarded_run(*args, **kwargs):
    _check(args[0] if args else kwargs.get("args"))
    return _real_run(*args, **kwargs)


class _GuardedPopen(_real_popen):
    def __init__(self, *args, **kwargs):
        _check(args[0] if args else kwargs.get("args"))
        super().__init__(*args, **kwargs)


subprocess.run = _guarded_run
subprocess.Popen = _GuardedPopen


@contextmanager
def allow_real_subprocess():
    """Opt out of rail 1 for one block, visibly and narrowly."""
    global _escape_allowed
    previous, _escape_allowed = _escape_allowed, True
    try:
        yield
    finally:
        _escape_allowed = previous


# -- rail 2: never the real runtime directory --------------------------------

# Unconditional rather than "only when it looks like the live one". A
# conditional rail is a convention again: it would pass on a machine with no
# daemon running and fail to protect the one machine that has it.
_runtime_dir = tempfile.mkdtemp(prefix="eww-tests-runtime-")
os.environ["XDG_RUNTIME_DIR"] = _runtime_dir
atexit.register(shutil.rmtree, _runtime_dir, ignore_errors=True)
