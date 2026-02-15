package auth

// OAuthIdentity represents user information obtained from an OAuth provider.
type OAuthIdentity struct {
	Email      string
	Name       *string
	AvatarURL  *string
	ProviderID string
}
