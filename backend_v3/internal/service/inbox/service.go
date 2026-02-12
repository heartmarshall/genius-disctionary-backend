package inbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/dictionary"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// DictionaryService определяет интерфейс для работы со словарем.
// Интерфейс объявлен здесь, так как используется в этом пакете.
type DictionaryService interface {
	// CreateWord создает новое слово со всеми связанными сущностями атомарно.
	CreateWord(ctx context.Context, input dictionary.CreateWordInput) (*model.DictionaryEntry, error)

	// UpdateWord обновляет существующее слово и все связанные сущности атомарно.
	UpdateWord(ctx context.Context, input dictionary.UpdateWordInput) (*model.DictionaryEntry, error)

	// DeleteWord удаляет слово и все связанные сущности атомарно.
	DeleteWord(ctx context.Context, input dictionary.DeleteWordInput) error
}

// Service реализует бизнес-логику для работы с inbox.
type Service struct {
	repos      *repository.Registry
	tx         *database.TxManager
	dictionary DictionaryService // Зависимость от другого сервиса через интерфейс
}

// NewService создает новый экземпляр сервиса inbox.
// DictionaryService внедряется через интерфейс для переиспользования логики создания слова.
// Возвращает ошибку, если repos, tx или dict равны nil.
func NewService(repos *repository.Registry, tx *database.TxManager, dict DictionaryService) (*Service, error) {
	if repos == nil {
		return nil, fmt.Errorf("repos cannot be nil")
	}
	if tx == nil {
		return nil, fmt.Errorf("tx cannot be nil")
	}
	if dict == nil {
		return nil, fmt.Errorf("dictionary service cannot be nil")
	}

	return &Service{
		repos:      repos,
		tx:         tx,
		dictionary: dict,
	}, nil
}

// AddToInbox создает новую заметку во входящих.
func (s *Service) AddToInbox(ctx context.Context, text string, contextStr *string) (*model.InboxItem, error) {
	if text == "" {
		return nil, types.NewValidationError("text", "cannot be empty")
	}

	item := &model.InboxItem{
		Text:    text,
		Context: contextStr,
	}

	created, err := s.repos.Inbox.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("create inbox item: %w", err)
	}

	return created, nil
}

// Delete удаляет заметку из входящих.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return types.NewValidationError("id", "cannot be nil")
	}

	err := s.repos.Inbox.Delete(ctx, id)
	if err != nil {
		if database.IsNotFoundError(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("delete inbox item: %w", err)
	}

	return nil
}

// List возвращает все входящие заметки.
// В будущем будет добавлена пагинация.
func (s *Service) List(ctx context.Context) ([]model.InboxItem, error) {
	items, err := s.repos.Inbox.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list inbox items: %w", err)
	}

	return items, nil
}

// ConvertToWord превращает InboxItem в DictionaryEntry.
// Метод проверяет существование inbox item и создает слово через DictionaryService.
// После успешного создания слова удаляет inbox item.
func (s *Service) ConvertToWord(ctx context.Context, inboxID uuid.UUID, input dictionary.CreateWordInput) (*model.DictionaryEntry, error) {
	if inboxID == uuid.Nil {
		return nil, types.NewValidationError("inboxID", "cannot be nil")
	}

	// Проверяем, что inbox item существует
	_, err := s.repos.Inbox.GetByID(ctx, inboxID)
	if err != nil {
		if database.IsNotFoundError(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get inbox item: %w", err)
	}

	// Создаем слово через DictionaryService
	entry, err := s.dictionary.CreateWord(ctx, input)
	if err != nil {
		// Сохраняем типизированные ошибки без обертки
		if errors.Is(err, types.ErrNotFound) ||
			errors.Is(err, types.ErrAlreadyExists) ||
			errors.Is(err, types.ErrInvalidInput) ||
			types.IsValidationError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("create word from inbox item: %w", err)
	}

	// Удаляем inbox item после успешного создания слова
	err = s.repos.Inbox.Delete(ctx, inboxID)
	if err != nil {
		// Если удаление не удалось, логируем ошибку, но не возвращаем её,
		// так как слово уже создано. В будущем можно добавить логирование.
		// Игнорируем ошибку "not found", так как элемент мог быть удален ранее
		if !database.IsNotFoundError(err) {
			// В продакшене здесь должно быть логирование ошибки
			// Для MVP просто игнорируем ошибку удаления
		}
	}

	return entry, nil
}
