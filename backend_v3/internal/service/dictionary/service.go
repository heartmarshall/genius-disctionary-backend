package dictionary

import (
	"context"
	"fmt"

	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
)

// Service реализует бизнес-логику для работы со словарем.
type Service struct {
	repos *repository.Registry
	tx    *database.TxManager
}

// NewService создает новый экземпляр сервиса словаря.
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

// CreateWord создает слово и все связанные сущности атомарно.
// Метод выполняет валидацию входных данных, проверку на дубликаты,
// создание основной записи и всех связанных сущностей (смыслы, переводы,
// примеры, изображения, произношения), а также опционально создает карточку для изучения.
func (s *Service) CreateWord(ctx context.Context, input CreateWordInput) (*model.DictionaryEntry, error) {
	if err := validateCreateWordInput(input); err != nil {
		return nil, err
	}

	textRaw := normalizeText(input.Text)
	textNorm := textRaw

	entry, err := s.createWordTx(ctx, input, textRaw, textNorm)
	if err != nil {
		return nil, wrapServiceError(err, "create word")
	}

	return entry, nil
}

// UpdateWord обновляет слово и все связанные сущности атомарно.
// Метод выполняет валидацию входных данных, проверку существования записи,
// обновление основной записи и пересоздание всех связанных сущностей.
func (s *Service) UpdateWord(ctx context.Context, input UpdateWordInput) (*model.DictionaryEntry, error) {
	if err := validateUpdateWordInput(input); err != nil {
		return nil, err
	}

	entryID, err := parseEntryID(input.ID)
	if err != nil {
		return nil, err
	}

	entry, err := s.updateWordTx(ctx, input, entryID)
	if err != nil {
		return nil, wrapServiceError(err, "update word")
	}

	return entry, nil
}

// DeleteWord удаляет слово и все связанные сущности атомарно.
// Метод выполняет валидацию ID, проверку существования записи,
// удаление записи и создание аудит-лога.
func (s *Service) DeleteWord(ctx context.Context, input DeleteWordInput) error {
	entryID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deleteWordTx(ctx, entryID); err != nil {
		return wrapServiceError(err, "delete word")
	}

	return nil
}

// AddSense добавляет новый смысл к записи словаря без удаления существующих.
// Метод создает смысл и связанные с ним переводы и примеры.
func (s *Service) AddSense(ctx context.Context, input AddSenseInput) (*model.Sense, error) {
	if err := validateAddSenseInput(input); err != nil {
		return nil, err
	}

	entryID, err := parseEntryID(input.EntryID)
	if err != nil {
		return nil, err
	}

	sense, err := s.addSenseTx(ctx, input, entryID)
	if err != nil {
		return nil, wrapServiceError(err, "add sense")
	}

	return sense, nil
}

// AddExamples добавляет новые примеры к существующему смыслу без удаления существующих.
func (s *Service) AddExamples(ctx context.Context, input AddExamplesInput) error {
	if err := validateAddExamplesInput(input); err != nil {
		return err
	}

	senseID, err := parseEntryID(input.SenseID)
	if err != nil {
		return err
	}

	if err := s.addExamplesTx(ctx, input, senseID); err != nil {
		return wrapServiceError(err, "add examples")
	}

	return nil
}

// AddTranslations добавляет новые переводы к существующему смыслу без удаления существующих.
func (s *Service) AddTranslations(ctx context.Context, input AddTranslationsInput) error {
	if err := validateAddTranslationsInput(input); err != nil {
		return err
	}

	senseID, err := parseEntryID(input.SenseID)
	if err != nil {
		return err
	}

	if err := s.addTranslationsTx(ctx, input, senseID); err != nil {
		return wrapServiceError(err, "add translations")
	}

	return nil
}

// AddImages добавляет новые изображения к записи словаря без удаления существующих.
func (s *Service) AddImages(ctx context.Context, input AddImagesInput) error {
	if err := validateAddImagesInput(input); err != nil {
		return err
	}

	entryID, err := parseEntryID(input.EntryID)
	if err != nil {
		return err
	}

	if err := s.addImagesTx(ctx, input, entryID); err != nil {
		return wrapServiceError(err, "add images")
	}

	return nil
}

// AddPronunciations добавляет новые произношения к записи словаря без удаления существующих.
func (s *Service) AddPronunciations(ctx context.Context, input AddPronunciationsInput) error {
	if err := validateAddPronunciationsInput(input); err != nil {
		return err
	}

	entryID, err := parseEntryID(input.EntryID)
	if err != nil {
		return err
	}

	if err := s.addPronunciationsTx(ctx, input, entryID); err != nil {
		return wrapServiceError(err, "add pronunciations")
	}

	return nil
}

// DeleteSense удаляет смысл и все связанные с ним переводы и примеры (CASCADE).
func (s *Service) DeleteSense(ctx context.Context, input DeleteSenseInput) error {
	if err := validateDeleteSenseInput(input); err != nil {
		return err
	}

	senseID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deleteSenseTx(ctx, senseID); err != nil {
		return wrapServiceError(err, "delete sense")
	}

	return nil
}

// DeleteExample удаляет пример.
func (s *Service) DeleteExample(ctx context.Context, input DeleteExampleInput) error {
	if err := validateDeleteExampleInput(input); err != nil {
		return err
	}

	exampleID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deleteExampleTx(ctx, exampleID); err != nil {
		return wrapServiceError(err, "delete example")
	}

	return nil
}

// DeleteTranslation удаляет перевод.
func (s *Service) DeleteTranslation(ctx context.Context, input DeleteTranslationInput) error {
	if err := validateDeleteTranslationInput(input); err != nil {
		return err
	}

	translationID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deleteTranslationTx(ctx, translationID); err != nil {
		return wrapServiceError(err, "delete translation")
	}

	return nil
}

// DeleteImage удаляет изображение.
func (s *Service) DeleteImage(ctx context.Context, input DeleteImageInput) error {
	if err := validateDeleteImageInput(input); err != nil {
		return err
	}

	imageID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deleteImageTx(ctx, imageID); err != nil {
		return wrapServiceError(err, "delete image")
	}

	return nil
}

// DeletePronunciation удаляет произношение.
func (s *Service) DeletePronunciation(ctx context.Context, input DeletePronunciationInput) error {
	if err := validateDeletePronunciationInput(input); err != nil {
		return err
	}

	pronunciationID, err := parseEntryID(input.ID)
	if err != nil {
		return err
	}

	if err := s.deletePronunciationTx(ctx, pronunciationID); err != nil {
		return wrapServiceError(err, "delete pronunciation")
	}

	return nil
}

// AddTopics добавляет топики к записи словаря без удаления существующих.
func (s *Service) AddTopics(ctx context.Context, input AddTopicsInput) error {
	if err := validateAddTopicsInput(input); err != nil {
		return err
	}

	entryID, err := parseEntryID(input.EntryID)
	if err != nil {
		return err
	}

	if err := s.addTopicsTx(ctx, input, entryID); err != nil {
		return wrapServiceError(err, "add topics")
	}

	return nil
}

// RemoveTopics удаляет топики у записи словаря.
func (s *Service) RemoveTopics(ctx context.Context, input RemoveTopicsInput) error {
	if err := validateRemoveTopicsInput(input); err != nil {
		return err
	}

	entryID, err := parseEntryID(input.EntryID)
	if err != nil {
		return err
	}

	if err := s.removeTopicsTx(ctx, input, entryID); err != nil {
		return wrapServiceError(err, "remove topics")
	}

	return nil
}
