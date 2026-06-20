package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"modex/backend/internal/store"
)

func TestSkillDiscoveryIndexMatchesTarballDigest(t *testing.T) {
	srv := New(store.NewTestStore())

	idxReq := httptest.NewRequest(http.MethodGet, "/.well-known/agent-skills/index.json", nil)
	idxRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(idxRec, idxReq)
	if idxRec.Code != http.StatusOK {
		t.Fatalf("index status = %d, body = %s", idxRec.Code, idxRec.Body.String())
	}

	var idx struct {
		Skills []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			URL    string `json:"url"`
			Digest string `json:"digest"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(idxRec.Body.Bytes(), &idx); err != nil {
		t.Fatalf("decode index: %v", err)
	}
	if len(idx.Skills) != 1 || idx.Skills[0].Name != "modex" || idx.Skills[0].Type != "archive" {
		t.Fatalf("unexpected skills index: %+v", idx.Skills)
	}

	tgzReq := httptest.NewRequest(http.MethodGet, idx.Skills[0].URL, nil)
	tgzRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(tgzRec, tgzReq)
	if tgzRec.Code != http.StatusOK {
		t.Fatalf("tarball status = %d, body = %s", tgzRec.Code, tgzRec.Body.String())
	}
	sum := sha256.Sum256(tgzRec.Body.Bytes())
	if got, want := fmt.Sprintf("sha256:%x", sum[:]), idx.Skills[0].Digest; got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
}
