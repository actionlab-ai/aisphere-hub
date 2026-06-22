package service

import (
	"context"
	"errors"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/sandbox"
)

var ErrSandboxDisabled = errors.New("sandbox manager is not enabled")

func (s *Service) EnsureSandbox(ctx context.Context, req model.SandboxEnsureRequest) (*model.SandboxStatus, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	return s.sandboxMgr.Ensure(ctx, req)
}

func (s *Service) GetSandbox(ctx context.Context, sandboxID string) (*model.SandboxStatus, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	return s.sandboxMgr.Get(ctx, sandboxID)
}

func (s *Service) ListSandboxes(ctx context.Context, q sandbox.ListQuery) ([]*model.SandboxStatus, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	return s.sandboxMgr.List(ctx, q)
}

func (s *Service) RestartSandbox(ctx context.Context, sandboxID string) (*model.SandboxStatus, error) {
	if s.sandboxMgr == nil {
		return nil, ErrSandboxDisabled
	}
	return s.sandboxMgr.Restart(ctx, sandboxID)
}

func (s *Service) DeleteSandbox(ctx context.Context, sandboxID string, deleteWorkspace bool) error {
	if s.sandboxMgr == nil {
		return ErrSandboxDisabled
	}
	return s.sandboxMgr.Delete(ctx, sandboxID, deleteWorkspace)
}

func (s *Service) SandboxLogs(ctx context.Context, sandboxID string, q model.SandboxLogQuery) (string, error) {
	if s.sandboxMgr == nil {
		return "", ErrSandboxDisabled
	}
	return s.sandboxMgr.Logs(ctx, sandboxID, q)
}
