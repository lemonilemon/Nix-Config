package collect

import (
	"os"
	"path/filepath"
	"testing"

	"ewwbar/internal/pyjson"
)

func resolvedValue(t *testing.T, modules pyjson.Ordered, name string) string {
	t.Helper()
	for _, pair := range modules {
		if pair.Key == name {
			return pair.Value.(string)
		}
	}
	t.Fatalf("%s missing from %v", name, modules)
	return ""
}

func TestResolveBarModulesListsEveryModuleInBarOrder(t *testing.T) {
	modules := ResolveBarModules(nil, nil)
	if len(modules) != len(barModuleNames) {
		t.Fatalf("got %d modules, want %d", len(modules), len(barModuleNames))
	}
	for i, pair := range modules {
		if pair.Key != barModuleNames[i] || pair.Value != "true" {
			t.Errorf("position %d: %s=%v, want %s=true", i, pair.Key, pair.Value, barModuleNames[i])
		}
	}
}

// The laptop hides storage by default; an override must win over that in both
// directions, and a module nobody mentioned must follow the host default.
func TestResolveBarModulesAppliesOverridesOverDefaults(t *testing.T) {
	defaults := DefaultHiddenBarModules("storage, cpu")
	modules := ResolveBarModules(defaults, map[string]bool{"storage": true, "memory": false})
	for name, want := range map[string]string{
		"storage": "true", "cpu": "false", "memory": "false", "network": "true",
	} {
		if got := resolvedValue(t, modules, name); got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
}

func TestDefaultHiddenBarModulesIgnoresUnknownNamesAndBlanks(t *testing.T) {
	hidden := DefaultHiddenBarModules("storage,,nonsense, ")
	if len(hidden) != 1 || !hidden["storage"] {
		t.Errorf("got %v, want only storage", hidden)
	}
}

// Toggling back to the host default must delete the override rather than store
// it, or a later change of default would be shadowed by a stale entry.
func TestToggleBarModuleOverrideDropsOverridesThatMatchTheDefault(t *testing.T) {
	defaults := DefaultHiddenBarModules("storage")

	shown, err := ToggleBarModuleOverride(defaults, map[string]bool{}, "storage")
	if err != nil || len(shown) != 1 || shown["storage"] != true {
		t.Fatalf("first toggle: %v %v, want storage=true", shown, err)
	}
	back, err := ToggleBarModuleOverride(defaults, shown, "storage")
	if err != nil || len(back) != 0 {
		t.Fatalf("second toggle: %v %v, want no overrides", back, err)
	}
	if len(shown) != 1 {
		t.Errorf("toggle mutated its input: %v", shown)
	}
}

func TestToggleBarModuleOverrideRejectsUnknownModules(t *testing.T) {
	if _, err := ToggleBarModuleOverride(nil, nil, "workspaces"); err != ErrUnknownBarModule {
		t.Errorf("got %v, want ErrUnknownBarModule", err)
	}
}

func TestBarModuleOverridesRoundTrip(t *testing.T) {
	text, err := EncodeBarModuleOverrides(map[string]bool{"storage": true, "cpu": false})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ParseBarModuleOverrides(text)
	if !ok || len(got) != 2 || got["storage"] != true || got["cpu"] != false {
		t.Errorf("round trip gave %v %v from %s", got, ok, text)
	}
}

func TestParseBarModuleOverridesRejectsOtherVersionsAndDropsRemovedModules(t *testing.T) {
	if _, ok := ParseBarModuleOverrides(`{"version":2,"shown":{}}`); ok {
		t.Error("accepted version 2")
	}
	if _, ok := ParseBarModuleOverrides("not json"); ok {
		t.Error("accepted garbage")
	}
	got, ok := ParseBarModuleOverrides(`{"version":1,"shown":{"storage":true,"gone":false}}`)
	if !ok || len(got) != 1 || !got["storage"] {
		t.Errorf("got %v %v, want only storage", got, ok)
	}
}

func TestToggleBarModulePersistsAcrossALoad(t *testing.T) {
	savedDefaults, savedOverrides := barModulesDefaults, barModulesOverrides
	defer func() { barModulesDefaults, barModulesOverrides = savedDefaults, savedOverrides }()
	barModulesDefaults = DefaultHiddenBarModules("storage")
	barModulesOverrides = map[string]bool{}

	path := filepath.Join(t.TempDir(), "eww-bar", "modules.json")
	if _, err := ToggleBarModule(path, "storage"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no file written: %v", err)
	}

	barModulesOverrides = map[string]bool{}
	LoadBarModules(path)
	if got := resolvedValue(t, BarModulesState(), "storage"); got != "true" {
		t.Errorf("storage = %s after reload, want true", got)
	}

	if _, err := ResetBarModules(path); err != nil {
		t.Fatal(err)
	}
	if got := resolvedValue(t, BarModulesState(), "storage"); got != "false" {
		t.Errorf("storage = %s after reset, want the host default false", got)
	}
}
