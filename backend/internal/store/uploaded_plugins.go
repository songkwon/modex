package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// UploadedPlugin is an admin-imported plugin whose JSX source is rendered inside
// a sandboxed iframe on the frontend. Kind "component" registers an MDX tag;
// "fence" handles a fenced code language. Enable/disable goes through the shared
// Plugins overrides (uploaded plugins default to disabled).
type UploadedPlugin struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Kind        string    `json:"kind"`           // "component" | "fence"
	Tag         string    `json:"tag,omitempty"`  // component tag, e.g. "Figma"
	Lang        string    `json:"lang,omitempty"` // fence language, e.g. "figma"
	Code        string    `json:"code"`           // JSX source defining a Plugin component
	Format      string    `json:"format"`         // "jsx"
	Version     int       `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	keyRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	tagRe = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	lngRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// UploadedPlugins returns a copy of the imported plugin list.
func (s *MemoryStore) UploadedPlugins() []UploadedPlugin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UploadedPlugin, len(s.settings.UploadedPlugins))
	copy(out, s.settings.UploadedPlugins)
	return out
}

// EnabledUploadedPlugins returns only imported plugins currently toggled on.
// Consumed by the renderer (uploaded plugins default to disabled).
func (s *MemoryStore) EnabledUploadedPlugins() []UploadedPlugin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []UploadedPlugin{}
	for _, up := range s.settings.UploadedPlugins {
		if ov, ok := s.settings.Plugins[up.Key]; ok && ov.Enabled {
			out = append(out, up)
		}
	}
	return out
}

// SaveUploadedPlugin validates and upserts an imported plugin (by key). It never
// changes the enabled flag — imports stay disabled until an admin turns them on.
func (s *MemoryStore) SaveUploadedPlugin(p UploadedPlugin) (UploadedPlugin, error) {
	p.Key = strings.TrimSpace(p.Key)
	p.Name = strings.TrimSpace(p.Name)
	p.Tag = strings.TrimSpace(p.Tag)
	p.Lang = strings.TrimSpace(p.Lang)
	p.Category = strings.TrimSpace(p.Category)
	if p.Category == "" {
		p.Category = "custom"
	}
	p.Format = "jsx"

	if !keyRe.MatchString(p.Key) {
		return UploadedPlugin{}, fmt.Errorf("key 需为小写字母/数字/连字符（如 my-plugin）")
	}
	for _, def := range pluginCatalog {
		if def.Key == p.Key {
			return UploadedPlugin{}, fmt.Errorf("key 与内置插件冲突：%s", p.Key)
		}
	}
	if p.Name == "" {
		return UploadedPlugin{}, fmt.Errorf("name 不能为空")
	}
	if strings.TrimSpace(p.Code) == "" {
		return UploadedPlugin{}, fmt.Errorf("code 不能为空")
	}
	switch p.Kind {
	case "component":
		if !tagRe.MatchString(p.Tag) {
			return UploadedPlugin{}, fmt.Errorf("component 插件需要大写开头的 tag（如 Figma）")
		}
		p.Lang = ""
	case "fence":
		if !lngRe.MatchString(p.Lang) {
			return UploadedPlugin{}, fmt.Errorf("fence 插件需要小写的 lang（如 figma）")
		}
		p.Tag = ""
	default:
		return UploadedPlugin{}, fmt.Errorf("kind 必须是 component 或 fence")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	p.UpdatedAt = time.Now().UTC()
	for i, ex := range s.settings.UploadedPlugins {
		if ex.Key == p.Key {
			p.Version = ex.Version + 1
			s.settings.UploadedPlugins[i] = p
			return p, nil
		}
	}
	p.Version = 1
	s.settings.UploadedPlugins = append(s.settings.UploadedPlugins, p)
	return p, nil
}

// DeleteUploadedPlugin removes an imported plugin and its enable override.
func (s *MemoryStore) DeleteUploadedPlugin(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, ex := range s.settings.UploadedPlugins {
		if ex.Key == key {
			s.settings.UploadedPlugins = append(s.settings.UploadedPlugins[:i], s.settings.UploadedPlugins[i+1:]...)
			delete(s.settings.Plugins, key)
			return true
		}
	}
	return false
}
