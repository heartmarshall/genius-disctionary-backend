package card

import (
	"context"
	"fmt"

	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
)

// Service реализует бизнес-логику для работы с карточками заучивания.
type Service struct {
	repos *repository.Registry
	tx    *database.TxManager
}

// NewService создает новый экземпляр сервиса карточек.
func NewService(repos *repository.Registry, tx *database.TxManager) (*Service, error) {
	if repos == nil {
		return nil, fmt.Errorf("repos cannot be nil")
	}
	if tx == nil {
		return nil, fmt.Errorf("tx cannot be nil")
	}

	return &Service{
		repos: repos,
		tx:    tx,
	}, nil
}

// CreateCard создает новую карточку для записи словаря.
// Метод выполняет валидацию входных данных, проверку существования записи словаря,
// проверку на дубликат карточки и создание карточки.
func (s *Service) CreateCard(ctx context.Context, input CreateCardInput) (*model.Card, error) {
	if err := validateCreateCardInput(input); err != nil {
		return nil, err
	}

	entryID, err := parseID(input.EntryID)
	if err != nil {
		return nil, err
	}

	card, err := s.createCardTx(ctx, input, entryID)
	if err != nil {
		return nil, wrapServiceError(err, "create card")
	}

	return card, nil
}

// UpdateCard обновляет карточку.
// Метод выполняет валидацию входных данных, проверку существования карточки
// и обновление только указанных полей.
func (s *Service) UpdateCard(ctx context.Context, input UpdateCardInput) (*model.Card, error) {
	if err := validateUpdateCardInput(input); err != nil {
		return nil, err
	}

	cardID, err := parseID(input.ID)
	if err != nil {
		return nil, err
	}

	card, err := s.updateCardTx(ctx, input, cardID)
	if err != nil {
		return nil, wrapServiceError(err, "update card")
	}

	return card, nil
}
