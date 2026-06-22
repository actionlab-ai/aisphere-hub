package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

func TestCreateDraftCanonicalAcceptsDirectJSONDraft(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	router := NewRouter(service.New(st, "tester"))

	body := map[string]interface{}{
		"name":        "frontend-draft",
		"displayName": "Frontend Draft",
		"description": "Created from the frontend draft dialog.",
		"version":     "1.2.3",
		"keywords":    []string{"frontend", "draft"},
		"bizTags":     []string{"console"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v3/aihub/skills/draft", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /v3/aihub/skills/draft status = %d, body = %s", w.Code, w.Body.String())
	}
	var result model.Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	if result.Data != "1.2.3" {
		t.Fatalf("created version = %v, want 1.2.3", result.Data)
	}

	rec, err := st.Load(model.DefaultNamespace, "frontend-draft")
	if err != nil {
		t.Fatalf("Load(frontend-draft) error = %v", err)
	}
	if rec == nil {
		t.Fatal("Load(frontend-draft) = nil")
	}
	if rec.Description != "Created from the frontend draft dialog." {
		t.Fatalf("description = %q", rec.Description)
	}
	if rec.EditingVersion != "1.2.3" {
		t.Fatalf("editing version = %q, want 1.2.3", rec.EditingVersion)
	}
	version := rec.Versions["1.2.3"]
	if version == nil {
		t.Fatal("version 1.2.3 missing")
	}
	if version.Skill.SkillMD == "" {
		t.Fatal("generated skill markdown is empty")
	}
}

func TestPlatformHealthEndpointsArePublicAndHealthy(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	router := NewRouter(service.New(st, "tester"))
	for _, path := range []string{"/healthz", "/livez", "/readyz"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", path, w.Code, w.Body.String())
		}
	}
}
