package auth

import "github.com/heartmarshall/myenglish-backend/internal/domain"

// AuthResult is returned by Login and Refresh operations.
type AuthResult struct {
	AccessToken  string
	RefreshToken string       // raw token, NOT hash
	User         *domain.User
}
