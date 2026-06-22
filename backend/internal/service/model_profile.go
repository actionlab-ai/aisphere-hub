package service

import (
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func (s *Service) ListModelProfiles(namespaceID string) ([]*model.ModelProfile, error) {
	var out []*model.ModelProfile
	err := s.store.WithRead(func() error { var err error; out, err = s.store.ListModelProfiles(namespaceID); return err })
	return out, err
}

func (s *Service) GetModelProfile(namespaceID, id string) (*model.ModelProfile, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httputil.BadRequest("model profile id is required")
	}
	var out *model.ModelProfile
	err := s.store.WithRead(func() error { var err error; out, err = s.store.LoadModelProfile(namespaceID, id); return err })
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, httputil.NotFound("model profile not found")
	}
	return out, nil
}

func (s *Service) SaveModelProfile(namespaceID string, p *model.ModelProfile) (*model.ModelProfile, error) {
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return nil, httputil.BadRequest("model profile id is required")
	}
	p.NamespaceID = namespaceID
	if p.Status == "" {
		p.Status = model.MetaStatusEnable
	}
	if p.Provider == "" {
		p.Provider = "openai-compatible"
	}
	if p.APIFormat == "" {
		p.APIFormat = "openai-chat-completions"
	}
	if p.Model == "" {
		p.Model = p.ID
	}
	if p.UpstreamModel == "" {
		p.UpstreamModel = p.Model
	}
	if p.UpstreamPath == "" {
		p.UpstreamPath = "/v1/chat/completions"
	}
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	now := model.NowMillis()
	if p.CreateTime == 0 {
		p.CreateTime = now
	}
	p.UpdateTime = now
	err := s.store.WithWrite(func() error { return s.store.SaveModelProfile(p) })
	return p, err
}

func (s *Service) DeleteModelProfile(namespaceID, id string) error {
	if strings.TrimSpace(id) == "" {
		return httputil.BadRequest("model profile id is required")
	}
	return s.store.WithWrite(func() error { return s.store.DeleteModelProfile(namespaceID, id) })
}
