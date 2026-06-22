package store

import (
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func TestToolStoreAndFailures(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := &model.ToolRecord{
		NamespaceID:   model.DefaultNamespace,
		App:           model.DefaultApp,
		ID:            "github.issue.create",
		Status:        model.MetaStatusEnable,
		Scope:         model.ScopePrivate,
		Labels:        map[string]string{},
		LatestVersion: "1.0.0",
		Versions: map[string]*model.ToolVersionRecord{
			"1.0.0": {Version: "1.0.0", Revision: "rev1", Definition: model.ToolDefinition{Runtime: model.ToolRuntimeDefinition{Type: "mcp", Server: "github", Name: "create_issue"}}},
		},
	}
	if err := st.SaveTool(rec); err != nil {
		t.Fatal(err)
	}
	loaded, err := st.LoadTool(model.DefaultNamespace, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.ID != rec.ID || loaded.Versions["1.0.0"] == nil {
		t.Fatalf("unexpected loaded tool: %#v", loaded)
	}
	items, err := st.ListTools(model.DefaultNamespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != rec.ID {
		t.Fatalf("unexpected list: %#v", items)
	}
	failure := &model.ToolFailureRecord{ID: "tf_1", ToolID: rec.ID, AgentID: "demo-agent", ErrorMessage: "boom", CreateTime: model.NowMillis()}
	if err := st.AppendToolFailure(failure); err != nil {
		t.Fatal(err)
	}
	failures, total, err := st.ListToolFailures(model.ToolFailureQuery{ToolID: rec.ID})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(failures) != 1 || failures[0].ErrorMessage != "boom" {
		t.Fatalf("unexpected failures: total=%d failures=%#v", total, failures)
	}
}
