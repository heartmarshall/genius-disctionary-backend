package impex

import (
	"github.com/heartmarshall/my-english/internal/model"
)

// ExportEntry представляет полную структуру слова для экспорта в JSON.
type ExportEntry struct {
	// Основная запись словаря
	Entry *model.DictionaryEntry `json:"entry"`

	// Связанные сущности
	Senses         []ExportSense         `json:"senses"`
	Images         []model.Image         `json:"images"`
	Pronunciations []model.Pronunciation `json:"pronunciations"`
	TopicNames     []string              `json:"topicNames,omitempty"` // Имена топиков (используем name вместо ID для переносимости)

	// Карточка (если есть)
	Card *ExportCard `json:"card,omitempty"`
}

// ExportSense представляет смысл с переводами и примерами.
type ExportSense struct {
	*model.Sense
	Translations []model.Translation `json:"translations"`
	Examples     []model.Example     `json:"examples"`
}

// ExportCard представляет карточку для экспорта.
type ExportCard struct {
	*model.Card
}

// ImportReport содержит статистику импорта.
type ImportReport struct {
	TotalProcessed int      `json:"totalProcessed"`
	SuccessCount   int      `json:"successCount"`
	SkippedCount   int      `json:"skippedCount"`
	FailedCount    int      `json:"failedCount"`
	Errors         []string `json:"errors"`
}
