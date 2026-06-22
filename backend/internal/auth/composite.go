package auth

import (
	"context"
	"errors"
	"net/http"
)

type CompositeAuthenticator struct {
	providers []AuthProvider
	mapper    SubjectMapper
}

func NewComposite(providers []AuthProvider, mapper SubjectMapper) *CompositeAuthenticator {
	return &CompositeAuthenticator{providers: providers, mapper: mapper}
}

func (c *CompositeAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*Principal, bool, error) {
	var lastErr error
	for _, p := range c.providers {
		id, ok, err := p.Authenticate(ctx, r)
		if err != nil {
			lastErr = err
			continue
		}
		if !ok {
			continue
		}
		principal, err := c.mapper.Map(ctx, id)
		if err != nil {
			return nil, false, err
		}
		return principal, true, nil
	}
	if lastErr != nil {
		return nil, false, lastErr
	}
	return nil, false, nil
}

var ErrForbidden = errors.New("forbidden")
