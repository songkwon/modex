package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (p *PostgresRepository) updateSettings(mutator func(*Settings) error) (Settings, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Settings{}, err
	}
	defer tx.Rollback(ctx)
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT value_json FROM platform_settings WHERE key='main' FOR UPDATE`).Scan(&raw)
	settings := Settings{}
	if err == nil {
		if err = json.Unmarshal(raw, &settings); err != nil {
			return Settings{}, err
		}
	}
	if err != nil {
		_, err = tx.Exec(ctx, `INSERT INTO platform_settings(key,value_json) VALUES('main','{}'::jsonb) ON CONFLICT(key) DO NOTHING`)
		if err != nil {
			return Settings{}, err
		}
	}
	if err = mutator(&settings); err != nil {
		return Settings{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO platform_settings(key,value_json) VALUES('main',$1::jsonb) ON CONFLICT(key) DO UPDATE SET value_json=EXCLUDED.value_json`, mustJSON(settings)); err != nil {
		return Settings{}, err
	}
	return settings, tx.Commit(ctx)
}

func (p *PostgresRepository) SaveAISettings(ai AISettings) Settings {
	settings, _ := p.updateSettings(func(current *Settings) error {
		if strings.TrimSpace(ai.AskAPIKey) == "" {
			ai.AskAPIKey = current.AI.AskAPIKey
		}
		if strings.TrimSpace(ai.EmbeddingAPIKey) == "" {
			ai.EmbeddingAPIKey = current.AI.EmbeddingAPIKey
		}
		if strings.TrimSpace(ai.RerankAPIKey) == "" {
			ai.RerankAPIKey = current.AI.RerankAPIKey
		}
		ai.UpdatedAt = time.Now().UTC()
		current.AI = ai
		return nil
	})
	return settings
}

func (p *PostgresRepository) PluginStates() []PluginState {
	settings := p.Settings()
	return mergePlugins(settings.Plugins, settings.UploadedPlugins)
}
func (p *PostgresRepository) PluginEffective() map[string]PluginSetting {
	states := p.PluginStates()
	result := make(map[string]PluginSetting, len(states))
	for _, state := range states {
		result[state.Key] = PluginSetting{Enabled: state.Enabled, Config: state.Config}
	}
	return result
}

func cleanPluginSettings(overrides map[string]PluginSetting, uploaded []UploadedPlugin) map[string]PluginSetting {
	allowed := map[string]map[string]bool{}
	for _, def := range pluginCatalog {
		fields := map[string]bool{}
		for _, field := range def.Fields {
			fields[field.Key] = true
		}
		allowed[def.Key] = fields
	}
	for _, plugin := range uploaded {
		if _, ok := allowed[plugin.Key]; !ok {
			allowed[plugin.Key] = map[string]bool{}
		}
	}
	clean := map[string]PluginSetting{}
	for key, override := range overrides {
		fields, known := allowed[key]
		if !known {
			continue
		}
		config := map[string]string{}
		for name, value := range override.Config {
			if fields[name] && strings.TrimSpace(value) != "" {
				config[name] = strings.TrimSpace(value)
			}
		}
		clean[key] = PluginSetting{Enabled: override.Enabled, Config: config}
	}
	return clean
}

func (p *PostgresRepository) SavePluginSettings(overrides map[string]PluginSetting) []PluginState {
	settings, _ := p.updateSettings(func(current *Settings) error {
		current.Plugins = cleanPluginSettings(overrides, current.UploadedPlugins)
		return nil
	})
	return mergePlugins(settings.Plugins, settings.UploadedPlugins)
}
func (p *PostgresRepository) UploadedPlugins() []UploadedPlugin {
	return append([]UploadedPlugin(nil), p.Settings().UploadedPlugins...)
}
func (p *PostgresRepository) EnabledUploadedPlugins() []UploadedPlugin {
	settings := p.Settings()
	result := []UploadedPlugin{}
	for _, plugin := range settings.UploadedPlugins {
		if override, ok := settings.Plugins[plugin.Key]; ok && override.Enabled {
			result = append(result, plugin)
		}
	}
	return result
}

