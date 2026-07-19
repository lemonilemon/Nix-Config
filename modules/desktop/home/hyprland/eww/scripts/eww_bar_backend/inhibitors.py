import subprocess


HYPRIDLE_INHIBIT_SERVICE = "eww-hypridle-inhibit.service"
LID_INHIBIT_SERVICE = "eww-lid-inhibit.service"


def bool_state(value):
    return "true" if value else "false"


def service_active(service):
    result = subprocess.run(
        ["systemctl", "--user", "is-active", "--quiet", service],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return result.returncode == 0


def set_service_active(service, enabled):
    action = "start" if enabled else "stop"
    result = subprocess.run(
        ["systemctl", "--user", action, service],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(f"systemctl --user {action} {service} failed")
    return service_active(service)


def idle_inhibited_state():
    return bool_state(service_active(HYPRIDLE_INHIBIT_SERVICE))


def lid_inhibited_state():
    return bool_state(service_active(LID_INHIBIT_SERVICE))


def set_idle_inhibited(enabled):
    return bool_state(set_service_active(HYPRIDLE_INHIBIT_SERVICE, enabled))


def set_lid_inhibited(enabled):
    return bool_state(set_service_active(LID_INHIBIT_SERVICE, enabled))


def toggle_idle_inhibited():
    return set_idle_inhibited(not service_active(HYPRIDLE_INHIBIT_SERVICE))
