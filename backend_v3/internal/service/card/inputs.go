package card

import (
	"time"

	"github.com/heartmarshall/my-english/internal/model"
)

// CreateCardInput — входные данные для создания карточки.
type CreateCardInput struct {
	EntryID      string                // UUID записи словаря
	Status       *model.LearningStatus // Опционально, по умолчанию NEW
	NextReviewAt *time.Time            // Опционально
	IntervalDays *int                  // Опционально, по умолчанию 0
	EaseFactor   *float64              // Опционально, по умолчанию 2.5
}

// UpdateCardInput — входные данные для обновления карточки.
type UpdateCardInput struct {
	ID           string                // UUID карточки
	Status       *model.LearningStatus // Опционально
	NextReviewAt *time.Time            // Опционально (nil для сброса)
	IntervalDays *int                  // Опционально
	EaseFactor   *float64              // Опционально
}
