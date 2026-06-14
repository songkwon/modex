package store

import "testing"

func TestSaveUploadedPluginValidation(t *testing.T) {
	st := New()
	cases := []struct {
		name string
		p    UploadedPlugin
		ok   bool
	}{
		{"good component", UploadedPlugin{Key: "demo", Name: "Demo", Kind: "component", Tag: "Demo", Code: "x"}, true},
		{"good fence", UploadedPlugin{Key: "echart", Name: "EChart", Kind: "fence", Lang: "echart", Code: "x"}, true},
		{"bad key", UploadedPlugin{Key: "Demo!", Name: "Demo", Kind: "component", Tag: "Demo", Code: "x"}, false},
		{"clashes builtin", UploadedPlugin{Key: "kroki", Name: "K", Kind: "component", Tag: "K", Code: "x"}, false},
		{"component no tag", UploadedPlugin{Key: "d2", Name: "D", Kind: "component", Code: "x"}, false},
		{"fence no lang", UploadedPlugin{Key: "d3", Name: "D", Kind: "fence", Code: "x"}, false},
		{"empty code", UploadedPlugin{Key: "d4", Name: "D", Kind: "component", Tag: "D", Code: " "}, false},
		{"bad kind", UploadedPlugin{Key: "d5", Name: "D", Kind: "widget", Code: "x"}, false},
	}
	for _, c := range cases {
		_, err := st.SaveUploadedPlugin(c.p)
		if (err == nil) != c.ok {
			t.Errorf("%s: ok=%v err=%v", c.name, c.ok, err)
		}
	}
}

func TestUploadedPluginEnableFlow(t *testing.T) {
	st := New()
	if _, err := st.SaveUploadedPlugin(UploadedPlugin{Key: "demo", Name: "Demo", Kind: "component", Tag: "Demo", Code: "code"}); err != nil {
		t.Fatal(err)
	}
	// Appears in admin states, disabled by default, flagged uploaded.
	states := st.PluginStates()
	demo, ok := findState(states, "demo")
	if !ok || !demo.Uploaded || demo.Enabled || demo.Kind != "component" || demo.Tag != "Demo" {
		t.Fatalf("unexpected state: %+v ok=%v", demo, ok)
	}
	// Not visible to the renderer while disabled.
	if len(st.EnabledUploadedPlugins()) != 0 {
		t.Error("disabled plugin should not be enabled-listed")
	}
	// Enable via the shared override path.
	st.SavePluginSettings(map[string]PluginSetting{"demo": {Enabled: true}})
	en := st.EnabledUploadedPlugins()
	if len(en) != 1 || en[0].Key != "demo" || en[0].Code != "code" {
		t.Fatalf("enabled list wrong: %+v", en)
	}
	// Re-import bumps version; delete also clears the override.
	saved, _ := st.SaveUploadedPlugin(UploadedPlugin{Key: "demo", Name: "Demo2", Kind: "component", Tag: "Demo", Code: "code2"})
	if saved.Version != 2 {
		t.Errorf("version = %d, want 2", saved.Version)
	}
	if !st.DeleteUploadedPlugin("demo") || len(st.UploadedPlugins()) != 0 {
		t.Error("delete failed")
	}
}
