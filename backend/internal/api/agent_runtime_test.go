package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

func TestResolveAgentRuntimeSnapshotPinsAgentAndSkills(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	svc := service.New(st, "tester")
	skill := &model.SkillRecord{
		NamespaceID: model.DefaultNamespace,
		Name:        "research-skill",
		Status:      model.MetaStatusEnable,
		Scope:       model.ScopePrivate,
		Labels:      map[string]string{model.LabelLatest: "1.0.0"},
		Versions: map[string]*model.VersionRecord{
			"1.0.0": {Version: "1.0.0", Status: model.VersionStatusOnline, Skill: model.Skill{SkillBase: model.SkillBase{Name: "research-skill"}, SkillMD: "---\nname: research-skill\n---\n"}},
		},
	}
	if err := st.Save(skill); err != nil {
		t.Fatalf("Save(skill) error = %v", err)
	}
	_, err = svc.CreateAgent(model.DefaultNamespace, model.AgentUpsertRequest{
		ID: "research-agent",
		Definition: model.AgentDefinition{
			EntryPoint: "root_agent.yaml",
			Files:      map[string]string{"root_agent.yaml": "name: research-agent\n"},
			Skills:     []model.AgentSkillRef{{Name: "research-skill", Required: true}},
		},
	}, "admin/test1")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	router := NewRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/v3/aihub/runtime/agents/research-agent/resolve", bytes.NewBufferString(`{"runtimeId":"adk-local","sessionId":"s-1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST runtime resolve status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"agentVersion":"1.0.0"`) {
		t.Fatalf("runtime snapshot does not pin agent version: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"name":"research-skill"`) {
		t.Fatalf("runtime snapshot does not include declared skill: %s", w.Body.String())
	}
}
