package collect

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// The two collectors that walk sysfs. Both glob, so they get a seam of their
// own: ListDir answers one directory, and these need a pattern across two
// levels.

// GlobFiles is the seam for pathlib's Path.glob.
var GlobFiles = func(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	sort.Strings(matches)
	return matches
}

// ModuleEnabled mirrors common.module_enabled.
//
// default.nix writes exactly "1" or "0" into EWW_BAR_<NAME> on the eww-bar
// service, so only that literal "0" disables. An absent variable means enabled,
// which keeps the backend usable when run by hand outside the unit.
func ModuleEnabled(name string) bool {
	return os.Getenv("EWW_BAR_"+name) != "0"
}

// CollectTemperature mirrors collectors.temperature_state.
func CollectTemperature() Temperature {
	readings := []int64{}
	for _, path := range GlobFiles("/sys/class/hwmon/hwmon*/temp*_input") {
		text, ok := ReadTextFile(path)
		if !ok {
			continue
		}
		value, err := strconv.ParseInt(Strip(text), 10, 64)
		if err != nil {
			continue
		}
		readings = append(readings, value)
	}
	return TemperatureStateFromReadings(readings)
}

// batteryFiles are every file BatteryStateFromFiles may look at. Read eagerly
// rather than on demand because the pure half takes a map, and a battery
// directory holds a dozen files at most.
var batteryFiles = []string{
	"capacity", "status", "voltage_now",
	"energy_now", "power_now", "energy_full", "energy_full_design",
	"charge_now", "current_now", "charge_full", "charge_full_design",
}

// CollectBattery mirrors collectors.battery_state.
//
// The EWW_BAR_BATTERY gate is what keeps a desktop from rendering a battery
// module that would read as "100%, charging" forever, and it is checked before
// any filesystem work.
func CollectBattery() Battery {
	if !ModuleEnabled("BATTERY") {
		return BatteryDefault()
	}
	for _, dir := range GlobFiles("/sys/class/power_supply/BAT*") {
		files := map[string]string{}
		for _, name := range batteryFiles {
			if text, ok := ReadTextFile(filepath.Join(dir, name)); ok {
				files[name] = text
			}
		}
		if battery, ok := BatteryStateFromFiles(files); ok {
			return battery
		}
	}
	return BatteryDefault()
}
