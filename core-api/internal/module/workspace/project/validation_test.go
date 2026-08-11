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
		GameType:       "Deck-building roguelike",
		Perspective:    domain.PerspectiveTopDown,
		TargetPlatform: domain.PlatformTypePC,
	}

	if err := valid.ValidateCreate(); err != nil {
		t.Fatalf("validate project with frontend-provided game type: %v", err)
	}

	tests := map[string]*domain.Project{
		"nil project":         nil,
		"missing user":        {Name: "Prototype", GameType: "RPG", Perspective: domain.PerspectiveTopDown, TargetPlatform: domain.PlatformTypePC},
		"blank name":          {UserID: 7, Name: " ", GameType: "RPG", Perspective: domain.PerspectiveTopDown, TargetPlatform: domain.PlatformTypePC},
		"invalid perspective": {UserID: 7, Name: "Prototype", GameType: "RPG", Perspective: "FirstPerson", TargetPlatform: domain.PlatformTypePC},
		"invalid platform":    {UserID: 7, Name: "Prototype", GameType: "RPG", Perspective: domain.PerspectiveTopDown, TargetPlatform: "Console"},
	}

	for name, project := range tests {
		t.Run(name, func(t *testing.T) {
			if err := project.ValidateCreate(); !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid project error, got %v", err)
			}
		})
	}
}

func TestProjectPerspectiveRequiresSupportedValue(t *testing.T) {
	if domain.Perspective("").Valid() {
		t.Fatal("expected empty perspective to be invalid")
	}
	if !domain.PlatformType("").Valid() {
		t.Fatal("expected empty platform type to be valid")
	}
	if domain.Perspective("SideView").Valid() {
		t.Fatal("expected legacy SideView not to be a valid perspective")
	}
	if !domain.PerspectiveSideOn.Valid() {
		t.Fatal("expected Side-On to be a valid perspective")
	}

	project := &domain.Project{UserID: 7, Name: "Prototype"}
	if err := project.ValidateCreate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected empty perspective to be rejected when creating a project: %v", err)
	}
	if err := project.ValidateReferenceGeneration(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected empty perspective to be rejected when generating a reference: %v", err)
	}

	customGameType := "Cozy farming simulator"
	emptyPlatformType := domain.PlatformType("")
	validUpdate := &domain.ProjectUpdate{
		ID:             42,
		GameType:       &customGameType,
		TargetPlatform: &emptyPlatformType,
	}
	if err := validUpdate.Validate(); err != nil {
		t.Fatalf("expected frontend-provided game type to be valid: %v", err)
	}

	emptyPerspective := domain.Perspective("")
	invalidUpdate := &domain.ProjectUpdate{ID: 42, Perspective: &emptyPerspective}
	if err := invalidUpdate.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected explicit empty perspective to be rejected: %v", err)
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
	tests := map[string]*domain.ProjectUpdate{
		"nil update": nil,
		"missing ID": {Description: new("updated")},
		"no fields":  {ID: 42},
		"blank name": {ID: 42, Name: &blankName},
	}

	for name, update := range tests {
		t.Run(name, func(t *testing.T) {
			if err := update.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid project error, got %v", err)
			}
		})
	}
}
