package generated

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/ast"
)

// UUID scalar methods
func (ec *executionContext) unmarshalInputUUID(ctx context.Context, obj interface{}) (uuid.UUID, error) {
	var res uuid.UUID
	err := res.UnmarshalText([]byte(obj.(string)))
	return res, err
}

func (ec *executionContext) marshalUUID(ctx context.Context, sel ast.SelectionSet, v uuid.UUID) graphql.Marshaler {
	res := graphql.MarshalString(v.String())
	return res
}

func (ec *executionContext) _UUID(ctx context.Context, sel ast.SelectionSet, v *uuid.UUID) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			ec.Errorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec.marshalUUID(ctx, sel, *v)
}

// DateTime scalar methods
func (ec *executionContext) unmarshalInputDateTime(ctx context.Context, obj interface{}) (time.Time, error) {
	switch v := obj.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	default:
		return time.Time{}, fmt.Errorf("DateTime must be a string in RFC3339 format")
	}
}

func (ec *executionContext) marshalDateTime(ctx context.Context, sel ast.SelectionSet, v time.Time) graphql.Marshaler {
	res := graphql.MarshalString(v.Format(time.RFC3339))
	return res
}

func (ec *executionContext) _DateTime(ctx context.Context, sel ast.SelectionSet, v *time.Time) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			ec.Errorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec.marshalDateTime(ctx, sel, *v)
}
