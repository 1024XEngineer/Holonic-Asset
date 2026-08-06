package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
)

const defaultShutdownTimeout = 10 * time.Second

// App owns the runtime components that have an explicit lifecycle.
type App struct {
	engine *echo.Echo
	tasks  task.Manager
	db     *gorm.DB
	logger logger.Logger

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

// InitServerFromConfig assembles dependencies in infrastructure-to-transport order.
func InitServerFromConfig(ctx context.Context, cfg config.Config) (*App, error) {
	if ctx == nil {
		return nil, errors.New("app: initialization context is required")
	}

	// infrastructure
	appLogger, err := InitLogger(&cfg.Log)
	if err != nil {
		return nil, err
	}
	db, err := InitDB(ctx, &cfg.DB, appLogger)
	if err != nil {
		_ = appLogger.Sync()
		return nil, err
	}

	// repository / DAO
	projectStore := InitProjectStore(db)
	assetStore := InitAssetStore(db)
	taskStore := InitTaskStore(db)

	// external clients / queue / outbox
	imageService := InitImageService(cfg.QNA)
	uploadStore, err := InitUploadStore(cfg.QiNiu)
	if err != nil {
		cleanupInitialization(db, appLogger)
		return nil, err
	}
	taskManager, err := InitTask(ctx, cfg.Queue, taskStore)
	if err != nil {
		cleanupInitialization(db, appLogger)
		return nil, err
	}

	// business modules
	workspaceModule := InitWorkspace(projectStore, assetStore, imageService)
	imageProcessor := InitImageProcessor()
	generatorExecutor := InitGeneratorExecutor(imageService, imageProcessor, workspaceModule.Assets)
	generatorEngine := InitGeneratorEngine(taskManager, generatorExecutor)
	uploadManager := InitUploadManager(uploadStore)

	// handlers
	handlers := InitHandlers(workspaceModule, generatorEngine, uploadManager)

	// router
	httpEngine := InitRouter(handlers)

	// application
	app := NewApp(httpEngine, taskManager, db, appLogger)
	appLogger.Info("application initialized")
	return app, nil
}

// LoadAppConfig loads one explicit configuration file.
func LoadAppConfig(path string) (config.Config, error) {
	var cfg config.Config
	if err := viperx.LoadConfig(path, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("app: load config: %w", err)
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

	if a.tasks != nil {
		if err := a.tasks.Start(ctx); err != nil {
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
