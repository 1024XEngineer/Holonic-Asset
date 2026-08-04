package project_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type projectStoreStub struct {
	update *domain.ProjectUpdate
}

func (*projectStoreStub) Insert(context.Context, *domain.Project) error { return nil }

func (*projectStoreStub) FindByID(context.Context, uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

func (*projectStoreStub) FindByUserID(context.Context, uint) ([]*domain.Project, error) {
	return []*domain.Project{}, nil
}

func (s *projectStoreStub) Update(_ context.Context, update *domain.ProjectUpdate) error {
	s.update = update
	return nil
}

func (*projectStoreStub) Remove(context.Context, uint) error { return nil }

func TestUpdateForwardsPartialProjectUpdate(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	update := &domain.ProjectUpdate{ID: 42, Reference: &reference}
	store := &projectStoreStub{}
	manager := domain.NewManager(store, nil)

	if err := manager.Update(context.Background(), update); err != nil {
		t.Fatalf("update project: %v", err)
	}
	if store.update != update {
		t.Fatalf("expected update pointer %p, got %p", update, store.update)
	}
}

type imageGenerationServiceStub struct {
	request *imageclient.GenerateRequest
	result  *imageclient.GenerateResult
	err     error
}

func (s *imageGenerationServiceStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	s.request = request
	return s.result, s.err
}

func TestGenerateVisualImageBuildsProjectScreenshotPromptAndReturnsDataURL(t *testing.T) {
	const reference = "data:image/png;base64,reference-image"
	project := &domain.Project{
		UserID:         7,
		ID:             42,
		Name:           "Lantern Vale",
		GameType:       domain.GameTypeRPG,
		ViewType:       domain.ViewTypeIsometric,
		TargetPlatform: domain.PlatformTypeMobile,
		Description:    "A courier explores a flooded clockwork city and protects a caravan from mechanical beasts.",
		Reference:      reference,
		Style:          "16-bit storybook fantasy pixel art with warm lantern light and turquoise water",
	}
	images := &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{
			Images: []imageclient.GeneratedImage{{Base64: "generated-image-base64", MediaType: "image/png"}},
		},
	}
	manager := domain.NewManager(&projectStoreStub{}, images)

	generated, err := manager.GenerateVisualImage(context.Background(), project)
	if err != nil {
		t.Fatalf("generate visual image: %v", err)
	}
	if generated != project {
		t.Fatalf("expected returned project %p, got %p", project, generated)
	}
	if generated.Reference != "data:image/png;base64,generated-image-base64" {
		t.Fatalf("expected generated image data URL, got %q", generated.Reference)
	}
	if images.request == nil {
		t.Fatal("expected an image generation request")
	}
	if images.request.Size != "auto" {
		t.Fatalf("expected automatic image orientation, got %q", images.request.Size)
	}
	if images.request.Params["quality"] != "high" {
		t.Fatalf("expected high quality generation, got %+v", images.request.Params)
	}
	if len(images.request.ReferenceImages) != 1 || images.request.ReferenceImages[0] != reference {
		t.Fatalf("expected project reference to be forwarded, got %+v", images.request.ReferenceImages)
	}

	for _, fragment := range []string{
		"Lantern Vale",
		`the user-described game type "RPG"`,
		"2D isometric pixel-art gameplay camera",
		"mobile",
		project.Description,
		project.Style,
		"authentic 2D pixel-art gameplay screenshot",
		"384x256 for landscape",
		"256x384 for portrait",
		"256x256 for square",
		"uniform 4x square pixel blocks",
		"compact production tileset and sprite sheet",
		"no permanent side panel",
		"NO GENERATED TEXT",
		"zero generated text or pseudo-text",
		"REFERENCE IMAGE",
	} {
		if !strings.Contains(images.request.Prompt, fragment) {
			t.Errorf("expected prompt to contain %q", fragment)
		}
	}

	for _, fragment := range []string{
		"Represent as many reusable 2D game elements as possible",
		"short pixel-font-like labels",
		"such as a counter",
		"present it at 1536x1024",
	} {
		if strings.Contains(images.request.Prompt, fragment) {
			t.Errorf("expected anti-clutter prompt to omit %q", fragment)
		}
	}
}

