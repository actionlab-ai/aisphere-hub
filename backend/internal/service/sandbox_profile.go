package service

import (
	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"strings"
)

func (s *Service) ListSandboxProfiles(namespaceID string) ([]*model.SandboxProfile, error) {
	var out []*model.SandboxProfile
	err := s.store.WithRead(func() error { var err error; out, err = s.store.ListSandboxProfiles(namespaceID); return err })
	return out, err
}

func (s *Service) GetSandboxProfile(namespaceID, id string) (*model.SandboxProfile, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httputil.BadRequest("profile id is required")
	}
	var out *model.SandboxProfile
	err := s.store.WithRead(func() error { var err error; out, err = s.store.LoadSandboxProfile(namespaceID, id); return err })
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, httputil.NotFound("sandbox profile not found")
	}
	return out, nil
}

func (s *Service) SaveSandboxProfile(namespaceID string, p *model.SandboxProfile) (*model.SandboxProfile, error) {
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return nil, httputil.BadRequest("profile id is required")
	}
	if p.Status == "" {
		p.Status = model.MetaStatusEnable
	}
	if p.Driver == "" {
		p.Driver = "agent-sandbox"
	}
	if p.Workspace.MountPath == "" {
		p.Workspace.MountPath = "/workspace"
	}
	if p.Network.Mode == "" {
		p.Network.Mode = "offline"
	}
	p.NamespaceID = namespaceID
	now := model.NowMillis()
	if p.CreateTime == 0 {
		p.CreateTime = now
	}
	p.UpdateTime = now
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	err := s.store.WithWrite(func() error { return s.store.SaveSandboxProfile(p) })
	return p, err
}

func (s *Service) DeleteSandboxProfile(namespaceID, id string) error {
	if strings.TrimSpace(id) == "" {
		return httputil.BadRequest("profile id is required")
	}
	return s.store.WithWrite(func() error { return s.store.DeleteSandboxProfile(namespaceID, id) })
}

func (s *Service) ListSandboxPolicies(namespaceID string) ([]*model.SandboxPolicy, error) {
	var out []*model.SandboxPolicy
	err := s.store.WithRead(func() error { var err error; out, err = s.store.ListSandboxPolicies(namespaceID); return err })
	return out, err
}

func (s *Service) GetSandboxPolicy(namespaceID, id string) (*model.SandboxPolicy, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httputil.BadRequest("policy id is required")
	}
	var out *model.SandboxPolicy
	err := s.store.WithRead(func() error { var err error; out, err = s.store.LoadSandboxPolicy(namespaceID, id); return err })
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, httputil.NotFound("sandbox policy not found")
	}
	return out, nil
}

func (s *Service) SaveSandboxPolicy(namespaceID string, p *model.SandboxPolicy) (*model.SandboxPolicy, error) {
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return nil, httputil.BadRequest("policy id is required")
	}
	p.NamespaceID = namespaceID
	now := model.NowMillis()
	if p.CreateTime == 0 {
		p.CreateTime = now
	}
	p.UpdateTime = now
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	err := s.store.WithWrite(func() error { return s.store.SaveSandboxPolicy(p) })
	return p, err
}

func (s *Service) DeleteSandboxPolicy(namespaceID, id string) error {
	if strings.TrimSpace(id) == "" {
		return httputil.BadRequest("policy id is required")
	}
	return s.store.WithWrite(func() error { return s.store.DeleteSandboxPolicy(namespaceID, id) })
}
