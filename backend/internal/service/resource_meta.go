package service

import (
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func normalizedResourceStatus(status string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return model.MetaStatusEnable, nil
	}
	switch s {
	case model.MetaStatusEnable, "enabled", "online", "active":
		return model.MetaStatusEnable, nil
	case model.MetaStatusDisable, "disabled", "offline", "inactive":
		return model.MetaStatusDisable, nil
	default:
		return "", httputil.BadRequest("status must be enable or disable")
	}
}

func normalizedResourceLabels(labels map[string]string, latestVersion string) map[string]string {
	out := map[string]string{}
	for k, v := range labels {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if strings.TrimSpace(latestVersion) != "" {
		out[model.LabelLatest] = strings.TrimSpace(latestVersion)
	}
	return out
}

func mergeResourceLabels(existing map[string]string, incoming map[string]string, latestVersion string) map[string]string {
	out := map[string]string{}
	for k, v := range existing {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key != "" && val != "" {
			out[key] = val
		}
	}
	for k, v := range incoming {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" {
			continue
		}
		if val == "" {
			delete(out, key)
			continue
		}
		out[key] = val
	}
	if strings.TrimSpace(latestVersion) != "" {
		out[model.LabelLatest] = strings.TrimSpace(latestVersion)
	}
	return out
}

func ensureRunnableResource(resourceType, id, status string) error {
	if strings.EqualFold(strings.TrimSpace(status), model.MetaStatusDisable) {
		return httputil.Conflict(resourceType + " is disabled: " + strings.TrimSpace(id))
	}
	return nil
}
