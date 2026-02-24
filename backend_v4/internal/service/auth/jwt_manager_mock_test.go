package auth

import (
	"github.com/google/uuid"
	"sync"
)

var _ jwtManager = &jwtManagerMock{}

type jwtManagerMock struct {
	GenerateAccessTokenFunc  func(userID uuid.UUID, role string) (string, error)
	GenerateRefreshTokenFunc func() (string, string, error)
	ValidateAccessTokenFunc  func(token string) (uuid.UUID, string, error)

	calls struct {
		GenerateAccessToken []struct {
			UserID uuid.UUID
			Role   string
		}
		GenerateRefreshToken []struct{}
		ValidateAccessToken  []struct {
			Token string
		}
	}
	lockGenerateAccessToken  sync.RWMutex
	lockGenerateRefreshToken sync.RWMutex
	lockValidateAccessToken  sync.RWMutex
}

func (mock *jwtManagerMock) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	if mock.GenerateAccessTokenFunc == nil {
		panic("jwtManagerMock.GenerateAccessTokenFunc: method is nil but jwtManager.GenerateAccessToken was just called")
	}
	callInfo := struct {
		UserID uuid.UUID
		Role   string
	}{UserID: userID, Role: role}
	mock.lockGenerateAccessToken.Lock()
	mock.calls.GenerateAccessToken = append(mock.calls.GenerateAccessToken, callInfo)
	mock.lockGenerateAccessToken.Unlock()
	return mock.GenerateAccessTokenFunc(userID, role)
}

func (mock *jwtManagerMock) GenerateAccessTokenCalls() []struct {
	UserID uuid.UUID
	Role   string
} {
	mock.lockGenerateAccessToken.RLock()
	calls := mock.calls.GenerateAccessToken
	mock.lockGenerateAccessToken.RUnlock()
	return calls
}

func (mock *jwtManagerMock) GenerateRefreshToken() (string, string, error) {
	if mock.GenerateRefreshTokenFunc == nil {
		panic("jwtManagerMock.GenerateRefreshTokenFunc: method is nil but jwtManager.GenerateRefreshToken was just called")
	}
	mock.lockGenerateRefreshToken.Lock()
	mock.calls.GenerateRefreshToken = append(mock.calls.GenerateRefreshToken, struct{}{})
	mock.lockGenerateRefreshToken.Unlock()
	return mock.GenerateRefreshTokenFunc()
}

func (mock *jwtManagerMock) GenerateRefreshTokenCalls() []struct{} {
	mock.lockGenerateRefreshToken.RLock()
	calls := mock.calls.GenerateRefreshToken
	mock.lockGenerateRefreshToken.RUnlock()
	return calls
}

func (mock *jwtManagerMock) ValidateAccessToken(token string) (uuid.UUID, string, error) {
	if mock.ValidateAccessTokenFunc == nil {
		panic("jwtManagerMock.ValidateAccessTokenFunc: method is nil but jwtManager.ValidateAccessToken was just called")
	}
	callInfo := struct{ Token string }{Token: token}
	mock.lockValidateAccessToken.Lock()
	mock.calls.ValidateAccessToken = append(mock.calls.ValidateAccessToken, callInfo)
	mock.lockValidateAccessToken.Unlock()
	return mock.ValidateAccessTokenFunc(token)
}

func (mock *jwtManagerMock) ValidateAccessTokenCalls() []struct{ Token string } {
	mock.lockValidateAccessToken.RLock()
	calls := mock.calls.ValidateAccessToken
	mock.lockValidateAccessToken.RUnlock()
	return calls
}
