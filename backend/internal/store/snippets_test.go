package store

import "testing"

func TestSaveSnippetDataCleansAndDedupes(t *testing.T) {
	st := NewTestStore()
	snips, vars := st.SaveSnippetData(
		[]Snippet{
			{Key: " intro ", Name: " 介绍 ", Content: "hello {{product}}"},
			{Key: "", Name: "blank", Content: "dropped"},
			{Key: "intro", Name: "override", Content: "world"},
		},
		map[string]string{" product ": " Modex ", "": "dropped"},
	)

	if len(snips) != 1 {
		t.Fatalf("snippets = %d, want 1 (blank dropped, dup merged)", len(snips))
	}
	if snips[0].Key != "intro" || snips[0].Content != "world" {
		t.Errorf("snippet not trimmed/overridden: %+v", snips[0])
	}
	if len(vars) != 1 || vars["product"] != "Modex" {
		t.Errorf("variables not cleaned: %+v", vars)
	}

	// Round-trips through the store.
	gotSnips, gotVars := st.SnippetData()
	if len(gotSnips) != 1 || gotVars["product"] != "Modex" {
		t.Errorf("SnippetData round-trip mismatch: %+v %+v", gotSnips, gotVars)
	}
}
