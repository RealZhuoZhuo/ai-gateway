package service

import (
	"context"
	"crypto/subtle"
	"errors"

	"github.com/RealZhuoZhuo/ai-gateway/internal/repo"
)

type APIKeyRepository interface {
	ValidAPIKey(ctx context.Context, token string) (bool, error)
}

type Authenticator struct {
	staticKeys []string
	repo       APIKeyRepository
}

func NewAuthenticator(staticKeys []string, repo APIKeyRepository) *Authenticator {
	return &Authenticator{staticKeys: staticKeys, repo: repo}
}

func (a *Authenticator) ValidAPIKey(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	if constantTimeContains(a.staticKeys, token) {
		return true, nil
	}
	if a.repo == nil {
		return false, nil
	}
	allowed, err := a.repo.ValidAPIKey(ctx, token)
	if errors.Is(err, repo.ErrNotConfigured) {
		return false, nil
	}
	return allowed, err
}

func constantTimeContains(keys []string, token string) bool {
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(token)) == 1 {
			return true
		}
	}
	return false
}
