// Main entry point for the application
package main

import (
	"github.com/1024XEngineer/Holonic-Asset/internal"
	generationhandler "github.com/1024XEngineer/Holonic-Asset/internal/generation/handler"
	generationservice "github.com/1024XEngineer/Holonic-Asset/internal/generation/service"
	mediahandler "github.com/1024XEngineer/Holonic-Asset/internal/media/handler"
	mediaservice "github.com/1024XEngineer/Holonic-Asset/internal/media/service"
	projecthandler "github.com/1024XEngineer/Holonic-Asset/internal/project/handler"
	projectrepository "github.com/1024XEngineer/Holonic-Asset/internal/project/repository"
	projectdao "github.com/1024XEngineer/Holonic-Asset/internal/project/repository/dao"
	projectservice "github.com/1024XEngineer/Holonic-Asset/internal/project/service"
	taxonomyhandler "github.com/1024XEngineer/Holonic-Asset/internal/taxonomy/handler"
	taxonomyservice "github.com/1024XEngineer/Holonic-Asset/internal/taxonomy/service"
)

func main() {
	projectDao := projectdao.NewMemoryProjectDao()
	projectRepository := projectrepository.NewProjectRepository(projectDao)
	projectService := projectservice.NewProjectService(projectRepository)
	projectHandler := projecthandler.NewProjectHandler(projectService)

	generationService := generationservice.NewGenerationService(nil, nil, nil)
	generationHandler := generationhandler.NewGenerationHandler(generationService)

	mediaService := mediaservice.NewMediaService()
	mediaHandler := mediahandler.NewMediaHandler(mediaService)

	taxonomyService := taxonomyservice.NewAssetDiscoveryService()
	taxonomyHandler := taxonomyhandler.NewTaxonomyHandler(taxonomyService)

	e := internal.Register(nil, projectHandler, generationHandler, mediaHandler, taxonomyHandler)
	e.Logger.Fatal(e.Start(":8080"))
}
