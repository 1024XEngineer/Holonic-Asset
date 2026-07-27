// Main entry point for the application
package main

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	appservice "github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func main() {
	projectDao := dao.NewMemoryProjectDao()
	projectRepository := repository.NewProjectRepository(projectDao)
	projectService := appservice.NewProjectService(projectRepository)
	projectHandler := handler.NewProjectHandler(projectService)

	generationService := appservice.NewGenerationService(nil, nil, nil)
	generationHandler := handler.NewGenerationHandler(generationService)

	mediaService := appservice.NewMediaService()
	mediaHandler := handler.NewMediaHandler(mediaService)

	taxonomyService := appservice.NewAssetDiscoveryService()
	taxonomyHandler := handler.NewTaxonomyHandler(taxonomyService)

	e := router.Register(nil, projectHandler, generationHandler, mediaHandler, taxonomyHandler)
	e.Logger.Fatal(e.Start(":8080"))
}
