package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

func TestAgentCRUDReturnsVersionedDefinition(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	router := NewRouter(service.New(st, "tester"))
	body := []byte(`{"id":"research-agent","displayName":"Research Agent","definition":{"entryPoint":"root_agent.yaml","files":{"root_agent.yaml":"name: research-agent\n"}}}`)
	create := httptest.NewRequest(http.MethodPost, "/v3/aihub/agents", bytes.NewReader(body))
	create.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, create)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /v3/aihub/agents status = %d, body = %s", w.Code, w.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/v3/aihub/agents/research-agent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, get)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v3/aihub/agents/:agentId status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"latestVersion":"1.0.0"`) {
		t.Fatalf("GET /v3/aihub/agents/:agentId body = %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"entryPoint":"root_agent.yaml"`) {
		t.Fatalf("GET /v3/aihub/agents/:agentId did not return the immutable definition: %s", w.Body.String())
	}
}
