package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

// InitProjectStore wires the project DAO to its repository adapter.
func InitProjectStore(db *gorm.DB) projectdomain.Store {
	return repository.NewProjectRepository(dao.NewGormProjectDao(db))
}

// InitAssetStore wires all asset DAOs to the transactional asset repository.
func InitAssetStore(db *gorm.DB) assetdomain.Store {
	return repository.NewAssetRepositoryWithDB(
		db,
		&dao.AssetDaoImpl{DB: db},
		&dao.AssetContentDaoImpl{DB: db},
		&dao.AssetRecordDaoImpl{DB: db},
	)
}

// InitTaskStore creates the persistence adapter used by the task module.
func InitTaskStore(db *gorm.DB) task.TaskStore {
	return repository.NewTaskRepository(db)
}

// InitImageService creates the external image provider and its application service.
func InitImageService(cfg config.QNAConfig) imageclient.ImageGenerationService {
	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		DefaultModel: cfg.DefaultModel,
	})
	return imageclient.NewImageGenerationService(provider)
}

// InitUploadStore creates the configured object storage adapter.
func InitUploadStore(cfg config.QiniuConfig) (upload.Store, error) {
	store, err := upload.NewQiniuStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("app: initialize upload storage: %w", err)
	}
	return store, nil
}

// InitWorkspace creates the project and asset business module.
func InitWorkspace(
	projectStore projectdomain.Store,
	assetStore assetdomain.Store,
	images imageclient.ImageGenerationService,
) *workspace.Workspace {
	return workspace.New(projectStore, assetStore, images)
}

// InitImageProcessor creates the deterministic image-processing service.
func InitImageProcessor() imageprocessor.Processor {
	return imageprocessor.NewProcessor()
}

// InitGeneratorExecutor creates the generation workflow executor.
func InitGeneratorExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets generator.AssetWriter,
) generator.Executor {
	return generator.NewExecutor(images, processor, assets)
}

// InitGeneratorEngine creates the generator module and registers its task handlers.
func InitGeneratorEngine(tasks task.Manager, executor generator.Executor) *generator.Engine {
	return generator.NewEngine(tasks, nil, executor)
}

// InitUploadManager creates the upload business module.
func InitUploadManager(store upload.Store) upload.Manager {
	return upload.NewManager(store)
}

// HTTPHandlers groups transport providers without hiding their dependencies.
type HTTPHandlers struct {
	Asset      router.AssetRouter
	Project    router.ProjectRouter
	Generation router.GenerationRouter
	Upload     router.UploadRouter
}

// InitHandlers creates all HTTP handlers from initialized business modules.
func InitHandlers(
	workspaceModule *workspace.Workspace,
	generatorEngine generator.RunManager,
	uploadManager upload.Manager,
) HTTPHandlers {
	return HTTPHandlers{
		Asset:      handler.NewHandler(workspaceModule.Assets),
		Project:    handler.NewProjectHandler(workspaceModule.Projects),
		Generation: handler.NewGenerationHandler(generatorEngine),
		Upload:     handler.NewUploadHandler(uploadManager),
	}
}

// InitRouter registers all application routes on a new Echo engine.
func InitRouter(handlers HTTPHandlers) *echo.Echo {
	return router.Register(
		handlers.Asset,
		handlers.Project,
		handlers.Generation,
		handlers.Upload,
	)
}
