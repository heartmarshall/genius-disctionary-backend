package resolver

import (
	"encoding/base64"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// encodeCursor encodes a sort value and entry ID into a base64 cursor
// using the "sortValue|entryID" format expected by the repository.
func encodeCursor(sortValue, entryID string) string {
	raw := sortValue + "|" + entryID
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// cursorFromEntry builds a cursor for the given entry and sort column.
func cursorFromEntry(e domain.Entry, sortBy string) string {
	var sortValue string
	switch sortBy {
	case "text":
		sortValue = e.TextNormalized
	case "updated_at":
		sortValue = e.UpdatedAt.Format(time.RFC3339Nano)
	default:
		sortValue = e.CreatedAt.Format(time.RFC3339Nano)
	}
	return encodeCursor(sortValue, e.ID.String())
}

func toSensePointers(senses []domain.Sense) []*domain.Sense {
	result := make([]*domain.Sense, len(senses))
	for i := range senses {
		result[i] = &senses[i]
	}
	return result
}

func toTranslationPointers(translations []domain.Translation) []*domain.Translation {
	result := make([]*domain.Translation, len(translations))
	for i := range translations {
		result[i] = &translations[i]
	}
	return result
}

func toExamplePointers(examples []domain.Example) []*domain.Example {
	result := make([]*domain.Example, len(examples))
	for i := range examples {
		result[i] = &examples[i]
	}
	return result
}

func toPronunciationPointers(prons []domain.RefPronunciation) []*domain.RefPronunciation {
	result := make([]*domain.RefPronunciation, len(prons))
	for i := range prons {
		result[i] = &prons[i]
	}
	return result
}

func toRefImagePointers(images []domain.RefImage) []*domain.RefImage {
	result := make([]*domain.RefImage, len(images))
	for i := range images {
		result[i] = &images[i]
	}
	return result
}

func toUserImagePointers(images []domain.UserImage) []*domain.UserImage {
	result := make([]*domain.UserImage, len(images))
	for i := range images {
		result[i] = &images[i]
	}
	return result
}

func toTopicPointers(topics []domain.Topic) []*domain.Topic {
	result := make([]*domain.Topic, len(topics))
	for i := range topics {
		result[i] = &topics[i]
	}
	return result
}
