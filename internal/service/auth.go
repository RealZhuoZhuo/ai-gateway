package service

import (
	"context"
	"crypto/subtle"
)

type Authenticator struct {
	staticKeys []string
}

func NewAuthenticator(staticKeys []string) *Authenticator {
	return &Authenticator{staticKeys: staticKeys}
}

func (a *Authenticator) ValidAPIKey(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	return constantTimeContains(a.staticKeys, token), nil
}

func constantTimeContains(keys []string, token string) bool {
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(token)) == 1 {
			return true
		}
	}
	return false
}
