package service

import (
	"context"
	"errors"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/toolgateway"
)

func (s *Service) ListSandboxTools(ctx context.Context, sandboxID string) (*model.SandboxToolListResponse, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	st, err := s.sandboxMgr.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	endpoint := endpointByName(st.Endpoints, "tools")
	if endpoint == "" {
		return nil, errors.New("sandbox tools endpoint not found")
	}
	out, err := toolgateway.NewHTTPGateway(nil).ListTools(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	out.SandboxID = st.SandboxID
	out.ModelTools = toolgateway.BuildModelToolSpecs(out.Tools)
	return out, nil
}

func (s *Service) CallSandboxTool(ctx context.Context, sandboxID string, req model.SandboxToolCallRequest) (*model.SandboxToolCallResult, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	st, err := s.sandboxMgr.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	endpoint := endpointByName(st.Endpoints, "tools")
	if endpoint == "" {
		return nil, errors.New("sandbox tools endpoint not found")
	}
	out, err := toolgateway.NewHTTPGateway(nil).Call(ctx, endpoint, req)
	if out != nil {
		out.SandboxID = st.SandboxID
	}
	return out, err
}

func endpointByName(items []model.SandboxEndpoint, name string) string {
	for _, ep := range items {
		if strings.EqualFold(ep.Name, name) && strings.TrimSpace(ep.URL) != "" {
			return strings.TrimSpace(ep.URL)
		}
	}
	return ""
}
