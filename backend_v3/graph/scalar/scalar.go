package scalar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/model"
)

// Используем type alias, чтобы типы были совместимы с internal/model без кастинга
type UUID = uuid.UUID
type Time = time.Time
type JSON = model.JSON

// ============================================================================
// UUID SCALAR
// ============================================================================

func MarshalUUID(u UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(fmt.Sprintf("%q", u.String())))
	})
}

func UnmarshalUUID(v interface{}) (UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	case []byte:
		return uuid.ParseBytes(v)
	default:
		return uuid.Nil, fmt.Errorf("UUID must be a string, got %T", v)
	}
}

// ============================================================================
// TIME SCALAR
// ============================================================================

func MarshalTime(t Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		// RFC3339Nano подходит для JSON
		w.Write([]byte(fmt.Sprintf("%q", t.Format(time.RFC3339Nano))))
	})
}

func UnmarshalTime(v interface{}) (Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	default:
		return time.Time{}, fmt.Errorf("Time must be a string, got %T", v)
	}
}

// ============================================================================
// JSON SCALAR
// ============================================================================

func MarshalJSON(j JSON) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		if j == nil {
			w.Write([]byte("null"))
			return
		}
		err := json.NewEncoder(w).Encode(j)
		if err != nil {
			panic(err) // Gqlgen ожидает панику при ошибке записи в WriterFunc
		}
	})
}

func UnmarshalJSON(v interface{}) (JSON, error) {
	if v == nil {
		return nil, nil
	}
	// Gqlgen обычно декодирует JSON объекты в map[string]interface{}
	if m, ok := v.(map[string]interface{}); ok {
		return JSON(m), nil
	}
	// Или в слайсы, если это массив
	if _, ok := v.([]interface{}); ok {
		return nil, errors.New("JSON scalar currently supports only objects, not arrays (update UnmarshalJSON if needed)")
	}

	return nil, fmt.Errorf("JSON must be an object, got %T", v)
}
