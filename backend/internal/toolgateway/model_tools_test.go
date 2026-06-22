package toolgateway

import (
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func TestBuildModelToolSpecs(t *testing.T) {
	items := []model.SandboxToolSchema{{Name: "workspace.read", Description: "read", InputSchema: map[string]interface{}{"type": "object"}}}
	tools := BuildModelToolSpecs(items)
	if len(tools) != 1 {
		t.Fatalf("expected one tool")
	}
	fn := tools[0]["function"].(map[string]interface{})
	if fn["name"] != "workspace__read" {
		t.Fatalf("unexpected model tool name: %v", fn["name"])
	}
	meta := tools[0]["x_aisphere"].(map[string]interface{})
	if meta["toolId"] != "workspace.read" {
		t.Fatalf("tool id not preserved: %#v", meta)
	}
}
