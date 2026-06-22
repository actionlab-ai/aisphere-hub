package service

import (
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

func TestCreateAgentAssignsFirstImmutableVersion(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	svc := New(st, "tester")

	created, err := svc.CreateAgent(model.DefaultNamespace, model.AgentUpsertRequest{
		ID:          "research-agent",
		DisplayName: "Research Agent",
		Definition: model.AgentDefinition{
			EntryPoint: "root_agent.yaml",
			Files: map[string]string{
				"root_agent.yaml": "name: research-agent\n",
			},
		},
	}, "admin/test1")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	if created.LatestVersion != "1.0.0" {
		t.Fatalf("LatestVersion = %q, want 1.0.0", created.LatestVersion)
	}
	version := created.Versions[created.LatestVersion]
	if version == nil {
		t.Fatalf("version %q is missing", created.LatestVersion)
	}
	if version.Definition.EntryPoint != "root_agent.yaml" {
		t.Fatalf("EntryPoint = %q, want root_agent.yaml", version.Definition.EntryPoint)
	}
}
