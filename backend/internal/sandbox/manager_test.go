package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func TestKubernetesManagerEnsureSandbox(t *testing.T) {
	var mu sync.Mutex
	objects := map[string]map[string]interface{}{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var obj map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&obj)
			name := nestedString(obj, "metadata", "name")
			if name == "" {
				name = strings.TrimPrefix(r.URL.Path, "/api/v1/namespaces/")
			}
			storeKey := r.URL.Path + "/" + name
			if strings.HasSuffix(r.URL.Path, "/pods") {
				obj["status"] = map[string]interface{}{"phase": "Running", "podIP": "10.1.2.3"}
			}
			objects[storeKey] = obj
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(obj)
		case http.MethodGet:
			if obj, ok := objects[r.URL.Path]; ok {
				_ = json.NewEncoder(w).Encode(obj)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","reason":"NotFound"}`))
		case http.MethodPut:
			var obj map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&obj)
			objects[r.URL.Path] = obj
			_ = json.NewEncoder(w).Encode(obj)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s", key)
		}
	}))
	defer srv.Close()

	mgr, err := NewKubernetesManager(Config{APIServer: srv.URL, Insecure: true, Namespace: "aisphere-sandbox", CreateNamespace: true, Image: "sandbox:test", ToolPort: 18081, BrowserPort: 9222, VNCOrWebPort: 6080, WorkspaceSize: "1Gi", DefaultCPU: "100m", DefaultMemory: "128Mi", MaxCPU: "500m", MaxMemory: "512Mi", NetworkPolicyEnabled: true, DefaultNetworkMode: model.SandboxNetworkModeOffline})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	st, err := mgr.Ensure(context.Background(), model.SandboxEnsureRequest{SessionID: "sess-001", OrgID: "org-a", ProjectID: "proj-a", AgentID: "agent-a", Network: model.SandboxNetworkPolicy{Mode: model.SandboxNetworkModeRestricted, EgressCIDRs: []string{"10.0.0.0/8"}}, Services: []model.RuntimeServiceManifest{{Kind: model.ServiceKindTool, Name: "project.files.write"}}})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if st.Phase != model.SandboxPhaseRunning {
		t.Fatalf("phase=%s", st.Phase)
	}
	if st.WorkspacePVC == "" || len(st.Endpoints) != 3 || st.NetworkMode != model.SandboxNetworkModeRestricted {
		t.Fatalf("unexpected status: %+v", st)
	}
	if _, ok := objects["/apis/networking.k8s.io/v1/namespaces/aisphere-sandbox/networkpolicies/aisb-sb-org-a-proj-a-sess-001"]; !ok {
		t.Fatalf("networkpolicy was not created; keys=%v", objects)
	}
}

func nestedString(obj map[string]interface{}, keys ...string) string {
	cur := interface{}(obj)
	for _, key := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return ""
		}
		cur = m[key]
	}
	s, _ := cur.(string)
	return s
}
