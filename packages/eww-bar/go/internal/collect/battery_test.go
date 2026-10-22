package collect

import "testing"

// TestBatteryHealthNeedsAWornCell pins the rule that keeps the Health row from
// reporting a constant as a measurement.
//
// charge_full over charge_full_design is a real metric only where the firmware
// maintains charge_full. Many laptops never touch it, so the pair stays exactly
// equal for the life of the machine and the quotient is 100% forever. The
// machine this bar runs on is one: 357 cycles in, charge_full is 3083000 and
// charge_full_design is 3083000, and upower reports the same misleading 100%.
//
// An exact match therefore reads as "no data" rather than "perfect health". A
// new battery is suppressed by the same rule, which is the safe direction: an
// unknown that is really 100% costs nothing, a permanent 100% on a worn cell is
// a number you would act on.
func TestBatteryHealthNeedsAWornCell(t *testing.T) {
	// The real values off this laptop.
	base := map[string]string{
		"capacity":           "98",
		"status":             "Charging",
		"charge_now":         "3020000",
		"current_now":        "1000000",
		"charge_full":        "3083000",
		"charge_full_design": "3083000",
		"voltage_now":        "11550000",
	}

	cases := []struct {
		name       string
		full       string
		design     string
		wantHealth string
	}{
		{
			name:       "firmware echoes the design value",
			full:       "3083000",
			design:     "3083000",
			wantHealth: "--",
		},
		{
			name:       "firmware tracks wear",
			full:       "2775000",
			design:     "3083000",
			wantHealth: "90%",
		},
		{
			// Some firmwares report slightly above design when new or after a
			// recalibration. That is still a real reading, so it is still shown.
			name:       "above design is still a reading",
			full:       "3100000",
			design:     "3083000",
			wantHealth: "101%",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := map[string]string{}
			for k, v := range base {
				files[k] = v
			}
			files["charge_full"] = c.full
			files["charge_full_design"] = c.design

			battery, ok := BatteryStateFromFiles(files)
			if !ok {
				t.Fatal("BatteryStateFromFiles rejected a well-formed battery")
			}
			if battery.Health != c.wantHealth {
				t.Errorf("charge_full=%s design=%s: health = %q, want %q",
					c.full, c.design, battery.Health, c.wantHealth)
			}
		})
	}
}

// TestBatteryHealthWithoutDesignIsUnknown covers the other way the reading goes
// missing: a battery directory with no *_full_design file at all.
func TestBatteryHealthWithoutDesignIsUnknown(t *testing.T) {
	battery, ok := BatteryStateFromFiles(map[string]string{
		"capacity":    "55",
		"status":      "Discharging",
		"charge_now":  "1700000",
		"current_now": "900000",
		"charge_full": "3083000",
		"voltage_now": "11550000",
	})
	if !ok {
		t.Fatal("BatteryStateFromFiles rejected a well-formed battery")
	}
	if battery.Health != "--" {
		t.Errorf("health = %q, want %q with no design capacity on disk",
			battery.Health, "--")
	}
}
