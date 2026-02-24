package middleware

import (
	"context"
	"github.com/google/uuid"
	"sync"
)

var _ tokenValidator = &tokenValidatorMock{}

type tokenValidatorMock struct {
	ValidateTokenFunc func(ctx context.Context, token string) (uuid.UUID, string, error)

	calls struct {
		ValidateToken []struct {
			Ctx   context.Context
			Token string
		}
	}
	lockValidateToken sync.RWMutex
}

func (mock *tokenValidatorMock) ValidateToken(ctx context.Context, token string) (uuid.UUID, string, error) {
	if mock.ValidateTokenFunc == nil {
		panic("tokenValidatorMock.ValidateTokenFunc: method is nil but tokenValidator.ValidateToken was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Token string
	}{Ctx: ctx, Token: token}
	mock.lockValidateToken.Lock()
	mock.calls.ValidateToken = append(mock.calls.ValidateToken, callInfo)
	mock.lockValidateToken.Unlock()
	return mock.ValidateTokenFunc(ctx, token)
}

func (mock *tokenValidatorMock) ValidateTokenCalls() []struct {
	Ctx   context.Context
	Token string
} {
	mock.lockValidateToken.RLock()
	calls := mock.calls.ValidateToken
	mock.lockValidateToken.RUnlock()
	return calls
}
