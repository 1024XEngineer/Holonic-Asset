package main

import (
	"context"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

const defaultConfigPath = "internal/config/config.yaml"
const configPathEnv = "HOLONIC_ASSET_CONFIG"

type App struct {
	engine *echo.Echo
	tasks  task.Manager
	db     *gorm.DB
}

func InitServer() (*App, error) {
	cfg, err := LoadAppConfig()
	if err != nil {
		return nil, err
	}
	return InitServerFromConfig(context.Background(), cfg)
}

func resolveConfigPath() string {
	if path := os.Getenv(configPathEnv); path != "" {
		return path
	}
	return defaultConfigPath
}

func NewApp(projectDao dao.ProjectDao) *App {
	return newApp(projectDao, nil)
}

func newApp(projectDao dao.ProjectDao, assetStore assetdomain.Store) *App {
	projectRepository := repository.NewProjectRepository(projectDao)
	workspaceModule := workspace.New(projectRepository, assetStore)
	projectHandler := handler.NewProjectHandler(workspaceModule.Projects)
	var assetRouter router.AssetRouter
	if workspaceModule.Assets != nil {
		assetRouter = handler.NewHandler(workspaceModule.Assets)
	}

	generatorEngine := generator.NewEngine(nil, nil, nil)
	generationHandler := handler.NewGenerationHandler(generatorEngine)

	uploadManager := upload.NewManager(nil)
	uploadHandler := handler.NewUploadHandler(uploadManager)

	return &App{
		engine: router.Register(assetRouter, projectHandler, generationHandler, uploadHandler),
	}
}

func InitServerFromConfig(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := InitDB(ctx, &cfg.DB, nil)
	if err != nil {
		return nil, err
	}

	assetRepository := repository.NewAssetRepositoryWithDB(
		db,
		&dao.AssetDaoImpl{DB: db},
		&dao.AssetContentDaoImpl{DB: db},
		&dao.AssetRecordDaoImpl{DB: db},
	)
	projectRepository := repository.NewProjectRepository(dao.NewGormProjectDao(db))
	workspaceModule := workspace.New(projectRepository, assetRepository)

	uploadStore, err := upload.NewQiniuStorage(cfg.QiNiu)
	if err != nil {
		closeDatabase(db)
		return nil, fmt.Errorf("app: initialize upload storage: %w", err)
	}

	taskRepository := repository.NewTaskRepository(db)
	taskManager, err := InitTask(ctx, cfg.Queue, taskRepository)
	if err != nil {
		closeDatabase(db)
		return nil, err
	}

	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      cfg.QNA.BaseURL,
		APIKey:       cfg.QNA.APIKey,
		DefaultModel: cfg.QNA.DefaultModel,
	})
	images := imageclient.NewImageGenerationService(provider)
	processor := imageprocessor.NewProcessor()
	executor := generator.NewExecutor(images, processor, workspaceModule.Assets)
	generatorEngine := generator.NewEngine(taskManager, nil, executor)

	assetHandler := handler.NewHandler(workspaceModule.Assets)
	projectHandler := handler.NewProjectHandler(workspaceModule.Projects)
	generationHandler := handler.NewGenerationHandler(generatorEngine)
	uploadManager := upload.NewManager(uploadStore)
	uploadHandler := handler.NewUploadHandler(uploadManager)

	return &App{
		engine: router.Register(assetHandler, projectHandler, generationHandler, uploadHandler),
		tasks:  taskManager,
		db:     db,
	}, nil
}

func LoadAppConfig() (config.Config, error) {
	var cfg config.Config
	if err := viperx.LoadConfig(resolveConfigPath(), &cfg); err != nil {
		return config.Config{}, fmt.Errorf("app: load config: %w", err)
	}
	return cfg, nil
}

func (a *App) Start(address string) error {
	if a.tasks != nil {
		if err := a.tasks.Start(context.Background()); err != nil {
			return err
		}
	}
	return a.engine.Start(address)
}

func closeDatabase(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
