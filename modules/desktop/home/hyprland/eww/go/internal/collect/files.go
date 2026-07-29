package collect

import (
	"fmt"
	"math"
	"strconv"
)

const (
	glyphCPU         = "\uF2DB" // U+F2DB microchip
	glyphTemperature = "\uF2CB" // U+F2CB thermometer
	degreeSign       = "\u00B0" // U+00B0 degree sign

	glyphBatteryCharging = "\U000F0084" // U+F0084 battery-charging
	glyphBattery20       = "\U000F007B" // U+F007B battery-20
	glyphBattery40       = "\U000F007C" // U+F007C battery-40
	glyphBattery60       = "\U000F007E" // U+F007E battery-60
	glyphBattery80       = "\U000F0080" // U+F0080 battery-80
	glyphBatteryFull     = "\U000F0079" // U+F0079 battery-full
	emDash               = "\u2014"     // U+2014 em dash -- BATTERY_DEFAULT["power"]
)

// BatteryDefault is common.BATTERY_DEFAULT.
func BatteryDefault() Battery {
	return Battery{
		Text:     glyphBatteryCharging,
		Alt:      glyphBatteryCharging,
		Capacity: 100,
		Class:    "charging",
		Status:   "Unknown",
		Time:     "N/A",
		Health:   "--",
		Power:    emDash,
	}
}

// Battery is collectors.battery_state's shape. Declared here rather than in
// package state so the collector and its default sit together.
type Battery struct {
	Text     string `json:"text"`
	Alt      string `json:"alt"`
	Capacity int    `json:"capacity"`
	Class    string `json:"class"`
	Status   string `json:"status"`
	Time     string `json:"time"`
	Health   string `json:"health"`
	Power    string `json:"power"`
}

// CPUStateFromSamples mirrors collectors.cpu_state's arithmetic.
//
// The original reads /proc/stat, sleeps 200 ms, reads again. Taking the two
// samples as arguments keeps the maths testable without a sleep in the test;
// CollectCPU does the reading.
func CPUStateFromSamples(first, second string) string {
	total1, idle1, ok1 := parseProcStat(first)
	total2, idle2, ok2 := parseProcStat(second)
	if !ok1 || !ok2 {
		return glyphCPU + " --%"
	}
	totalDelta := total2 - total1
	idleDelta := idle2 - idle1
	if totalDelta <= 0 {
		return glyphCPU + " --%"
	}
	usage := float64(totalDelta-idleDelta) * 100 / float64(totalDelta)
	return fmt.Sprintf("%s %.0f%%", glyphCPU, usage)
}

// parseProcStat reads the aggregate cpu line: the first seven fields, with
// idle being fields 4 and 5 (idle + iowait) as the original has it.
func parseProcStat(text string) (total, idle int64, ok bool) {
	lines := SplitLines(text)
	if len(lines) == 0 {
		return 0, 0, false
	}
	fields := SplitWhitespaceN(lines[0], -1)
	if len(fields) < 8 {
		return 0, 0, false
	}
	values := make([]int64, 0, 7)
	for _, field := range fields[1:8] {
		value, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		values = append(values, value)
	}
	idle = values[3] + values[4]
	for _, value := range values {
		total += value
	}
	return total, idle, true
}

// CollectCPU mirrors collectors.cpu_state.
func CollectCPU(sleep func()) string {
	first, ok := ReadTextFile("/proc/stat")
	if !ok {
		return glyphCPU + " --%"
	}
	sleep()
	second, ok := ReadTextFile("/proc/stat")
	if !ok {
		return glyphCPU + " --%"
	}
	return CPUStateFromSamples(first, second)
}

// CollectMemory mirrors collectors.memory_state.
func CollectMemory() Module {
	text, ok := ReadTextFile("/proc/meminfo")
	if !ok {
		text = ""
	}
	return MemoryStateFromText(text)
}

// TemperatureStateFromReadings mirrors collectors.temperature_state's tail: it
// takes every hwmon reading in millidegrees and renders the hottest.
func TemperatureStateFromReadings(millidegrees []int64) Temperature {
	if len(millidegrees) == 0 {
		return Temperature{Text: glyphTemperature + " --" + degreeSign + "C", Class: ""}
	}
	hottest := millidegrees[0]
	for _, value := range millidegrees[1:] {
		if value > hottest {
			hottest = value
		}
	}
	// int(round(x)) on the degrees, so half-to-even like everywhere else.
	rounded := RoundHalfEven(float64(hottest) / 1000)
	class := ""
	if rounded >= 90 {
		class = "critical"
	}
	return Temperature{
		Text:  fmt.Sprintf("%s %d%sC", glyphTemperature, rounded, degreeSign),
		Class: class,
	}
}

// Temperature is collectors.temperature_state's shape.
type Temperature struct {
	Text  string `json:"text"`
	Class string `json:"class"`
}

