package dataloader

import (
	"context"
	"errors"
)

// MustFor извлекает Loaders из контекста или возвращает ошибку.
// Используется в resolvers для безопасного получения DataLoaders.
func MustFor(ctx context.Context) (*Loaders, error) {
	loaders := For(ctx)
	if loaders == nil {
		return nil, errors.New("dataloaders not available in context")
	}
	return loaders, nil
}
