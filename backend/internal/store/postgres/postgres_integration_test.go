package postgresstore

import (
	"os"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func TestPostgresSandboxProfilePolicyRoundTrip(t *testing.T) {
	dsn := os.Getenv("HUB_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HUB_POSTGRES_DSN is not set")
	}
	st, err := New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	profile := &model.SandboxProfile{NamespaceID: model.DefaultNamespace, ID: "it-default-python-offline", Driver: "agent-sandbox", TemplateRef: "aisphere-agent-session", Status: model.MetaStatusEnable}
	if err := st.SaveSandboxProfile(profile); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoadSandboxProfile(model.DefaultNamespace, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.TemplateRef != profile.TemplateRef {
		t.Fatalf("unexpected profile: %#v", got)
	}
	policy := &model.SandboxPolicy{NamespaceID: model.DefaultNamespace, ID: "it-org-policy", TargetType: "org", TargetID: "org-it", AllowedProfiles: []string{profile.ID}}
	if err := st.SaveSandboxPolicy(policy); err != nil {
		t.Fatal(err)
	}
	loaded, err := st.LoadSandboxPolicy(model.DefaultNamespace, policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || len(loaded.AllowedProfiles) != 1 {
		t.Fatalf("unexpected policy: %#v", loaded)
	}
}
