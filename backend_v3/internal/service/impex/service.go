package impex

import (
	"fmt"

	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/service/dictionary"
)

// Service реализует бизнес-логику для импорта и экспорта словаря.
type Service struct {
	repos         *repository.Registry
	dictionarySvc *dictionary.Service
}

// NewService создает новый экземпляр сервиса импорта/экспорта.
func NewService(repos *repository.Registry, dictionarySvc *dictionary.Service) (*Service, error) {
	if repos == nil {
		return nil, fmt.Errorf("repos cannot be nil")
	}
	if dictionarySvc == nil {
		return nil, fmt.Errorf("dictionary service cannot be nil")
	}

	return &Service{
		repos:         repos,
		dictionarySvc: dictionarySvc,
	}, nil
}
