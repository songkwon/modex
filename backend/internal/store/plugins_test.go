package store

import "testing"

func findState(states []PluginState, key string) (PluginState, bool) {
	for _, s := range states {
		if s.Key == key {
			return s, true
		}
	}
	return PluginState{}, false
}

func TestPluginStatesDefaults(t *testing.T) {
	states := NewTestStore().PluginStates()
	if len(states) != len(pluginCatalog) {
		t.Fatalf("states = %d, want %d", len(states), len(pluginCatalog))
	}
	kroki, ok := findState(states, "kroki")
	if !ok {
		t.Fatal("kroki plugin missing from catalog")
	}
	if !kroki.Enabled {
		t.Error("kroki should be enabled by default")
	}
	if kroki.Category != "diagram" || len(kroki.Fields) == 0 || kroki.Fields[0].Key != "base_url" {
		t.Errorf("kroki def unexpected: %+v", kroki.PluginDef)
	}
}

func TestSavePluginSettingsOverrideAndFilter(t *testing.T) {
	st := NewTestStore()
	st.SavePluginSettings(map[string]PluginSetting{
		"kroki":   {Enabled: false, Config: map[string]string{"base_url": "  http://kroki.internal:8000  ", "bogus": "x"}},
		"unknown": {Enabled: true},
	})
	states := st.PluginStates()

	kroki, _ := findState(states, "kroki")
	if kroki.Enabled {
		t.Error("kroki should be disabled after override")
	}
	if got := kroki.Config["base_url"]; got != "http://kroki.internal:8000" {
		t.Errorf("base_url = %q, want trimmed value", got)
	}
	if _, ok := kroki.Config["bogus"]; ok {
		t.Error("unknown config field should be dropped")
	}
	if _, ok := findState(states, "unknown"); ok {
		t.Error("unknown plugin key should not appear")
	}
	// A plugin we did not touch keeps its default-enabled state.
	if math, _ := findState(states, "math"); !math.Enabled {
		t.Error("untouched math plugin should stay enabled")
	}
}

func TestPluginEffective(t *testing.T) {
	st := NewTestStore()
	st.SavePluginSettings(map[string]PluginSetting{"math": {Enabled: false}})
	eff := st.PluginEffective()
	if eff["math"].Enabled {
		t.Error("math should be disabled in effective config")
	}
	if !eff["kroki"].Enabled {
		t.Error("kroki should remain enabled in effective config")
	}
}
