package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectDaoStub struct {
	dao.ProjectDao
	createInput *dao.Project
	createID    uint
	createErr   error

	findByIDParam uint
	findByIDRes   *dao.Project
	findByIDErr   error

	findByUserIDParam uint
	findByUserIDRes   []*dao.Project
	findByUserIDErr   error

	updateInput *dao.ProjectUpdate
	updateErr   error

	deleteID  uint
	deleteErr error
}

func (s *projectDaoStub) CreateProject(_ context.Context, project *dao.Project) (uint, error) {
	s.createInput = project
	if s.createErr != nil {
		return 0, s.createErr
	}
	return s.createID, nil
}

func (s *projectDaoStub) FindByID(_ context.Context, id uint) (*dao.Project, error) {
	s.findByIDParam = id
	return s.findByIDRes, s.findByIDErr
}

func (s *projectDaoStub) FindByUserID(_ context.Context, userID uint) ([]*dao.Project, error) {
	s.findByUserIDParam = userID
	return s.findByUserIDRes, s.findByUserIDErr
}

func (s *projectDaoStub) Update(_ context.Context, update *dao.ProjectUpdate) error {
	s.updateInput = update
	return s.updateErr
}

func (s *projectDaoStub) Delete(_ context.Context, id uint) error {
	s.deleteID = id
	return s.deleteErr
}

func TestProjectRepositoryInsert(t *testing.T) {
	stub := &projectDaoStub{createID: 99}
	repo := NewProjectRepository(stub)

	p := &domain.Project{
		UserID:         1,
		Name:           "Test Project",
		GameType:       "RPG",
		Perspective:    domain.PerspectiveTopDown,
		TargetPlatform: domain.PlatformTypeWeb,
		Description:    "A test RPG",
		Reference:      "ref",
		Style:          "pixel",
	}

	if err := repo.Insert(context.Background(), p); err != nil {
		t.Fatalf("unexpected insert error: %v", err)
	}

	if p.ID != 99 {
		t.Fatalf("expected project ID to be set to 99, got %d", p.ID)
	}

	if stub.createInput == nil || stub.createInput.Name != "Test Project" {
		t.Fatalf("unexpected create input: %+v", stub.createInput)
	}

	// Test error case
	wantErr := errors.New("insert failed")
	stubErr := &projectDaoStub{createErr: wantErr}
	repoErr := NewProjectRepository(stubErr)
	if err := repoErr.Insert(context.Background(), p); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestProjectRepositoryFindByID(t *testing.T) {
	stub := &projectDaoStub{
		findByIDRes: &dao.Project{
			ID:             42,
			UserID:         2,
			Name:           "My Game",
			GameType:       "Action",
			Perspective:    "Side-Scrolling",
			TargetPlatform: "PC",
			Description:    "desc",
			Reference:      "ref_url",
			Style:          "retro",
		},
	}
	repo := NewProjectRepository(stub)

	got, err := repo.FindByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if stub.findByIDParam != 42 {
		t.Fatalf("expected queried ID 42, got %d", stub.findByIDParam)
	}
	want := &domain.Project{
		ID:             42,
		UserID:         2,
		Name:           "My Game",
		GameType:       "Action",
		Perspective:    domain.Perspective("Side-Scrolling"),
		TargetPlatform: domain.PlatformType("PC"),
		Description:    "desc",
		Reference:      "ref_url",
		Style:          "retro",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	// Test NotFound normalization
	stubNotFound := &projectDaoStub{findByIDErr: dao.ErrProjectNotFound}
	repoNotFound := NewProjectRepository(stubNotFound)
	_, err = repoNotFound.FindByID(context.Background(), 42)
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected domain.ErrProjectNotFound, got %v", err)
	}
}

func TestProjectRepositoryFindByUserID(t *testing.T) {
	stub := &projectDaoStub{
		findByUserIDRes: []*dao.Project{
			{
				ID:             1,
				UserID:         10,
				Name:           "Game 1",
				Perspective:    "Top-Down",
				TargetPlatform: "Web",
			},
			{
				ID:             2,
				UserID:         10,
				Name:           "Game 2",
				Perspective:    "Side-Scrolling",
				TargetPlatform: "Mobile",
			},
		},
	}
	repo := NewProjectRepository(stub)

	list, err := repo.FindByUserID(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if stub.findByUserIDParam != 10 {
		t.Fatalf("expected userID 10, got %d", stub.findByUserIDParam)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(list))
	}
	if list[0].Name != "Game 1" || list[1].Name != "Game 2" {
		t.Fatalf("unexpected projects: %+v, %+v", list[0], list[1])
	}

	// Error case
	wantErr := errors.New("db error")
	stubErr := &projectDaoStub{findByUserIDErr: wantErr}
	_, err = NewProjectRepository(stubErr).FindByUserID(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestProjectRepositoryUpdate(t *testing.T) {
	stub := &projectDaoStub{}
	repo := NewProjectRepository(stub)

	name := "Updated Name"
	persp := domain.PerspectiveSideOn
	platform := domain.PlatformTypeMobile

	update := &domain.ProjectUpdate{
		ID:             15,
		Name:           &name,
		Perspective:    &persp,
		TargetPlatform: &platform,
	}

	if err := repo.Update(context.Background(), update); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if stub.updateInput == nil || stub.updateInput.ID != 15 || *stub.updateInput.Name != "Updated Name" ||
		*stub.updateInput.Perspective != "Side-On" || *stub.updateInput.TargetPlatform != "Mobile" {
		t.Fatalf("unexpected update input: %+v", stub.updateInput)
	}

	// Test NotFound error normalization
	stubNotFound := &projectDaoStub{updateErr: dao.ErrProjectNotFound}
	if err := NewProjectRepository(stubNotFound).Update(context.Background(), update); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected domain.ErrProjectNotFound, got %v", err)
	}
}

func TestProjectRepositoryRemove(t *testing.T) {
	stub := &projectDaoStub{}
	repo := NewProjectRepository(stub)

	if err := repo.Remove(context.Background(), 7); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	if stub.deleteID != 7 {
		t.Fatalf("expected deleteID 7, got %d", stub.deleteID)
	}

	// Test NotFound error normalization
	stubNotFound := &projectDaoStub{deleteErr: dao.ErrProjectNotFound}
	if err := NewProjectRepository(stubNotFound).Remove(context.Background(), 7); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected domain.ErrProjectNotFound, got %v", err)
	}
}

func TestProjectNilConversions(t *testing.T) {
	if got := convertProjectToDao(nil); got != nil {
		t.Fatalf("expected nil for nil domain project, got %+v", got)
	}
	if got := convertProjectUpdateToDao(nil); got != nil {
		t.Fatalf("expected nil for nil domain project update, got %+v", got)
	}
	if got := convertProjectToDomain(nil); got != nil {
		t.Fatalf("expected nil for nil dao project, got %+v", got)
	}
}
