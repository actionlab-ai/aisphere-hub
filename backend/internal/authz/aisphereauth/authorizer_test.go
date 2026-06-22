package aisphereauth

import (
	"testing"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
)

func TestActionForMapsResourceWritesToCRUD(t *testing.T) {
	tests := []struct {
		name   string
		action string
		res    core.ResourceRef
		want   string
	}{
		{
			name:   "skill label update",
			action: "skill:admin:write",
			res:    core.ResourceRef{Path: "/v3/aihub/skill/search/labels", HTTPMethod: "PUT"},
			want:   "update",
		},
		{
			name:   "skillset member bind",
			action: "skill:group:write",
			res:    core.ResourceRef{Path: "/v3/aihub/skillset/ops/skills", HTTPMethod: "POST"},
			want:   "create",
		},
		{
			name:   "skillset member unbind",
			action: "skill:group:write",
			res:    core.ResourceRef{Path: "/v3/aihub/skillset/ops/skills/search", HTTPMethod: "DELETE"},
			want:   "delete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := actionFor(tt.action, tt.res); got != tt.want {
				t.Fatalf("actionFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
