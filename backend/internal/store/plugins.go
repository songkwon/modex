package store

import "strings"

// PluginSetting is the persisted per-plugin override (enable flag + config
// values). It lives inside Settings, so it is snapshotted with everything else.
type PluginSetting struct {
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config,omitempty"`
}

// PluginField describes one configurable value of a plugin for the admin UI.
type PluginField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     string `json:"default,omitempty"`
}

// PluginDef is a static catalog entry for a built-in plugin. The catalog is the
// single source of truth; admins only toggle/configure these, they cannot add
// arbitrary third-party code.
type PluginDef struct {
	Key            string        `json:"key"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Category       string        `json:"category"`
	DefaultEnabled bool          `json:"default_enabled"`
	Fields         []PluginField `json:"fields,omitempty"`
}

// PluginState is a catalog entry merged with its saved override (admin view).
// Uploaded (imported) plugins carry extra fields so the admin UI can show their
// provenance, kind and source.
type PluginState struct {
	PluginDef
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
	Uploaded bool              `json:"uploaded,omitempty"`
	Kind     string            `json:"kind,omitempty"`
	Tag      string            `json:"tag,omitempty"`
	Lang     string            `json:"lang,omitempty"`
	Code     string            `json:"code,omitempty"`
}

// pluginCatalog is the curated set of built-in doc-engine plugins.
var pluginCatalog = []PluginDef{
	{Key: "kroki", Name: "Kroki 图表", Category: "diagram", DefaultEnabled: true,
		Description: "用 Kroki 渲染 PlantUML、Graphviz、C4、DITAA、D2 等图表代码块。",
		Fields:      []PluginField{{Key: "base_url", Label: "Kroki 服务地址", Placeholder: "https://kroki.io"}}},
	{Key: "mermaid", Name: "Mermaid 图表", Category: "diagram", DefaultEnabled: true,
		Description: "渲染 Mermaid 流程图、时序图、甘特图等图表。"},
	{Key: "math", Name: "数学公式", Category: "math", DefaultEnabled: true,
		Description: "用 KaTeX 渲染行内公式和块级 LaTeX 公式。"},
	{Key: "github_alerts", Name: "GitHub 提示块", Category: "content", DefaultEnabled: true,
		Description: "把 GitHub 风格的 NOTE、WARNING 等提示块转为提示卡片。"},
	{Key: "toc", Name: "自动目录", Category: "content", DefaultEnabled: true,
		Description: "把目录占位标记替换为页面内目录导航。"},
	{Key: "footnotes", Name: "脚注", Category: "content", DefaultEnabled: true,
		Description: "渲染 Markdown 脚注。"},
	{Key: "snippets", Name: "可复用片段", Category: "content", DefaultEnabled: true,
		Description: "支持变量插值与可复用内容片段。"},
	{Key: "openapi", Name: "OpenAPI / API 调试台", Category: "api", DefaultEnabled: true,
		Description: "渲染交互式 API 调试台与从 OpenAPI 规范生成的接口参考。",
		Fields:      []PluginField{{Key: "default_spec_url", Label: "默认 OpenAPI 规范地址", Placeholder: "https://example.com/openapi.json"}}},
}

// PluginCatalog returns the static built-in plugin catalog.
func PluginCatalog() []PluginDef { return pluginCatalog }

// mergePlugins overlays saved overrides onto the catalog defaults, then appends
// uploaded plugins (which default to disabled).
func mergePlugins(overrides map[string]PluginSetting, uploaded []UploadedPlugin) []PluginState {
	out := make([]PluginState, 0, len(pluginCatalog)+len(uploaded))
	for _, def := range pluginCatalog {
		st := PluginState{PluginDef: def, Enabled: def.DefaultEnabled, Config: map[string]string{}}
		for _, f := range def.Fields {
			if f.Default != "" {
				st.Config[f.Key] = f.Default
			}
		}
		if ov, ok := overrides[def.Key]; ok {
			st.Enabled = ov.Enabled
			for k, v := range ov.Config {
				st.Config[k] = v
			}
		}
		out = append(out, st)
	}
	for _, up := range uploaded {
		st := PluginState{
			PluginDef: PluginDef{Key: up.Key, Name: up.Name, Description: up.Description, Category: up.Category, DefaultEnabled: false},
			Enabled:   false,
			Config:    map[string]string{},
			Uploaded:  true,
			Kind:      up.Kind,
			Tag:       up.Tag,
			Lang:      up.Lang,
			Code:      up.Code,
		}
		if ov, ok := overrides[up.Key]; ok {
			st.Enabled = ov.Enabled
		}
		out = append(out, st)
	}
	return out
}

// PluginStates returns the catalog merged with saved overrides (admin view).
func (s *MemoryStore) PluginStates() []PluginState {
	s.mu.RLock()
	overrides := s.settings.Plugins
	uploaded := s.settings.UploadedPlugins
	s.mu.RUnlock()
	return mergePlugins(overrides, uploaded)
}

// SavePluginSettings persists enable/config overrides, ignoring unknown plugin
// keys and config fields not declared in the catalog.
func (s *MemoryStore) SavePluginSettings(overrides map[string]PluginSetting) []PluginState {
	allowed := map[string]map[string]bool{}
	for _, def := range pluginCatalog {
		fs := map[string]bool{}
		for _, f := range def.Fields {
			fs[f.Key] = true
		}
		allowed[def.Key] = fs
	}
	s.mu.RLock()
	uploaded := s.settings.UploadedPlugins
	for _, up := range uploaded {
		if _, ok := allowed[up.Key]; !ok {
			allowed[up.Key] = map[string]bool{} // uploaded plugins: enable flag only
		}
	}
	s.mu.RUnlock()

	clean := map[string]PluginSetting{}
	for key, ov := range overrides {
		fields, known := allowed[key]
		if !known {
			continue
		}
		cfg := map[string]string{}
		for k, v := range ov.Config {
			if fields[k] {
				if tv := strings.TrimSpace(v); tv != "" {
					cfg[k] = tv
				}
			}
		}
		clean[key] = PluginSetting{Enabled: ov.Enabled, Config: cfg}
	}
	s.mu.Lock()
	s.settings.Plugins = clean
	uploaded = s.settings.UploadedPlugins
	s.mu.Unlock()
	return mergePlugins(clean, uploaded)
}

// PluginEffective returns a slim enabled+config map for the public config API,
// consumed by the doc renderer to drive conditional plugins.
func (s *MemoryStore) PluginEffective() map[string]PluginSetting {
	states := s.PluginStates()
	out := make(map[string]PluginSetting, len(states))
	for _, st := range states {
		out[st.Key] = PluginSetting{Enabled: st.Enabled, Config: st.Config}
	}
	return out
}
