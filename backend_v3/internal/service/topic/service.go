package topic

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

type Service struct {
	repos *repository.Registry
	tx    *database.TxManager
}

func NewService(repos *repository.Registry, tx *database.TxManager) *Service {
	return &Service{
		repos: repos,
		tx:    tx,
	}
}

// CreateTopic создает новый топик.
func (s *Service) CreateTopic(ctx context.Context, input CreateTopicInput) (*model.Topic, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, types.NewValidationError("name", "cannot be empty")
	}

	topic := &model.Topic{
		Name:        name,
		Description: input.Description,
	}

	created, err := s.repos.Topics.Create(ctx, topic)
	if err != nil {
		if database.IsDuplicateError(err) {
			return nil, types.NewValidationError("name", "topic already exists")
		}
		return nil, fmt.Errorf("create topic: %w", err)
	}

	return created, nil
}

// UpdateTopic обновляет топик.
func (s *Service) UpdateTopic(ctx context.Context, input UpdateTopicInput) (*model.Topic, error) {
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, types.NewValidationError("id", "invalid format")
	}

	// Получаем текущий топик
	current, err := s.repos.Topics.GetByID(ctx, id)
	if err != nil {
		if database.IsNotFoundError(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get topic: %w", err)
	}

	// Обновляем поля
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, types.NewValidationError("name", "cannot be empty")
		}
		current.Name = name
	}
	if input.Description != nil {
		current.Description = input.Description
	}

	updated, err := s.repos.Topics.Update(ctx, id, current)
	if err != nil {
		if database.IsDuplicateError(err) {
			return nil, types.NewValidationError("name", "topic already exists")
		}
		return nil, fmt.Errorf("update topic: %w", err)
	}

	return updated, nil
}

// DeleteTopic удаляет топик.
func (s *Service) DeleteTopic(ctx context.Context, input DeleteTopicInput) error {
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return types.NewValidationError("id", "invalid format")
	}

	err = s.repos.Topics.Delete(ctx, id)
	if err != nil {
		if database.IsNotFoundError(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("delete topic: %w", err)
	}

	return nil
}

// ListTopics возвращает список всех топиков.
func (s *Service) ListTopics(ctx context.Context) ([]model.Topic, error) {
	return s.repos.Topics.ListAll(ctx)
}

// GetByID возвращает топик по ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error) {
	topic, err := s.repos.Topics.GetByID(ctx, id)
	if err != nil {
		if database.IsNotFoundError(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get topic: %w", err)
	}
	return topic, nil
}
