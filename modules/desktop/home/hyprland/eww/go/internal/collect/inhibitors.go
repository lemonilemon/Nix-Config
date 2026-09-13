package collect

import (
	"fmt"
	"time"
)

// The two systemd user units that hold the idle and lid inhibitors. Starting
// and stopping a unit IS the state: nothing is cached, because the user can
// change it from outside the bar.

const (
	hypridleInhibitService = "eww-hypridle-inhibit.service"
	lidInhibitService      = "eww-lid-inhibit.service"
)

// systemctlTimeout exists because a hung systemctl would otherwise block a
// control-socket worker forever, and the socket has a bounded pool.
const systemctlTimeout = 5 * time.Second

// BoolState renders a flag the way eww reads it, as a string.
func BoolState(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func ServiceActive(service string) bool {
	return RunStatus(systemctlTimeout, "systemctl", "--user", "is-active", "--quiet", service) == 0
}

// SetServiceActive returns the state it reads back rather than the state it asked
// for: `systemctl start` on a unit whose ExecStart fails still exits zero for a
// Type=oneshot with RemainAfterExit, so the only honest answer is to ask again.
func SetServiceActive(service string, enabled bool) (bool, error) {
	action := "stop"
	if enabled {
		action = "start"
	}
	if code := RunStatus(systemctlTimeout, "systemctl", "--user", action, service); code != 0 {
		return false, fmt.Errorf("systemctl --user %s %s failed", action, service)
	}
	return ServiceActive(service), nil
}

func IdleInhibitedState() string { return BoolState(ServiceActive(hypridleInhibitService)) }

func LidInhibitedState() string { return BoolState(ServiceActive(lidInhibitService)) }

func SetIdleInhibited(enabled bool) (string, error) {
	active, err := SetServiceActive(hypridleInhibitService, enabled)
	if err != nil {
		return "", err
	}
	return BoolState(active), nil
}

func SetLidInhibited(enabled bool) (string, error) {
	active, err := SetServiceActive(lidInhibitService, enabled)
	if err != nil {
		return "", err
	}
	return BoolState(active), nil
}

func ToggleIdleInhibited() (string, error) {
	return SetIdleInhibited(!ServiceActive(hypridleInhibitService))
}
