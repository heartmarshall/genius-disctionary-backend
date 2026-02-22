package resolver

import (
	"encoding/base64"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// encodeCursor encodes an ID string into a base64 cursor.
func encodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(id))
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
