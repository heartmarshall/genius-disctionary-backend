package dictionary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"

	// Используем именованный импорт, чтобы избежать конфликта имен пакетов
	repo_dictionary "github.com/heartmarshall/my-english/internal/database/repository/dictionary"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// DictionaryFilter — псевдоним типа из репозитория.
// Это позволяет клиентам сервиса использовать этот тип, импортируя только сервис.
type DictionaryFilter = repo_dictionary.DictionaryFilter

// Find ищет слова по фильтру.
func (s *Service) Find(ctx context.Context, filter DictionaryFilter) ([]model.DictionaryEntry, error) {
	entries, err := s.repos.Dictionary.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find dictionary entries: %w", err)
	}
	return entries, nil
}

// GetByID получает слово по ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.DictionaryEntry, error) {
	entry, err := s.repos.Dictionary.GetByID(ctx, id)
	if err != nil {
		if database.IsNotFoundError(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get dictionary entry: %w", err)
	}
	return entry, nil
}
