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
	insertCalls int
	inserted    *domain.Project
	update      *domain.ProjectUpdate
}

func (s *projectStoreStub) Insert(_ context.Context, project *domain.Project) error {
	s.insertCalls++
	s.inserted = project
	return nil
}

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

type referenceStoreStub struct {
	resolved     string
	persisted    string
	resolveCall  string
	resolveCalls []string
	persistCall  string
}

func (s *referenceStoreStub) ResolveReference(_ context.Context, reference string) (string, error) {
	s.resolveCall = reference
	s.resolveCalls = append(s.resolveCalls, reference)
	if s.resolved != "" {
		return s.resolved, nil
	}
	return "signed:" + reference, nil
}

func (s *referenceStoreStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persistCall = reference
	if s.persisted != "" {
		return s.persisted, nil
	}
	return "key:" + reference, nil
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

func TestCreatePersistsReferenceAsObjectKey(t *testing.T) {
	store := &projectStoreStub{}
	references := &referenceStoreStub{persisted: "projects/7/reference.png"}
	project := validProject()
	project.UserID = 101
	project.Name = "Starbound"
	project.Reference = "https://cdn.example/reference.png?e=123&token=signed"
	manager := domain.NewManager(store, nil, references)

	if err := manager.Create(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if references.persistCall != "https://cdn.example/reference.png?e=123&token=signed" || store.inserted == nil || store.inserted.Reference != "projects/7/reference.png" {
		t.Fatalf("reference was not normalized before persistence: persist=%q project=%+v", references.persistCall, store.inserted)
	}
}

func TestGenerateReferenceResolvesInputAndPersistsGeneratedImage(t *testing.T) {
	project := validProject()
	project.Reference = "https://cdn.example/reference.png?e=123&token=signed"
	images := &imageGenerationServiceStub{result: &imageclient.GenerateResult{
		Images: []imageclient.GeneratedImage{{Base64: "generated-image-base64", MediaType: "image/png"}},
	}}
	references := &referenceStoreStub{resolved: "https://cdn.example/reference.png?e=456&token=signed", persisted: "projects/7/generated.png"}
	manager := domain.NewManager(&projectStoreStub{}, images, references)

	generated, err := manager.GenerateReference(context.Background(), project)
	if err != nil {
		t.Fatalf("generate reference: %v", err)
	}
	if len(references.resolveCalls) != 2 || references.resolveCalls[0] != project.Reference || references.resolveCalls[1] != "projects/7/generated.png" || references.persistCall != "data:image/png;base64,generated-image-base64" {
		t.Fatalf("unexpected reference storage calls: %+v", references)
	}
	if len(images.request.ReferenceImages) != 1 || images.request.ReferenceImages[0] != references.resolved {
		t.Fatalf("expected signed input reference, got %+v", images.request.ReferenceImages)
	}
	if generated != references.resolved {
		t.Fatalf("expected generated reference URL %q, got %q", references.resolved, generated)
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

func TestGenerateReferenceBuildsProjectScreenshotPromptAndReturnsURL(t *testing.T) {
	const reference = "https://media.example/reference.png"
	project := &domain.Project{
		Name:           "Lantern Vale",
		GameType:       domain.GameTypeRPG,
		Perspective:    domain.PerspectiveIsometric,
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
	store := &projectStoreStub{}
	references := &referenceStoreStub{resolved: "https://cdn.example/generated.png?e=456&token=signed", persisted: "projects/7/generated.png"}
	manager := domain.NewManager(store, images, references)

	generated, err := manager.GenerateReference(context.Background(), project)
	if err != nil {
		t.Fatalf("generate reference: %v", err)
	}
	if generated != references.resolved {
		t.Fatalf("expected generated reference URL %q, got %q", references.resolved, generated)
	}
	if project.Reference != reference {
		t.Fatalf("expected input reference to remain unchanged, got %q", project.Reference)
	}
	if store.insertCalls != 0 {
		t.Fatalf("expected reference generation not to persist a project, got %d inserts", store.insertCalls)
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
	if len(images.request.ReferenceImages) != 1 || images.request.ReferenceImages[0] != references.resolved {
		t.Fatalf("expected resolved project reference to be forwarded, got %+v", images.request.ReferenceImages)
	}

	for _, fragment := range []string{
		"Lantern Vale",
		`the user-described game type "RPG"`,
		"Isometric 2D pixel-art gameplay perspective",
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
		"REFERENCE REGENERATION",
		"user's current result",
		"clearly new alternative",
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

func TestGenerateReferenceUsesTheUserBriefForDifferentGameTypes(t *testing.T) {
	cases := []struct {
		name        string
		gameType    domain.GameType
		description string
		forbidden   []string
	}{
		{
			name:        "farming",
			gameType:    domain.GameType(""),
			description: "玩家在安静的农场照料鸡、羊和菜地，给动物喂食，收获成熟作物。没有战斗，没有敌人。",
			forbidden:   []string{"three rectangular crop plots", "two chickens and one sheep", "date/time and coin or produce counter"},
		},
		{
			name:        "maze",
			gameType:    domain.GameType(""),
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
				Perspective:    domain.PerspectiveTopDown,
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

			if _, err := manager.GenerateReference(context.Background(), project); err != nil {
				t.Fatalf("generate reference: %v", err)
			}
			prompt := images.request.Prompt
			if !strings.Contains(prompt, tc.description) {
				t.Fatalf("expected raw user description in prompt")
			}
			for _, fragment := range []string{
				"SCENE DECISION",
				"AUTHENTIC PIXEL ART",
				"GAMEPLAY SCREEN UI SET",
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

func TestGenerateReferenceLetsTheModelFollowExplicitUISetRequests(t *testing.T) {
	project := validProject()
	project.Description = "玩家打开背包整理采集到的物品。"
	images := &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{
			Images: []imageclient.GeneratedImage{{Base64: "generated-image", MediaType: "image/png"}},
		},
	}
	manager := domain.NewManager(&projectStoreStub{}, images)

	if _, err := manager.GenerateReference(context.Background(), project); err != nil {
		t.Fatalf("generate reference: %v", err)
	}
	for _, fragment := range []string{
		"unless the user explicitly asks for it",
		"show only that requested interface",
		"use iconography or empty slots instead of readable words",
	} {
		if !strings.Contains(images.request.Prompt, fragment) {
			t.Errorf("expected UI Set policy to contain %q", fragment)
		}
	}
	if strings.Contains(images.request.Prompt, "REFERENCE REGENERATION") {
		t.Fatal("expected regeneration instructions only when a current reference is supplied")
	}
}

func TestGenerateReferenceRejectsInvalidReferenceFormats(t *testing.T) {
	tests := []struct {
		name      string
		reference string
	}{
		{name: "bare base64", reference: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"},
		{name: "malformed data URI", reference: "data:image/png;base64,%%%"},
		{name: "non-image base64 content", reference: "data:image/png;base64,aGVsbG8="},
		{name: "non-image data URI", reference: "data:text/plain;base64,aGVsbG8="},
		{name: "unsupported URL scheme", reference: "ftp://media.example/reference.png"},
		{name: "URL without host", reference: "https:///reference.png"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			project := validProject()
			project.Reference = tc.reference
			images := &imageGenerationServiceStub{}
			manager := domain.NewManager(&projectStoreStub{}, images)

			generated, err := manager.GenerateReference(context.Background(), project)
			if !errors.Is(err, domain.ErrInvalidProject) {
				t.Fatalf("expected invalid project error, got %v", err)
			}
			if generated != "" {
				t.Fatalf("expected no generated reference, got %q", generated)
			}
			if images.request != nil {
				t.Fatalf("expected invalid reference not to reach image service, got %+v", images.request)
			}
		})
	}
}

func TestGenerateReferenceDoesNotReplaceReferenceWhenGenerationFails(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	project := validProject()
	project.Reference = "https://media.example/existing-reference.png"
	images := &imageGenerationServiceStub{err: wantErr}
	manager := domain.NewManager(&projectStoreStub{}, images)

	generated, err := manager.GenerateReference(context.Background(), project)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected generation error %v, got %v", wantErr, err)
	}
	if generated != "" {
		t.Fatalf("expected empty reference, got %q", generated)
	}
	if project.Reference != "https://media.example/existing-reference.png" {
		t.Fatalf("expected existing reference to remain unchanged, got %q", project.Reference)
	}
}

func TestGenerateReferenceRequiresImageServiceAndImageResult(t *testing.T) {
	project := validProject()
	manager := domain.NewManager(&projectStoreStub{}, nil)

	generated, err := manager.GenerateReference(context.Background(), project)
	if !errors.Is(err, domain.ErrImageServiceRequired) {
		t.Fatalf("expected image service error, got %v", err)
	}
	if generated != "" {
		t.Fatalf("expected empty reference, got %q", generated)
	}

	manager = domain.NewManager(&projectStoreStub{}, &imageGenerationServiceStub{
		result: &imageclient.GenerateResult{},
	})
	generated, err = manager.GenerateReference(context.Background(), project)
	if !errors.Is(err, domain.ErrReferenceRequired) {
		t.Fatalf("expected reference result error, got %v", err)
	}
	if generated != "" {
		t.Fatalf("expected empty reference, got %q", generated)
	}
}

func validProject() *domain.Project {
	return &domain.Project{
		Name:           "Test Project",
		GameType:       domain.GameTypeACT,
		Perspective:    domain.PerspectiveSideOn,
		TargetPlatform: domain.PlatformTypePC,
	}
}
