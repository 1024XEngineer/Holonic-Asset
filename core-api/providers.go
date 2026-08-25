package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
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

func InitUserStore(db *gorm.DB) auth.Store {
	return repository.NewUserRepository(dao.NewGormUserDao(db))
}

func InitAuthService(cfg config.AuthConfig, store auth.Store) (*auth.Service, error) {
	return auth.NewService(store, cfg.JWTSecret, cfg.TokenExpiry)
}

// InitImageService creates the external image provider and its application service.
func InitImageService(cfg config.ImageClientConfig, appLogger logger.Logger) imageclient.ImageGenerationService {
	models := make([]imageclient.ModelConfig, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		models = append(models, imageclient.ModelConfig{
			Name:     model.Name,
			Protocol: model.Protocol,
		})
	}
	provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL:       cfg.BaseURL,
		APIKey:        cfg.APIKey,
		DefaultModel:  cfg.DefaultModel,
		FallbackModel: cfg.FallbackModel,
		Provider:      cfg.Provider,
		Models:        models,
		Logger:        appLogger,
	})
	return imageclient.NewImageGenerationService(provider)
}

// InitLLMService creates the external multimodal provider and its application service.
func InitLLMService(cfg config.LLMClientConfig, appLogger ...logger.Logger) llmclient.LLMService {
	var providerLogger logger.Logger
	if len(appLogger) > 0 {
		providerLogger = appLogger[0]
	}
	models := make([]llmclient.ModelConfig, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		models = append(models, llmclient.ModelConfig{
			Name:     model.Name,
			Protocol: model.Protocol,
		})
	}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		DefaultModel: cfg.DefaultModel,
		Models:       models,
		Logger:       providerLogger,
	})
	return llmclient.NewLLMService(provider)
}

// InitVideoService creates the QNA video provider and its application service.
func InitVideoService(cfg config.VideoClientConfig, appLogger logger.Logger) videoclient.VideoGenerationService {
	models := make([]videoclient.ModelConfig, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		models = append(models, videoclient.ModelConfig{
			Name:     model.Name,
			Protocol: model.Protocol,
		})
	}
	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Models:       models,
		PollInterval: cfg.PollInterval,
		PollTimeout:  cfg.PollTimeout,
		MaxRetries:   cfg.MaxRetries,
		RetryDelay:   cfg.RetryDelay,
		Logger:       appLogger,
	})
	return videoclient.NewVideoGenerationService(provider)
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
	llm llmclient.LLMService,
	processor imageprocessor.Processor,
	assets generator.AssetWriter,
) generator.Executor {
	return generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{
		LLM: llm,
	})
}

// InitGeneratorEngine creates the generator module and registers its task handlers.
func InitGeneratorEngine(tasks task.Manager, executor generator.Executor) *generator.Engine {
	return generator.NewEngine(tasks, executor)
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
	references ...upload.ReferenceResolver,
) HTTPHandlers {
	var resolver upload.ReferenceResolver
	if len(references) > 0 {
		resolver = references[0]
	}
	return HTTPHandlers{
		Asset:      handler.NewHandler(workspaceModule.Assets, resolver),
		Project:    handler.NewProjectHandler(workspaceModule.Projects, resolver),
		Generation: handler.NewGenerationHandler(generatorEngine, resolver),
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
