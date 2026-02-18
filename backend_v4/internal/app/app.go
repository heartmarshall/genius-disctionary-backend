package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit"
	authmethodrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/authmethod"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	inboxrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/sense"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/session"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/token"
	topicrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	userrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/user"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/provider/freedict"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/provider/google"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/provider/translate"
	authpkg "github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	authsvc "github.com/heartmarshall/myenglish-backend/internal/service/auth"
	"github.com/heartmarshall/myenglish-backend/internal/service/content"
	"github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
	inboxsvc "github.com/heartmarshall/myenglish-backend/internal/service/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/service/refcatalog"
	"github.com/heartmarshall/myenglish-backend/internal/service/study"
	topicsvc "github.com/heartmarshall/myenglish-backend/internal/service/topic"
	usersvc "github.com/heartmarshall/myenglish-backend/internal/service/user"
	gqlpkg "github.com/heartmarshall/myenglish-backend/internal/transport/graphql"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/dataloader"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/resolver"
	"github.com/heartmarshall/myenglish-backend/internal/transport/middleware"
	"github.com/heartmarshall/myenglish-backend/internal/transport/rest"
)

// Run is the application entry point. It loads configuration, initializes
// all layers (repos, services, transport), starts the HTTP server, and
// waits for a shutdown signal for graceful termination.
func Run(ctx context.Context) error {
	// -----------------------------------------------------------------------
	// 1. Load and validate config
	// -----------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// -----------------------------------------------------------------------
	// 2. Initialize logger
	// -----------------------------------------------------------------------
	logger := NewLogger(cfg.Log)

	logger.Info("starting application",
		slog.String("version", BuildVersion()),
		slog.String("log_level", cfg.Log.Level),
	)

	// -----------------------------------------------------------------------
	// 3. Connect to DB (pool)
	// -----------------------------------------------------------------------
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	logger.Info("database connected",
		slog.Int("max_conns", int(cfg.Database.MaxConns)),
	)

	// -----------------------------------------------------------------------
	// 4. Create TxManager
	// -----------------------------------------------------------------------
	txm := postgres.NewTxManager(pool)

	// -----------------------------------------------------------------------
	// 5. Create repositories (15 packages)
	// -----------------------------------------------------------------------
	auditRepo := audit.New(pool)
	authMethodRepo := authmethodrepo.New(pool)
	cardRepo := card.New(pool)
	entryRepo := entry.New(pool)
	exampleRepo := example.New(pool, txm)
	imageRepo := image.New(pool)
	inboxRepo := inboxrepo.New(pool)
	pronunciationRepo := pronunciation.New(pool)
	refentryRepo := refentry.New(pool, txm)
	reviewlogRepo := reviewlog.New(pool)
	senseRepo := sense.New(pool, txm)
	sessionRepo := session.New(pool)
	tokenRepo := token.New(pool)
	topicRepo := topicrepo.New(pool)
	translationRepo := translation.New(pool, txm)
	userRepo := userrepo.New(pool)

	// -----------------------------------------------------------------------
	// 6. Create external providers
	// -----------------------------------------------------------------------
	dictProvider := freedict.NewProvider(logger)
	transProvider := translate.NewStub()

	// -----------------------------------------------------------------------
	// 7. Create JWT manager + OAuth verifier
	// -----------------------------------------------------------------------
	jwtManager := authpkg.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTIssuer,
		cfg.Auth.AccessTokenTTL,
	)

	oauthVerifier := google.NewVerifier(
		cfg.Auth.GoogleClientID,
		cfg.Auth.GoogleClientSecret,
		cfg.Auth.GoogleRedirectURI,
		logger,
	)

	// -----------------------------------------------------------------------
	// 8. Create services (8 packages)
	// -----------------------------------------------------------------------
	authService := authsvc.NewService(
		logger, userRepo, userRepo, tokenRepo, authMethodRepo, txm, oauthVerifier, jwtManager, cfg.Auth,
	)

	userService := usersvc.NewService(
		logger, userRepo, userRepo, auditRepo, txm,
	)

	refCatalogService := refcatalog.NewService(
		logger, refentryRepo, txm, dictProvider, transProvider,
	)

	srsConfig := domain.SRSConfig{
		DefaultEaseFactor:    cfg.SRS.DefaultEaseFactor,
		MinEaseFactor:        cfg.SRS.MinEaseFactor,
		MaxIntervalDays:      cfg.SRS.MaxIntervalDays,
		GraduatingInterval:   cfg.SRS.GraduatingInterval,
		LearningSteps:        cfg.SRS.LearningSteps,
		NewCardsPerDay:       cfg.SRS.NewCardsPerDay,
		ReviewsPerDay:        cfg.SRS.ReviewsPerDay,
		EasyInterval:         cfg.SRS.EasyInterval,
		RelearningSteps:      cfg.SRS.RelearningSteps,
		IntervalModifier:     cfg.SRS.IntervalModifier,
		HardIntervalModifier: cfg.SRS.HardIntervalModifier,
		EasyBonus:            cfg.SRS.EasyBonus,
		LapseNewInterval:     cfg.SRS.LapseNewInterval,
		UndoWindowMinutes:    cfg.SRS.UndoWindowMinutes,
	}

	dictionaryService := dictionary.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		pronunciationRepo, imageRepo, cardRepo, auditRepo, txm,
		refCatalogService, cfg.Dictionary,
	)

	contentService := content.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		imageRepo, auditRepo, txm,
	)

	studyService := study.NewService(
		logger, cardRepo, reviewlogRepo, sessionRepo, entryRepo,
		senseRepo, userRepo, auditRepo, txm, srsConfig,
	)

	topicService := topicsvc.NewService(
		logger, topicRepo, entryRepo, auditRepo, txm,
	)

	inboxService := inboxsvc.NewService(
		logger, inboxRepo,
	)

	// -----------------------------------------------------------------------
	// 9. Create GraphQL resolver + handler
	// -----------------------------------------------------------------------
	res := resolver.NewResolver(
		logger, dictionaryService, contentService, studyService,
		topicService, inboxService, userService,
	)

	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers: res,
	})

	gqlSrv := handler.NewDefaultServer(schema)
	gqlSrv.SetErrorPresenter(gqlpkg.NewErrorPresenter(logger))

	// -----------------------------------------------------------------------
	// 10. Create DataLoader middleware
	// -----------------------------------------------------------------------
	dlRepos := &dataloader.Repos{
		Sense:         senseRepo,
		Translation:   translationRepo,
		Example:       exampleRepo,
		Pronunciation: pronunciationRepo,
		Image:         imageRepo,
		Card:          cardRepo,
		Topic:         topicRepo,
		ReviewLog:     reviewlogRepo,
	}

	// -----------------------------------------------------------------------
	// 11. Create Health + Auth handlers
	// -----------------------------------------------------------------------
	healthHandler := rest.NewHealthHandler(pool, BuildVersion())
	authHandler := rest.NewAuthHandler(authService, logger)

	// -----------------------------------------------------------------------
	// 12. Assemble middleware chain
	// -----------------------------------------------------------------------
	graphqlHandler := middleware.Chain(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.CORS(cfg.CORS),
		middleware.Auth(authService),
		middleware.Middleware(dataloader.Middleware(dlRepos)),
	)(gqlSrv)

	// -----------------------------------------------------------------------
	// 13. Create ServeMux and register routes
	// -----------------------------------------------------------------------
	mux := http.NewServeMux()

	// Health endpoints - outside middleware stack
	mux.HandleFunc("GET /live", healthHandler.Live)
	mux.HandleFunc("GET /ready", healthHandler.Ready)
	mux.HandleFunc("GET /health", healthHandler.Health)

	// Auth endpoints - CORS only (no auth middleware)
	authCORS := middleware.CORS(cfg.CORS)
	mux.Handle("POST /auth/register", authCORS(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /auth/login", authCORS(http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /auth/login/password", authCORS(http.HandlerFunc(authHandler.LoginWithPassword)))
	mux.Handle("POST /auth/refresh", authCORS(http.HandlerFunc(authHandler.Refresh)))
	mux.Handle("POST /auth/logout", authCORS(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("OPTIONS /auth/{path...}", authCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	// GraphQL - full middleware chain
	mux.Handle("POST /query", graphqlHandler)
	mux.Handle("OPTIONS /query", graphqlHandler)

	// Playground (conditional)
	if cfg.GraphQL.PlaygroundEnabled {
		mux.Handle("GET /", playground.Handler("MyEnglish GraphQL", "/query"))
		logger.Info("GraphQL playground enabled", slog.String("path", "/"))
	}

	// -----------------------------------------------------------------------
	// 14. Create and start HTTP server
	// -----------------------------------------------------------------------
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Info("HTTP server started", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", slog.String("error", err.Error()))
		}
	}()

	// -----------------------------------------------------------------------
	// 15. Wait for signal -> graceful shutdown
	// -----------------------------------------------------------------------
	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	logger.Info("HTTP server stopped")

	// pool.Close() called via defer
	logger.Info("shutdown complete")

	return nil
}
