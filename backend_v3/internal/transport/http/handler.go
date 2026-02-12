package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/heartmarshall/my-english/graph"
	"github.com/heartmarshall/my-english/internal/config"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/service"
	transportErrors "github.com/heartmarshall/my-english/internal/transport"
	"github.com/heartmarshall/my-english/internal/transport/dataloader"
	"github.com/heartmarshall/my-english/internal/transport/middleware"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	// DefaultAPQCacheSize is the default cache size for Automatic Persisted Queries
	DefaultAPQCacheSize = 100
)

// HandlerConfig содержит конфигурацию для создания HTTP handler.
type HandlerConfig struct {
	Config         *config.Config
	Logger         *slog.Logger
	Services       *service.Services
	Repos          *repository.Registry
	CORSOrigins    []string
	CORSMethods    []string
	CORSHeaders    []string
	EnableCORS     bool
	EnableSecurity bool
}

// NewHandler создает и настраивает основной HTTP хендлер приложения.
func NewHandler(
	cfg *config.Config,
	logger *slog.Logger,
	services *service.Services,
	repos *repository.Registry,
) http.Handler {
	return NewHandlerWithConfig(HandlerConfig{
		Config:         cfg,
		Logger:         logger,
		Services:       services,
		Repos:          repos,
		EnableCORS:     true,
		EnableSecurity: true,
	})
}

// NewHandlerWithConfig создает HTTP handler с расширенной конфигурацией.
func NewHandlerWithConfig(cfg HandlerConfig) http.Handler {
	// 1. Инициализация обработчика ошибок
	devMode := cfg.Config.Log.Level == "debug"
	errorHandler := transportErrors.NewErrorHandler(cfg.Logger, devMode)
	transportErrors.SetDefaultErrorHandler(errorHandler)

	// 2. Инициализация GraphQL схемы
	gqlConfig := graph.Config{
		Resolvers: &graph.Resolver{
			Services: cfg.Services,
		},
	}

	// Создаем сервер gqlgen
	srv := handler.New(graph.NewExecutableSchema(gqlConfig))

	// 3. Настройка транспортов GraphQL
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// 4. Настройка кэширования запросов и сложности
	srv.SetQueryCache(lru.New[*ast.QueryDocument](cfg.Config.GraphQL.QueryCacheSize))

	// Интроспекция только если включена
	if cfg.Config.GraphQL.EnableIntrospection {
		srv.Use(extension.Introspection{})
	}

	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](DefaultAPQCacheSize),
	})

	// 5. Настройка обработки ошибок GraphQL
	srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		// Если ошибка уже является GraphQL ошибкой, возвращаем её как есть
		if gqlErr, ok := err.(*gqlerror.Error); ok {
			return gqlErr
		}
		// Используем наш обработчик ошибок, который теперь типобезопасно возвращает *gqlerror.Error
		return errorHandler.HandleError(ctx, err)
	})

	// 6. Настройка роутера
	mux := http.NewServeMux()

	// GraphQL Playground (только если включен)
	if cfg.Config.GraphQL.EnablePlayground {
		mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}

	// Основной GraphQL Endpoint
	mux.Handle("/query", srv)

	// 7. Подключение Middleware (порядок важен!)
	// Middleware применяются в обратном порядке (последний в коде выполняется первым)
	var handler http.Handler = mux

	// Внедряем DataLoaders (обязательно перед GraphQL хендлером)
	loaderConfig := dataloader.DefaultLoaderConfig()
	loaderConfig.Logger = cfg.Logger
	handler = dataloader.MiddlewareWithConfig(cfg.Repos, loaderConfig)(handler)

	// Request ID (должен быть первым, чтобы другие middleware могли его использовать)
	handler = middleware.RequestIDMiddleware()(handler)

	// Security headers (применяются ко всем запросам)
	if cfg.EnableSecurity {
		handler = middleware.SecurityHeadersMiddleware()(handler)
	}

	// CORS (должен быть перед другими middleware, которые могут писать заголовки)
	if cfg.EnableCORS {
		handler = middleware.CORSMiddleware(cfg.CORSOrigins, cfg.CORSMethods, cfg.CORSHeaders)(handler)
	}

	// Timeout (должен быть перед логированием, чтобы таймауты логировались)
	handler = middleware.TimeoutMiddleware(cfg.Config.Server.RequestTimeout)(handler)

	// Logging (логирует все запросы)
	handler = middleware.LoggingMiddleware(cfg.Logger)(handler)

	// Recovery (должен быть последним, чтобы перехватывать все паники)
	handler = middleware.RecoveryMiddleware(cfg.Logger)(handler)

	return handler
}
