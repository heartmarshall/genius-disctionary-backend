package model

import (
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

// DateTime wraps time.Time for GraphQL scalar marshaling.
type DateTime time.Time

func MarshalDateTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, `"`+t.Format(time.RFC3339)+`"`)
	})
}

func UnmarshalDateTime(v interface{}) (time.Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	default:
		return time.Time{}, fmt.Errorf("DateTime must be a string in RFC3339 format")
	}
}

// MarshalUUID marshals UUID to GraphQL string.
func MarshalUUID(u uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, `"`+u.String()+`"`)
	})
}

// UnmarshalUUID unmarshals GraphQL string to UUID.
func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	default:
		return uuid.UUID{}, fmt.Errorf("UUID must be a string")
	}
}
