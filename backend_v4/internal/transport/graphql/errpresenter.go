package graphql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// NewErrorPresenter returns a gqlgen error presenter that maps domain errors
// to GraphQL error codes.
func NewErrorPresenter(log *slog.Logger) graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		// Get original error (gqlgen wraps errors)
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)

		// Unwrap to domain error
		var origErr error
		if unwrapped := errors.Unwrap(err); unwrapped != nil {
			origErr = unwrapped
		} else {
			origErr = err
		}

		switch {
		case errors.Is(origErr, domain.ErrNotFound):
			gqlErr.Extensions = map[string]interface{}{"code": "NOT_FOUND"}

		case errors.Is(origErr, domain.ErrAlreadyExists):
			gqlErr.Extensions = map[string]interface{}{"code": "ALREADY_EXISTS"}

		case errors.Is(origErr, domain.ErrValidation):
			gqlErr.Extensions = map[string]interface{}{"code": "VALIDATION"}
			var ve *domain.ValidationError
			if errors.As(err, &ve) {
				gqlErr.Extensions["fields"] = ve.Errors
			}

		case errors.Is(origErr, domain.ErrUnauthorized):
			gqlErr.Extensions = map[string]interface{}{"code": "UNAUTHENTICATED"}

		case errors.Is(origErr, domain.ErrForbidden):
			gqlErr.Extensions = map[string]interface{}{"code": "FORBIDDEN"}

		case errors.Is(origErr, domain.ErrConflict):
			gqlErr.Extensions = map[string]interface{}{"code": "CONFLICT"}

		default:
			// Unexpected error - log it, return generic message to client
			requestID := ctxutil.RequestIDFromCtx(ctx)
			log.ErrorContext(ctx, "unexpected GraphQL error",
				slog.String("error", origErr.Error()),
				slog.String("request_id", requestID),
			)
			gqlErr.Message = "internal error"
			gqlErr.Extensions = map[string]interface{}{"code": "INTERNAL"}
		}

		return gqlErr
	}
}