// BatteryStateFromFiles mirrors collectors.battery_state's body for one battery
// directory, taking the file contents rather than reading them.
//
// files is keyed by the file name under /sys/class/power_supply/BAT*, e.g.
// "capacity", "status", "energy_now". A missing key is a missing file.
func BatteryStateFromFiles(files map[string]string) (Battery, bool) {
	capacityText, present := files["capacity"]
	if !present {
		return Battery{}, false
	}
	capacity, err := strconv.ParseInt(Strip(capacityText), 10, 64)
	if err != nil {
		return Battery{}, false
	}
	status := Strip(files["status"])

	var icon string
	switch {
	case status == "Charging":
		icon = glyphBatteryCharging
	case capacity < 20:
		icon = glyphBattery20
	case capacity < 40:
		icon = glyphBattery40
	case capacity < 60:
		icon = glyphBattery60
	case capacity < 80:
		icon = glyphBattery80
	default:
		icon = glyphBatteryFull
	}

	// The original tries the energy_* family first and falls back to charge_*,
	// because laptops report one or the other and the unit decides how watts
	// are computed further down.
	var now, rate, fullValue, designValue, voltage int64
	var haveNow, haveRate, haveFull, haveDesign, haveVoltage bool
	unit := ""
	for _, family := range [][5]string{
		{"energy", "energy_now", "power_now", "energy_full", "energy_full_design"},
		{"charge", "charge_now", "current_now", "charge_full", "charge_full_design"},
	} {
		n, nOK := intFile(files, family[1])
		r, rOK := intFile(files, family[2])
		f, fOK := intFile(files, family[3])
		if !nOK || !rOK || !fOK {
			now, rate, fullValue = 0, 0, 0
			haveNow, haveRate, haveFull = false, false, false
			continue
		}
		now, rate, fullValue = n, r, f
		haveNow, haveRate, haveFull = true, true, true
		unit = family[0]
		designValue, haveDesign = intFile(files, family[4])
		if unit == "charge" {
			voltage, haveVoltage = intFile(files, "voltage_now")
		}
		break
	}

	timeStr := "N/A"
	var minutes int64
	haveMinutes := false
	if haveNow && haveRate && rate > 0 {
		if status == "Discharging" {
			minutes = now * 60 / rate
			haveMinutes = true
		} else if status == "Charging" && haveFull && fullValue != 0 {
			minutes = (fullValue - now) * 60 / rate
			haveMinutes = true
		}
	}
	if haveMinutes {
		timeStr = fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
	}

	// Health is charge_full over charge_full_design, which is a measurement only
	// if the firmware actually maintains charge_full. Plenty do not -- they echo
	// the design value forever -- and this laptop is one of them: 357 cycles in,
	// charge_full is 3083000 against a charge_full_design of 3083000, identical
	// to the microamp-hour. No real cell is at exactly 100.000% of design after
	// 357 cycles; upower reads the same pair and reports the same 100%.
	//
	// So an exact match is treated as no data rather than as perfect health.
	// Rendering a constant as though it were a reading is worse than admitting
	// the firmware does not say.
	//
	// A genuinely new battery also reads full == design and is suppressed by the
	// same rule. That is the right direction to be wrong in: an unknown that is
	// really 100% costs nothing, while a permanent 100% on a worn cell is a lie
	// you would act on.
	health := "--"
	if haveFull && fullValue != 0 && haveDesign && designValue > 0 && fullValue != designValue {
		health = fmt.Sprintf("%d%%", RoundHalfEven(float64(fullValue)*100/float64(designValue)))
	}

	power := emDash
	if haveRate && rate > 0 {
		watts := math.NaN()
		if unit == "energy" {
			watts = float64(rate) / 1_000_000
		} else if unit == "charge" && haveVoltage && voltage != 0 {
			watts = float64(rate) * float64(voltage) / 1_000_000_000_000
		}
		if !math.IsNaN(watts) {
			power = fmt.Sprintf("%.1f W", watts)
		}
	}

	class := ""
	switch {
	case status == "Charging":
		class = "charging"
	case capacity < 20:
		class = "critical"
	case capacity < 30:
		class = "warning"
	}

	displayStatus := status
	if displayStatus == "" {
		displayStatus = "Unknown"
	}
	return Battery{
		Text:     fmt.Sprintf("%s %d%%", icon, capacity),
		Alt:      fmt.Sprintf("%s %s", icon, timeStr),
		Capacity: int(capacity),
		Class:    class,
		Status:   displayStatus,
		Time:     timeStr,
		Health:   health,
		Power:    power,
	}, true
}

func intFile(files map[string]string, name string) (int64, bool) {
	text, present := files[name]
	if !present {
		return 0, false
	}
	value, err := strconv.ParseInt(Strip(text), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
