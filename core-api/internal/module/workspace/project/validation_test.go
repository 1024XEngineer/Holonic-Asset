package project_test

import (
	"errors"
	"strings"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestValidateUserIDAndProjectID(t *testing.T) {
	if err := domain.ValidateUserID(0); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject for userID 0, got %v", err)
	}
	if err := domain.ValidateUserID(10); err != nil {
		t.Fatalf("expected nil for valid userID, got %v", err)
	}

	if err := domain.ValidateProjectID(0); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject for projectID 0, got %v", err)
	}
	if err := domain.ValidateProjectID(10); err != nil {
		t.Fatalf("expected nil for valid projectID, got %v", err)
	}
}

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

func TestProjectValidateReferenceGeneration(t *testing.T) {
	var nilProject *domain.Project
	if err := nilProject.ValidateReferenceGeneration(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected error for nil project reference generation, got %v", err)
	}

	valid := &domain.Project{
		Name:           "Prototype",
		Perspective:    domain.PerspectiveTopDown,
		TargetPlatform: domain.PlatformTypePC,
		Reference:      "https://example.com/ref.png",
	}
	if err := valid.ValidateReferenceGeneration(); err != nil {
		t.Fatalf("expected valid reference generation, got %v", err)
	}
}

func TestProjectPerspectiveRequiresSupportedValue(t *testing.T) {
	if domain.Perspective("").Valid() {
		t.Fatal("expected empty perspective to be invalid")
	}
	if !domain.PlatformType("").Valid() {
		t.Fatal("expected empty platform type to be valid")
	}
	if domain.PlatformType("InvalidPlatform").Valid() {
		t.Fatal("expected invalid platform type to return false")
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

	customGameType := "  Cozy farming simulator  "
	emptyPlatformType := domain.PlatformType("")
	validUpdate := &domain.ProjectUpdate{
		ID:             42,
		GameType:       &customGameType,
		TargetPlatform: &emptyPlatformType,
	}
	if err := validUpdate.Validate(); err != nil {
		t.Fatalf("expected frontend-provided game type to be valid: %v", err)
	}
	if *validUpdate.GameType != "Cozy farming simulator" {
		t.Fatalf("expected game type to be trimmed, got %q", *validUpdate.GameType)
	}

	emptyPerspective := domain.Perspective("")
	invalidUpdate := &domain.ProjectUpdate{ID: 42, Perspective: &emptyPerspective}
	if err := invalidUpdate.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected explicit empty perspective to be rejected: %v", err)
	}

	invalidPlatform := domain.PlatformType("Nintendo")
	invalidPlatformUpdate := &domain.ProjectUpdate{ID: 42, TargetPlatform: &invalidPlatform}
	if err := invalidPlatformUpdate.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected invalid platform in update to be rejected: %v", err)
	}
}

func TestProjectValidateGameTypeRejectsBlankOversizedAndControlInput(t *testing.T) {
	oversized := strings.Repeat("a", 101)
	control := "action\nroguelike"
	tests := map[string]string{
		"blank":     "   ",
		"oversized": oversized,
		"control":   control,
	}

	for name, gameType := range tests {
		t.Run(name, func(t *testing.T) {
			project := &domain.Project{
				UserID:         7,
				Name:           "Prototype",
				GameType:       gameType,
				Perspective:    domain.PerspectiveTopDown,
				TargetPlatform: domain.PlatformTypePC,
			}
			if err := project.ValidateCreate(); !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid game type error, got %v", err)
			}
		})
	}

	empty := "   "
	update := &domain.ProjectUpdate{ID: 42, GameType: &empty}
	if err := update.Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected blank game type update to be rejected, got %v", err)
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

func TestProjectValidateReferenceGenerationAcceptsHTTPReferences(t *testing.T) {
	for _, reference := range []string{
		"",
		"  ",
		"http://media.example/reference.png",
		"HTTPS://media.example/reference.png?token=abc",
	} {
		project := &domain.Project{
			Name:           "Prototype",
			GameType:       "RPG",
			Perspective:    domain.PerspectiveTopDown,
			TargetPlatform: domain.PlatformTypePC,
			Reference:      reference,
		}
		if err := project.ValidateReferenceGeneration(); err != nil {
			t.Errorf("reference %q should be accepted: %v", reference, err)
		}
	}
}

func TestProjectValidateReferenceGenerationRejectsNilAndInvalidReferences(t *testing.T) {
	if err := (*domain.Project)(nil).ValidateReferenceGeneration(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected nil project error, got %v", err)
	}
	for _, reference := range []string{
		"ftp://media.example/reference.png",
		"https:///reference.png",
		"data:image/png;base64,aGVsbG8=",
		"reference.png",
	} {
		project := &domain.Project{
			Name:           "Prototype",
			GameType:       "RPG",
			Perspective:    domain.PerspectiveTopDown,
			TargetPlatform: domain.PlatformTypePC,
			Reference:      reference,
		}
		if err := project.ValidateReferenceGeneration(); !errors.Is(err, domain.ErrInvalidProject) {
			t.Errorf("reference %q should be rejected: %v", reference, err)
		}
	}
}

func TestProjectUpdateValidateRejectsInvalidPerspectiveAndPlatform(t *testing.T) {
	invalidPerspective := domain.Perspective("FirstPerson")
	if err := (&domain.ProjectUpdate{ID: 42, Perspective: &invalidPerspective}).Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected invalid perspective error, got %v", err)
	}
	invalidPlatform := domain.PlatformType("Console")
	if err := (&domain.ProjectUpdate{ID: 42, TargetPlatform: &invalidPlatform}).Validate(); !errors.Is(err, domain.ErrInvalidProject) {
		t.Fatalf("expected invalid platform error, got %v", err)
	}
}
