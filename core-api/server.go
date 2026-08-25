package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/middleware"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/export"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/pprof"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

const defaultShutdownTimeout = 10 * time.Second

// App owns the runtime components that have an explicit lifecycle.
type App struct {
	engine *echo.Echo
	tasks  task.Manager
	db     *gorm.DB
	logger logger.Logger
	pprof  *pprof.Server

	lifecycleMu sync.Mutex
	started     bool
	stopped     bool

	shutdownOnce sync.Once
	shutdownErr  error
}

// InitServer is the application injector. It mirrors a Wire-generated injector:
// load configuration first, then assemble the dependency graph explicitly.
func InitServer(path string) (*App, error) {
	cfg, err := LoadAppConfig(path)
	if err != nil {
		return nil, err
	}
	return InitServerFromConfig(context.Background(), cfg)
}

// newApp and newAppWithServices provide a lightweight composition root for
// tests and callers that only need the HTTP dependency graph.
func newApp(
	projectDao dao.ProjectDao,
	assetStore assetdomain.Store,
	imageService imageclient.ImageGenerationService,
) *App {
	return newAppWithServices(projectDao, assetStore, imageService, nil)
}

func newAppWithServices(
	projectDao dao.ProjectDao,
	assetStore assetdomain.Store,
	imageService imageclient.ImageGenerationService,
	uploadStore upload.Store,
) *App {
	var references upload.ReferenceStore
	if candidate, ok := uploadStore.(upload.ReferenceStore); ok {
		references = candidate
	}
	projectRepository := repository.NewProjectRepository(projectDao)
	workspaceModule := workspace.New(projectRepository, assetStore, imageService, references)
	projectHandler := handler.NewProjectHandler(workspaceModule.Projects, references)

	var assetRouter router.AssetRouter
	if workspaceModule.Assets != nil {
		assetRouter = handler.NewHandler(workspaceModule.Assets, references)
	}

	generationHandler := handler.NewGenerationHandler(generator.NewEngine(nil, nil), references)
	uploadHandler := handler.NewUploadHandler(upload.NewManager(uploadStore))

	return &App{
		engine: router.Register(assetRouter, projectHandler, generationHandler, uploadHandler),
	}
}

// InitServerFromConfig assembles dependencies in infrastructure-to-transport order.
func InitServerFromConfig(ctx context.Context, cfg config.Config) (*App, error) {
	if ctx == nil {
		return nil, errors.New("app: initialization context is required")
	}
	if len(cfg.HTTP.AllowedOrigins) == 0 {
		return nil, errors.New("app: at least one allowed HTTP origin is required")
	}

	// Infrastructure.
	appLogger, err := InitLogger(&cfg.Log)
	if err != nil {
		return nil, err
	}
	db, err := InitDB(ctx, &cfg.DB, appLogger)
	if err != nil {
		_ = appLogger.Sync()
		return nil, err
	}

	// Repositories and external services.
	userStore := InitUserStore(db)
	authService, err := InitAuthService(cfg.Auth, userStore)
	if err != nil {
		cleanupInitialization(db, appLogger)
		return nil, err
	}
	projectStore := InitProjectStore(db)
	assetStore := InitAssetStore(db)
	taskStore := InitTaskStore(db)
	imageService := InitImageService(cfg.Image, appLogger)
	llmService := InitLLMService(cfg.LLM, appLogger)
	uploadStore, err := InitUploadStore(cfg.QiNiu)
	if err != nil {
		cleanupInitialization(db, appLogger)
		return nil, err
	}
	var references upload.ReferenceStore
	if candidate, ok := uploadStore.(upload.ReferenceStore); ok {
		references = candidate
	}
	var resources upload.ResourceStore
	if candidate, ok := uploadStore.(upload.ResourceStore); ok {
		resources = candidate
	}
	artifactStore, ok := uploadStore.(upload.ArtifactStore)
	if !ok {
		cleanupInitialization(db, appLogger)
		return nil, errors.New("app: artifact storage is not configured")
	}
	taskManager, err := InitTask(ctx, cfg.Queue, taskStore)
	if err != nil {
		cleanupInitialization(db, appLogger)
		return nil, err
	}

	// Business modules.
	workspaceModule := workspace.New(projectStore, assetStore, imageService, references)
	videos := InitVideoService(cfg.Video, appLogger)
	imageProcessor := InitImageProcessor()
	animations := generator.NewAnimationGenerationServiceWithDependencies(
		videos,
		imageProcessor,
		generator.AnimationGenerationDependencies{
			ReferenceResolver: references,
			Logger:            appLogger,
		},
	)
	generatorExecutor := generator.NewExecutorWithDependencies(
		imageService,
		imageProcessor,
		workspaceModule.Assets,
		generator.ExecutorDependencies{
			References: references,
			Projects:   workspaceModule.Projects,
			Resources:  resources,
			LLM:        llmService,
			Animations: animations,
			Logger:     appLogger,
		},
	)
	generatorEngine := generator.NewEngine(taskManager, generatorExecutor, generator.EngineDependencies{
		Projects:   workspaceModule.Projects,
		Assets:     workspaceModule.Assets,
		References: references,
	})
	exportManager := export.NewManager(workspaceModule.Assets, references, artifactStore, taskManager)
	taskManager.Register(string(export.TaskType), exportManager)

	// Transport.
	assetHandler := handler.NewHandler(workspaceModule.Assets, references)
	projectHandler := handler.NewProjectHandler(workspaceModule.Projects, references)
	generationHandler := handler.NewGenerationHandler(generatorEngine, references)
	uploadHandler := handler.NewUploadHandler(upload.NewManager(uploadStore))
	exportHandler := handler.NewExportHandler(exportManager)
	authHandler := handler.NewAuthHandler(authService)
	httpEngine := router.RegisterWithExport(
		assetHandler,
		projectHandler,
		generationHandler,
		uploadHandler,
		exportHandler,
		router.Authentication{
			Router:         authHandler,
			Middleware:     middleware.JWT(authService),
			AllowedOrigins: cfg.HTTP.AllowedOrigins,
		},
	)

	app := NewApp(httpEngine, taskManager, db, appLogger)
	if cfg.Pprof.Enabled {
		app.pprof = pprof.New()
	}
	appLogger.Info("application initialized")
	return app, nil
}