func validateUploadedPlugin(plugin UploadedPlugin) (UploadedPlugin, error) {
	plugin.Key = strings.TrimSpace(plugin.Key)
	plugin.Name = strings.TrimSpace(plugin.Name)
	plugin.Tag = strings.TrimSpace(plugin.Tag)
	plugin.Lang = strings.TrimSpace(plugin.Lang)
	plugin.Category = strings.TrimSpace(plugin.Category)
	if plugin.Category == "" {
		plugin.Category = "custom"
	}
	plugin.Format = "jsx"
	if !keyRe.MatchString(plugin.Key) {
		return UploadedPlugin{}, fmt.Errorf("key 需为小写字母/数字/连字符（如 my-plugin）")
	}
	for _, definition := range pluginCatalog {
		if definition.Key == plugin.Key {
			return UploadedPlugin{}, fmt.Errorf("key 与内置插件冲突：%s", plugin.Key)
		}
	}
	if plugin.Name == "" {
		return UploadedPlugin{}, fmt.Errorf("name 不能为空")
	}
	if strings.TrimSpace(plugin.Code) == "" {
		return UploadedPlugin{}, fmt.Errorf("code 不能为空")
	}
	switch plugin.Kind {
	case "component":
		if !tagRe.MatchString(plugin.Tag) {
			return UploadedPlugin{}, fmt.Errorf("component 插件需要大写开头的 tag（如 Figma）")
		}
		plugin.Lang = ""
	case "fence":
		if !lngRe.MatchString(plugin.Lang) {
			return UploadedPlugin{}, fmt.Errorf("fence 插件需要小写的 lang（如 figma）")
		}
		plugin.Tag = ""
	default:
		return UploadedPlugin{}, fmt.Errorf("kind 必须是 component 或 fence")
	}
	return plugin, nil
}
func (p *PostgresRepository) SaveUploadedPlugin(plugin UploadedPlugin) (UploadedPlugin, error) {
	validated, err := validateUploadedPlugin(plugin)
	if err != nil {
		return UploadedPlugin{}, err
	}
	settings, err := p.updateSettings(func(current *Settings) error {
		validated.UpdatedAt = time.Now().UTC()
		for index, existing := range current.UploadedPlugins {
			if existing.Key == validated.Key {
				validated.Version = existing.Version + 1
				current.UploadedPlugins[index] = validated
				return nil
			}
		}
		validated.Version = 1
		current.UploadedPlugins = append(current.UploadedPlugins, validated)
		return nil
	})
	if err != nil {
		return UploadedPlugin{}, err
	}
	for _, saved := range settings.UploadedPlugins {
		if saved.Key == validated.Key {
			return saved, nil
		}
	}
	return UploadedPlugin{}, ErrNotFound
}
func (p *PostgresRepository) DeleteUploadedPlugin(key string) bool {
	deleted := false
	_, err := p.updateSettings(func(current *Settings) error {
		for index, plugin := range current.UploadedPlugins {
			if plugin.Key == key {
				current.UploadedPlugins = append(current.UploadedPlugins[:index], current.UploadedPlugins[index+1:]...)
				delete(current.Plugins, key)
				deleted = true
				break
			}
		}
		return nil
	})
	return err == nil && deleted
}

func cleanSnippetData(snippets []Snippet, variables map[string]string) ([]Snippet, map[string]string) {
	cleanSnippets := make([]Snippet, 0, len(snippets))
	seen := map[string]int{}
	for _, snippet := range snippets {
		key := strings.TrimSpace(snippet.Key)
		if key == "" {
			continue
		}
		entry := Snippet{Key: key, Name: strings.TrimSpace(snippet.Name), Content: snippet.Content}
		if index, ok := seen[key]; ok {
			cleanSnippets[index] = entry
		} else {
			seen[key] = len(cleanSnippets)
			cleanSnippets = append(cleanSnippets, entry)
		}
	}
	cleanVariables := map[string]string{}
	for key, value := range variables {
		key = strings.TrimSpace(key)
		if key != "" {
			cleanVariables[key] = strings.TrimSpace(value)
		}
	}
	return cleanSnippets, cleanVariables
}
func (p *PostgresRepository) SnippetData() ([]Snippet, map[string]string) {
	settings := p.Settings()
	snippets := append([]Snippet(nil), settings.Snippets...)
	variables := map[string]string{}
	for key, value := range settings.Variables {
		variables[key] = value
	}
	return snippets, variables
}
func (p *PostgresRepository) SaveSnippetData(snippets []Snippet, variables map[string]string) ([]Snippet, map[string]string) {
	snippets, variables = cleanSnippetData(snippets, variables)
	_, _ = p.updateSettings(func(current *Settings) error { current.Snippets = snippets; current.Variables = variables; return nil })
	return snippets, variables
}
