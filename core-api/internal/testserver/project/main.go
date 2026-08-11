package main

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"sort"
	"sync"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

func main() {
	store := &memoryProjectStore{projects: make(map[uint]projectdomain.Project)}
	manager := projectdomain.NewManager(store, nil)
	projectHandler := handler.NewProjectHandler(manager)
	server := httptest.NewServer(router.Register(nil, projectHandler, nil, nil))

	fmt.Println(server.URL)
	_, _ = io.Copy(io.Discard, os.Stdin)
	server.Close()
}

type memoryProjectStore struct {
	mu       sync.RWMutex
	nextID   uint
	projects map[uint]projectdomain.Project
}

func (s *memoryProjectStore) Insert(_ context.Context, project *projectdomain.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	project.ID = s.nextID
	s.projects[project.ID] = *project
	return nil
}

func (s *memoryProjectStore) FindByID(_ context.Context, projectID uint) (*projectdomain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return nil, projectdomain.ErrProjectNotFound
	}
	return cloneProject(project), nil
}

func (s *memoryProjectStore) FindByUserID(_ context.Context, userID uint) ([]*projectdomain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]*projectdomain.Project, 0)
	for _, project := range s.projects {
		if project.UserID == userID {
			projects = append(projects, cloneProject(project))
		}
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ID < projects[j].ID
	})
	return projects, nil
}

func (s *memoryProjectStore) Update(_ context.Context, update *projectdomain.ProjectUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[update.ID]
	if !ok {
		return projectdomain.ErrProjectNotFound
	}
	applyProjectUpdate(&project, update)
	s.projects[project.ID] = project
	return nil
}

func (s *memoryProjectStore) Remove(_ context.Context, projectID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[projectID]; !ok {
		return projectdomain.ErrProjectNotFound
	}
	delete(s.projects, projectID)
	return nil
}

func cloneProject(project projectdomain.Project) *projectdomain.Project {
	cloned := project
	return &cloned
}

func applyProjectUpdate(project *projectdomain.Project, update *projectdomain.ProjectUpdate) {
	if update.Name != nil {
		project.Name = *update.Name
	}
	if update.GameType != nil {
		project.GameType = *update.GameType
	}
	if update.Perspective != nil {
		project.Perspective = *update.Perspective
	}
	if update.TargetPlatform != nil {
		project.TargetPlatform = *update.TargetPlatform
	}
	if update.Description != nil {
		project.Description = *update.Description
	}
	if update.Reference != nil {
		project.Reference = *update.Reference
	}
	if update.Style != nil {
		project.Style = *update.Style
	}
}

var _ projectdomain.Store = (*memoryProjectStore)(nil)
