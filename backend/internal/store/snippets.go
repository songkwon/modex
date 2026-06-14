package store

import "strings"

// Snippet is a reusable Markdown partial referenced from docs as
// <Snippet name="key"/>. Together with Variables it powers Mintlify-style
// reusable content. Both live in Settings, so they snapshot automatically.
type Snippet struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// SnippetData returns the snippet library and the global variable map.
func (s *Store) SnippetData() ([]Snippet, map[string]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snips := make([]Snippet, len(s.settings.Snippets))
	copy(snips, s.settings.Snippets)
	vars := make(map[string]string, len(s.settings.Variables))
	for k, v := range s.settings.Variables {
		vars[k] = v
	}
	return snips, vars
}

// SaveSnippetData replaces the snippet library and variables. Snippets with a
// blank key are dropped; keys are trimmed and de-duplicated (last wins). Blank
// variable keys are dropped and keys/values trimmed.
func (s *Store) SaveSnippetData(snips []Snippet, vars map[string]string) ([]Snippet, map[string]string) {
	cleanSnips := make([]Snippet, 0, len(snips))
	seen := map[string]int{}
	for _, sn := range snips {
		key := strings.TrimSpace(sn.Key)
		if key == "" {
			continue
		}
		entry := Snippet{Key: key, Name: strings.TrimSpace(sn.Name), Content: sn.Content}
		if idx, ok := seen[key]; ok {
			cleanSnips[idx] = entry
			continue
		}
		seen[key] = len(cleanSnips)
		cleanSnips = append(cleanSnips, entry)
	}
	cleanVars := map[string]string{}
	for k, v := range vars {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		cleanVars[key] = strings.TrimSpace(v)
	}
	s.mu.Lock()
	s.settings.Snippets = cleanSnips
	s.settings.Variables = cleanVars
	s.mu.Unlock()
	return cleanSnips, cleanVars
}
