package project_test

import (
	"errors"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestProjectValidateCreate(t *testing.T) {
	valid := &domain.Project{
		UserID:         7,
		Name:           "Prototype",
		GameType:       domain.GameTypeRPG,
		ViewType:       domain.ViewTypeTopDown,
		TargetPlatform: domain.PlatformTypePC,
	}

	if err := valid.ValidateCreate(); err != nil {
		t.Fatalf("validate project: %v", err)
	}

	tests := map[string]*domain.Project{
		"nil project":       nil,
		"missing user":      {Name: "Prototype", GameType: domain.GameTypeRPG, ViewType: domain.ViewTypeTopDown, TargetPlatform: domain.PlatformTypePC},
		"blank name":        {UserID: 7, Name: " ", GameType: domain.GameTypeRPG, ViewType: domain.ViewTypeTopDown, TargetPlatform: domain.PlatformTypePC},
		"invalid game type": {UserID: 7, Name: "Prototype", GameType: "FPS", ViewType: domain.ViewTypeTopDown, TargetPlatform: domain.PlatformTypePC},
		"invalid view type": {UserID: 7, Name: "Prototype", GameType: domain.GameTypeRPG, ViewType: "FirstPerson", TargetPlatform: domain.PlatformTypePC},
		"invalid platform":  {UserID: 7, Name: "Prototype", GameType: domain.GameTypeRPG, ViewType: domain.ViewTypeTopDown, TargetPlatform: "Console"},
	}

	for name, project := range tests {
		t.Run(name, func(t *testing.T) {
			if err := project.ValidateCreate(); !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid project error, got %v", err)
			}
		})
	}
}

func TestProjectUpdateValidateAllowsExplicitEmptyOptionalFields(t *testing.T) {
	empty := ""
	update := &domain.ProjectUpdate{ID: 42, Description: &empty, Reference: &empty, Style: &empty}

	if err := update.Validate(); err != nil {
		t.Fatalf("validate partial update: %v", err)
	}
}

func TestProjectUpdateValidateRejectsEmptyOrInvalidUpdates(t *testing.T) {
	blankName := " "
	invalidGameType := domain.GameType("FPS")
	tests := map[string]*domain.ProjectUpdate{
		"nil update":        nil,
		"missing ID":        {Description: new("updated")},
		"no fields":         {ID: 42},
		"blank name":        {ID: 42, Name: &blankName},
		"invalid game type": {ID: 42, GameType: &invalidGameType},
	}

	for name, update := range tests {
		t.Run(name, func(t *testing.T) {
			if err := update.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid project error, got %v", err)
			}
		})
	}
}
