package model

const (
	CatalogEventSkillUpdated       = "skill.updated"
	CatalogEventSkillPublished     = "skill.published"
	CatalogEventSkillDeleted       = "skill.deleted"
	CatalogEventSkillStatusChanged = "skill.status_changed"
	CatalogEventSkillSetUpdated    = "skillset.updated"
	CatalogEventSkillSetDeleted    = "skillset.deleted"
	CatalogEventAgentUpdated       = "agent.updated"
	CatalogEventAgentDeleted       = "agent.deleted"
	CatalogEventToolUpdated        = "tool.updated"
	CatalogEventToolDeleted        = "tool.deleted"
	CatalogEventServiceResolved    = "service.resolved"
	CatalogEventGrantChanged       = "grant.changed"
	CatalogEventRuntimeReported    = "runtime.reported"
)

type CatalogEvent struct {
	ID           int64                  `json:"id"`
	App          string                 `json:"app"`
	EventType    string                 `json:"type"`
	Object       string                 `json:"object"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId"`
	SkillSetName string                 `json:"skillset,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Revision     string                 `json:"revision,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	CreatedAt    int64                  `json:"createdAt"`
}

type CatalogEventQuery struct {
	App          string
	SkillSetName string
	ResourceType string
	ResourceID   string
	SinceID      int64
	Limit        int
}
