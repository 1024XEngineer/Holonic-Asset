package main

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	appservice "github.com/1024XEngineer/Holonic-Asset/internal/service"
)

type App struct {
	engine *echo.Echo
}

func InitServer() *App {
	projectDao := dao.NewMemoryProjectDao()
	projectRepository := repository.NewProjectRepository(projectDao)
	projectService := appservice.NewProjectService(projectRepository)
	projectHandler := handler.NewProjectHandler(projectService)

	generatorEngine := generator.NewEngine(nil, nil, nil, nil)
	generationHandler := handler.NewGenerationHandler(generatorEngine)

	uploader := upload.NewUploader(nil)
	uploadHandler := handler.NewUploadHandler(uploader)

	return &App{
		engine: router.Register(nil, projectHandler, generationHandler, uploadHandler),
	}
}

func (a *App) Start(address string) error {
	return a.engine.Start(address)
}
