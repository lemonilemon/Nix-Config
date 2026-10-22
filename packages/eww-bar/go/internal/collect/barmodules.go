package collect

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	"ewwbar/internal/pyjson"
)

// modules.json is also what bar-modules.nix reads, so the list exists once.
//
//go:embed modules.json
var barModulesJSON []byte

// barModuleNames is every bar module the bar knows, in bar order. The
// bar_module_row calls in eww.yuck still name each one.
var barModuleNames = parseBarModuleNames(barModulesJSON)

func parseBarModuleNames(data []byte) []string {
	var names []string
	if err := json.Unmarshal(data, &names); err != nil || len(names) == 0 {
		panic("collect: modules.json is not a non-empty JSON list of names")
	}
	return names
}

// EnabledBarModules parses EWW_BAR_MODULES, the comma-separated modules this
// host builds the bar with. Unset or empty means every module.
func EnabledBarModules(env string) map[string]bool {
	enabled := map[string]bool{}
	for _, name := range strings.Split(env, ",") {
		if name = strings.TrimSpace(name); name != "" {
			enabled[name] = true
		}
	}
	if len(enabled) == 0 {
		for _, name := range barModuleNames {
			enabled[name] = true
		}
	}
	return enabled
}

var barModulesEnabled = EnabledBarModules(os.Getenv("EWW_BAR_MODULES"))

// BarModuleEnabled reports whether this host's bar has the module at all. A
// module left out has no widget, no Settings row and no collectors.
func BarModuleEnabled(name string) bool {
	return barModulesEnabled[name]
}

// barModulesVersion is the on-disk version of the overrides file.
const barModulesVersion = 1

// ErrUnknownBarModule is a toggle naming a module the bar does not have,
// either one it never knew or one this host leaves out.
var ErrUnknownBarModule = errors.New("unknown bar module")

func isBarModule(name string) bool {
	if !barModulesEnabled[name] {
		return false
	}
	for _, known := range barModuleNames {
		if known == name {
			return true
		}
	}
	return false
}

// DefaultHiddenBarModules parses EWW_BAR_DEFAULT_HIDDEN, a comma-separated list.
func DefaultHiddenBarModules(env string) map[string]bool {
	hidden := map[string]bool{}
	for _, name := range strings.Split(env, ",") {
		if name = strings.TrimSpace(name); isBarModule(name) {
			hidden[name] = true
		}
	}
	return hidden
}

// ResolveBarModules is the visibility eww renders: the host's defaults with the
// user's overrides on top, as "true"/"false" per module in bar order.
func ResolveBarModules(defaultHidden, overrides map[string]bool) pyjson.Ordered {
	resolved := make(pyjson.Ordered, 0, len(barModuleNames))
	for _, name := range barModuleNames {
		shown := !defaultHidden[name]
		if override, set := overrides[name]; set {
			shown = override
		}
		if !barModulesEnabled[name] {
			shown = false
		}
		resolved = append(resolved, pyjson.Pair{Key: name, Value: boolText(shown)})
	}
	return resolved
}

// ToggleBarModuleOverride flips one module and returns the new overrides. An
// override equal to the default is dropped, so the file only ever records
// choices that differ from what the host would do on its own.
func ToggleBarModuleOverride(defaultHidden, overrides map[string]bool, name string) (map[string]bool, error) {
	if !isBarModule(name) {
		return nil, ErrUnknownBarModule
	}
	shown := !defaultHidden[name]
	if override, set := overrides[name]; set {
		shown = override
	}
	next := map[string]bool{}
	for key, value := range overrides {
		next[key] = value
	}
	if !shown == !defaultHidden[name] {
		delete(next, name)
	} else {
		next[name] = !shown
	}
	return next, nil
}

type barModulesFile struct {
	Version int             `json:"version"`
	Shown   map[string]bool `json:"shown"`
}

// ParseBarModuleOverrides decodes the overrides file, dropping names the bar no
// longer has.
func ParseBarModuleOverrides(text string) (map[string]bool, bool) {
	var file barModulesFile
	if err := json.Unmarshal([]byte(text), &file); err != nil || file.Version != barModulesVersion {
		return nil, false
	}
	overrides := map[string]bool{}
	for name, shown := range file.Shown {
		if isBarModule(name) {
			overrides[name] = shown
		}
	}
	return overrides, true
}

// EncodeBarModuleOverrides renders the overrides file.
func EncodeBarModuleOverrides(overrides map[string]bool) (string, error) {
	if overrides == nil {
		overrides = map[string]bool{}
	}
	encoded, err := json.MarshalIndent(barModulesFile{Version: barModulesVersion, Shown: overrides}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

var (
	barModulesLock      sync.Mutex
	barModulesDefaults  = DefaultHiddenBarModules(os.Getenv("EWW_BAR_DEFAULT_HIDDEN"))
	barModulesOverrides = map[string]bool{}
)

// LoadBarModules seeds the overrides from path; a missing or unreadable file
// leaves the host defaults in force.
func LoadBarModules(path string) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if overrides, ok := ParseBarModuleOverrides(string(data)); ok {
		barModulesLock.Lock()
		barModulesOverrides = overrides
		barModulesLock.Unlock()
	}
}

// BarModulesState is the current resolved visibility.
func BarModulesState() pyjson.Ordered {
	barModulesLock.Lock()
	defer barModulesLock.Unlock()
	return ResolveBarModules(barModulesDefaults, barModulesOverrides)
}

// ToggleBarModule flips one module and writes the overrides to path.
func ToggleBarModule(path, name string) (pyjson.Ordered, error) {
	barModulesLock.Lock()
	defer barModulesLock.Unlock()
	next, err := ToggleBarModuleOverride(barModulesDefaults, barModulesOverrides, name)
	if err != nil {
		return nil, err
	}
	barModulesOverrides = next
	return ResolveBarModules(barModulesDefaults, next), saveBarModules(path, next)
}

// ResetBarModules drops every override and writes the empty set to path.
func ResetBarModules(path string) (pyjson.Ordered, error) {
	barModulesLock.Lock()
	defer barModulesLock.Unlock()
	barModulesOverrides = map[string]bool{}
	return ResolveBarModules(barModulesDefaults, barModulesOverrides), saveBarModules(path, barModulesOverrides)
}

func saveBarModules(path string, overrides map[string]bool) error {
	if path == "" {
		return nil
	}
	text, err := EncodeBarModuleOverrides(overrides)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, text)
}
