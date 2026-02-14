package translate

import "context"

// Stub is a no-op translation provider for MVP.
// Returns nil (no translations available).
type Stub struct{}

// NewStub creates a new no-op translation provider.
func NewStub() *Stub { return &Stub{} }

// FetchTranslations always returns nil — no translations on MVP.
func (s *Stub) FetchTranslations(ctx context.Context, word string) ([]string, error) {
	return nil, nil
}
