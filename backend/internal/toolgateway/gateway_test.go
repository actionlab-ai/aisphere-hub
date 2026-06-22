package toolgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func TestHTTPGatewayListAndCall(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/tools":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": []map[string]interface{}{{"name": "workspace.list", "description": "list"}}})
		case "/v1/tools/call":
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "tool": req["tool"], "result": map[string]interface{}{"done": true}, "durationMillis": 12})
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	g := NewHTTPGateway(nil)
	tools, err := g.ListTools(context.Background(), s.URL)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "workspace.list" {
		t.Fatalf("unexpected tools: %#v", tools.Tools)
	}
	out, err := g.Call(context.Background(), s.URL, model.SandboxToolCallRequest{Tool: "workspace.list", Input: map[string]interface{}{"path": "."}})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if !out.OK || out.Tool != "workspace.list" || out.Result["done"] != true {
		t.Fatalf("unexpected call result: %#v", out)
	}
}

func TestHTTPGatewayStructuredFailure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": map[string]interface{}{"code": "ValueError", "message": "bad input"}, "durationMillis": 3})
	}))
	defer s.Close()

	out, err := NewHTTPGateway(nil).Call(context.Background(), s.URL, model.SandboxToolCallRequest{Tool: "workspace.read"})
	if err != nil {
		t.Fatalf("Call should return structured failure without transport error: %v", err)
	}
	if out.OK || out.Error == nil || out.Error.Code != "ValueError" {
		t.Fatalf("unexpected failure result: %#v", out)
	}
}
