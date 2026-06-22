package toolgateway

import (
	"regexp"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

var unsafeToolNameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// ModelToolName converts platform tool ids such as "workspace.read" into a
// conservative function name accepted by most model APIs. The original platform
// tool id must be kept in metadata and used when dispatching the tool call.
func ModelToolName(toolID string) string {
	name := unsafeToolNameChars.ReplaceAllString(strings.TrimSpace(toolID), "__")
	name = strings.Trim(name, "_")
	if name == "" {
		return "tool"
	}
	if len(name) > 64 {
		return strings.Trim(name[:56], "_") + "__" + shortHash(name)
	}
	return name
}

// BuildModelToolSpecs returns the small, model-facing tool schema set. It must
// not include execution placement, PVC names, Pod DNS, secret references or
// Kubernetes policy. ToolGateway keeps those details in RuntimeServiceManifest.
func BuildModelToolSpecs(tools []model.SandboxToolSchema) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		if strings.TrimSpace(t.Name) == "" {
			continue
		}
		params := t.InputSchema
		if params == nil {
			params = map[string]interface{}{"type": "object"}
		}
		out = append(out, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        ModelToolName(t.Name),
				"description": strings.TrimSpace(t.Description),
				"parameters":  params,
			},
			"x_aisphere": map[string]interface{}{
				"toolId": t.Name,
			},
		})
	}
	return out
}

func shortHash(v string) string {
	h := uint32(2166136261)
	for _, c := range []byte(v) {
		h ^= uint32(c)
		h *= 16777619
	}
	const digits = "0123456789abcdef"
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = digits[h&0xf]
		h >>= 4
	}
	return string(b)
}