// LoadAppConfig loads one explicit configuration file.
func LoadAppConfig(path string) (config.Config, error) {
	var cfg config.Config
	if err := viperx.LoadConfig(path, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("app: load config: %w", err)
	}
	if secret, ok := os.LookupEnv("HOLONIC_AUTH_JWT_SECRET"); ok {
		cfg.Auth.JWTSecret = secret
	}
	return cfg, nil
}

// NewApp creates the runtime application from fully initialized dependencies.
func NewApp(engine *echo.Echo, tasks task.Manager, db *gorm.DB, appLogger logger.Logger) *App {
	return &App{
		engine: engine,
		tasks:  tasks,
		db:     db,
		logger: appLogger,
	}
}

// Start starts background workers and the HTTP server, then blocks until the
// server exits or the supplied context is cancelled.
func (a *App) Start(ctx context.Context, address string) error {
	if a == nil {
		return errors.New("app: application is nil")
	}
	if ctx == nil {
		return errors.New("app: start context is required")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("app: start context: %w", err)
	}
	if a.engine == nil {
		return errors.New("app: HTTP engine is required")
	}
	if strings.TrimSpace(address) == "" {
		return errors.New("app: HTTP address is required")
	}

	a.lifecycleMu.Lock()
	if a.stopped {
		a.lifecycleMu.Unlock()
		return errors.New("app: application is already stopped")
	}
	if a.started {
		a.lifecycleMu.Unlock()
		return errors.New("app: application is already started")
	}
	a.started = true
	a.lifecycleMu.Unlock()

	if a.pprof != nil {
		if err := a.pprof.Start(ctx); err != nil {
			return fmt.Errorf("app: start pprof server: %w", err)
		}
	}

	if a.tasks != nil {
		if err := a.tasks.Start(ctx); err != nil {
			if a.pprof != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
				defer cancel()
				_ = a.pprof.Shutdown(shutdownCtx)
			}
			return fmt.Errorf("app: start task manager: %w", err)
		}
	}

	if a.logger != nil {
		a.logger.Info("application starting", logger.String("address", address))
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.engine.Start(address)
	}()

	select {
	case err := <-serverErr:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("app: start HTTP server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		return a.Shutdown(shutdownCtx)
	}
}

// Shutdown stops runtime components in reverse initialization order.
func (a *App) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	a.shutdownOnce.Do(func() {
		var shutdownErrors []error
		a.lifecycleMu.Lock()
		a.stopped = true
		a.lifecycleMu.Unlock()

		if a.engine != nil {
			if err := a.engine.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("app: shutdown HTTP server: %w", err))
			}
		}

		if a.pprof != nil {
			if err := a.pprof.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("app: shutdown pprof server: %w", err))
			}
		}

		if a.tasks != nil {
			if err := a.tasks.Stop(); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("app: stop task manager: %w", err))
			}
		}

		if err := closeDatabase(a.db); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}

		if a.logger != nil {
			a.logger.Info("application stopped")
			if err := a.logger.Sync(); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("app: sync logger: %w", err))
			}
		}

		a.shutdownErr = errors.Join(shutdownErrors...)
	})

	return a.shutdownErr
}

func cleanupInitialization(db *gorm.DB, appLogger logger.Logger) {
	if err := closeDatabase(db); err != nil && appLogger != nil {
		appLogger.Error("database cleanup failed", logger.Error(err))
	}
	if appLogger != nil {
		_ = appLogger.Sync()
	}
}

func closeDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("app: get database for shutdown: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("app: close database: %w", err)
	}
	return nil
}
