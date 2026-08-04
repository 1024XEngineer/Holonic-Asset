package main

import (
	"context"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

const defaultConfigPath = "internal/config/config.example.yaml"
const configPathEnv = "HOLONIC_ASSET_CONFIG"

type App struct {
	engine *echo.Echo
}

func InitServer() (*App, error) {
	var cfg config.Config
	if err := viperx.LoadConfig(resolveConfigPath(), &cfg); err != nil {
		return nil, fmt.Errorf("app: load config: %w", err)
	}

	db, err := InitDB(context.Background(), &cfg.DB, nil)
	if err != nil {
		return nil, err
	}

	assetRepository := repository.NewAssetRepositoryWithDB(
		db,
		&dao.AssetDaoImpl{DB: db},
		&dao.AssetContentDaoImpl{DB: db},
		&dao.AssetRecordDaoImpl{DB: db},
	)
	imageProvider := imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      cfg.QNA.BaseURL,
		APIKey:       cfg.QNA.APIKey,
		DefaultModel: cfg.QNA.DefaultModel,
	})
	imageService := imageclient.NewImageGenerationService(imageProvider)

	return newApp(dao.NewGormProjectDao(db), assetRepository, imageService), nil
}

func resolveConfigPath() string {
	if path := os.Getenv(configPathEnv); path != "" {
		return path
	}
	return defaultConfigPath
}

func NewApp(projectDao dao.ProjectDao) *App {
	return newApp(projectDao, nil, nil)
}

func newApp(
	projectDao dao.ProjectDao,
	assetStore assetdomain.Store,
	imageService imageclient.ImageGenerationService,
) *App {
	projectRepository := repository.NewProjectRepository(projectDao)
	workspaceModule := workspace.New(projectRepository, assetStore, imageService)
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

func (a *App) Start(address string) error {
	return a.engine.Start(address)
}