func TestGenerateVisualImageUsesTheUserBriefForDifferentGameTypes(t *testing.T) {
	cases := []struct {
		name        string
		gameType    domain.GameType
		description string
		forbidden   []string
	}{
		{
			name:        "farming",
			gameType:    domain.GameTypeOther,
			description: "玩家在安静的农场照料鸡、羊和菜地，给动物喂食，收获成熟作物。没有战斗，没有敌人。",
			forbidden:   []string{"three rectangular crop plots", "two chickens and one sheep", "date/time and coin or produce counter"},
		},
		{
			name:        "maze",
			gameType:    domain.GameTypeOther,
			description: "玩家在一座废弃的石头迷宫里寻找出口，火把照亮一小段走廊。不要战斗，不要装备界面。",
			forbidden:   []string{"farming and animal-husbandry", "chickens", "sheep", "crop plots"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := &domain.Project{
				UserID:         7,
				Name:           tc.name,
				GameType:       tc.gameType,
				ViewType:       domain.ViewTypeTopDown,
				TargetPlatform: domain.PlatformTypePC,
				Description:    tc.description,
				Style:          "简单的2D像素风",
			}
			images := &imageGenerationServiceStub{
				result: &imageclient.GenerateResult{
					Images: []imageclient.GeneratedImage{{Base64: "generated-image-base64", MediaType: "image/png"}},
				},
			}
			manager := domain.NewManager(&projectStoreStub{}, images)

			if _, err := manager.GenerateVisualImage(context.Background(), project); err != nil {
				t.Fatalf("generate visual image: %v", err)
			}
			prompt := images.request.Prompt
			if !strings.Contains(prompt, tc.description) {
				t.Fatalf("expected raw user description in prompt")
			}
			for _, fragment := range []string{
				"SCENE DECISION",
				"AUTHENTIC PIXEL ART",
				"GAMEPLAY SCREEN UI",
				"no permanent side panel",
				"NO GENERATED TEXT",
				"Do not draw counters, labels, dialogue, or interaction text",
			} {
				if !strings.Contains(prompt, fragment) {
					t.Errorf("expected generic prompt to contain %q", fragment)
				}
			}
			for _, fragment := range tc.forbidden {
				if strings.Contains(strings.ToLower(prompt), strings.ToLower(fragment)) {
					t.Errorf("generic prompt unexpectedly hardcodes %q", fragment)
				}
			}
		})
	}
}

func TestGenerateVisualImageLetsTheModelFollowExplicitUIRequests(t *testing.T) {
	project := validProject()
	project.Description = "玩家打开背包整理采集到的物品。"
	images := &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{
			Images: []imageclient.GeneratedImage{{Base64: "generated-image", MediaType: "image/png"}},
		},
	}
	manager := domain.NewManager(&projectStoreStub{}, images)

	if _, err := manager.GenerateVisualImage(context.Background(), project); err != nil {
		t.Fatalf("generate visual image: %v", err)
	}
	for _, fragment := range []string{
		"unless the user explicitly asks for it",
		"show only that requested interface",
		"use iconography or empty slots instead of readable words",
	} {
		if !strings.Contains(images.request.Prompt, fragment) {
			t.Errorf("expected UI policy to contain %q", fragment)
		}
	}
}

func TestGenerateVisualImageDoesNotReuseGeneratedRawBase64AsStyleReference(t *testing.T) {
	project := validProject()
	project.Reference = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	images := &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{
			Images: []imageclient.GeneratedImage{{Base64: "next-generated-image", MediaType: "image/png"}},
		},
	}
	manager := domain.NewManager(&projectStoreStub{}, images)

	if _, err := manager.GenerateVisualImage(context.Background(), project); err != nil {
		t.Fatalf("generate visual image: %v", err)
	}
	if len(images.request.ReferenceImages) != 0 {
		t.Fatalf("expected generated preview base64 not to be reused, got %+v", images.request.ReferenceImages)
	}
}

func TestGenerateVisualImageDoesNotReplaceReferenceWhenGenerationFails(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	project := validProject()
	project.Reference = "existing-reference"
	images := &imageGenerationServiceStub{err: wantErr}
	manager := domain.NewManager(&projectStoreStub{}, images)

	generated, err := manager.GenerateVisualImage(context.Background(), project)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected generation error %v, got %v", wantErr, err)
	}
	if generated != nil {
		t.Fatalf("expected nil project, got %+v", generated)
	}
	if project.Reference != "existing-reference" {
		t.Fatalf("expected existing reference to remain unchanged, got %q", project.Reference)
	}
}

func TestGenerateVisualImageRequiresImageServiceAndImageResult(t *testing.T) {
	project := validProject()
	manager := domain.NewManager(&projectStoreStub{}, nil)

	generated, err := manager.GenerateVisualImage(context.Background(), project)
	if !errors.Is(err, domain.ErrImageServiceRequired) {
		t.Fatalf("expected image service error, got %v", err)
	}
	if generated != nil {
		t.Fatalf("expected nil project, got %+v", generated)
	}

	manager = domain.NewManager(&projectStoreStub{}, &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{},
	})
	generated, err = manager.GenerateVisualImage(context.Background(), project)
	if !errors.Is(err, domain.ErrVisualImageRequired) {
		t.Fatalf("expected visual image result error, got %v", err)
	}
	if generated != nil {
		t.Fatalf("expected nil project, got %+v", generated)
	}
}

func validProject() *domain.Project {
	return &domain.Project{
		UserID:         7,
		Name:           "Test Project",
		GameType:       domain.GameTypeACT,
		ViewType:       domain.ViewTypeSideView,
		TargetPlatform: domain.PlatformTypePC,
	}
}
