package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

const (
	referenceSize    = "auto"
	referenceQuality = "high"
)

var (
	ErrImageServiceRequired = errors.New("project: image generation service is required")
	ErrReferenceRequired    = errors.New("project: generated reference is required")
)

// Manager exposes the project lifecycle to transports and other modules.
type Manager interface {
	Create(ctx context.Context, project *Project) error
	ListByUID(ctx context.Context, userID uint) ([]*Project, error)
	GetDetail(ctx context.Context, projectID uint) (*Project, error)
	Update(ctx context.Context, update *ProjectUpdate) error
	Delete(ctx context.Context, projectID uint) error
	GenerateReference(ctx context.Context, project *Project) (string, error)
}

type manager struct {
	store       Store
	imageClient imageclient.ImageGenerationService
}

// NewManager constructs project use cases. The image service is optional so
// CRUD-only callers do not need to initialize the generation provider.
func NewManager(store Store, imageService imageclient.ImageGenerationService) Manager {
	return &manager{store: store, imageClient: imageService}
}

func (m *manager) Create(ctx context.Context, project *Project) error {
	if err := project.ValidateCreate(); err != nil {
		return err
	}
	return m.store.Insert(ctx, project)
}

func (m *manager) ListByUID(ctx context.Context, userID uint) ([]*Project, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	return m.store.FindByUserID(ctx, userID)
}

func (m *manager) GetDetail(ctx context.Context, projectID uint) (*Project, error) {
	if err := ValidateProjectID(projectID); err != nil {
		return nil, err
	}
	return m.store.FindByID(ctx, projectID)
}

func (m *manager) Update(ctx context.Context, update *ProjectUpdate) error {
	if err := update.Validate(); err != nil {
		return err
	}
	return m.store.Update(ctx, update)
}

func (m *manager) Delete(ctx context.Context, projectID uint) error {
	if err := ValidateProjectID(projectID); err != nil {
		return err
	}
	return m.store.Remove(ctx, projectID)
}

func (m *manager) GenerateReference(ctx context.Context, project *Project) (string, error) {
	if err := project.ValidateReferenceGeneration(); err != nil {
		return "", err
	}
	if m.imageClient == nil {
		return "", ErrImageServiceRequired
	}

	request := &imageclient.GenerateRequest{
		Prompt: buildReferencePrompt(project),
		Size:   referenceSize,
		Params: imageclient.Params{
			"quality": referenceQuality,
		},
	}
	if reference := referenceForGeneration(project.Reference); reference != "" {
		request.ReferenceImages = []string{reference}
	}

	generated, err := m.imageClient.Generate(ctx, request)
	if err != nil {
		return "", fmt.Errorf("project: generate reference: %w", err)
	}
	if generated == nil || len(generated.Images) == 0 || strings.TrimSpace(generated.Images[0].Base64) == "" {
		return "", ErrReferenceRequired
	}

	return referenceDataURL(generated.Images[0]), nil
}

func referenceDataURL(image imageclient.GeneratedImage) string {
	mediaType := strings.TrimSpace(image.MediaType)
	if mediaType == "" {
		mediaType = "image/png"
	}
	return "data:" + mediaType + ";base64," + image.Base64
}

var _ Manager = (*manager)(nil)

func buildReferencePrompt(project *Project) string {
	gameType := gameTypePrompt(project.GameType)
	perspective := perspectivePrompt(project.Perspective)
	platform := platformPrompt(project.TargetPlatform)
	gameplayPlan := gameplayPlanPrompt(project)
	hudPlan := hudPlanPrompt(project)
	style := promptValue(project.Style, "a clear, restrained 2D pixel-art style")
	description := promptValue(project.Description, "Show one ordinary playable moment that makes the game's main activity understandable.")

	return fmt.Sprintf(`Create one authentic 2D pixel-art gameplay screenshot from a finished game. It should look like a normal frame captured while a player is controlling the game, not concept art, key art, a poster, a character portrait, a map overview, or a collage.

USER PROJECT
- Name: %q
- Game type: %s
- Camera: %s
- Platform: %s
- Art style: %q
- User description: %q

SCENE DECISION
%s

Use the user description as the source of truth. First infer the game's main action and choose one small, playable moment that shows it clearly. If the description is broad, choose the simplest believable moment; do not fill the image with genre clichés. Show only the characters, creatures, objects, environment pieces, and interface needed for that moment. Never assume combat, enemies, weapons, a quest log, an inventory, or an equipment panel unless the user asks for it.

AUTHENTIC PIXEL ART
- Choose the canvas orientation from the user's description and the gameplay composition, not from the platform alone. Use a matching native logical pixel canvas: 384x256 for landscape, 256x384 for portrait, or 256x256 for square; then upscale it as uniform 4x square pixel blocks to the provider's corresponding output size. Keep every edge aligned to one coherent logical pixel grid. Never force a portrait game into landscape or add empty side filler merely to fit a wider canvas.
- Use deliberate connected pixel clusters, stepped curves, readable silhouettes, selective outlines, and short colour ramps. Use two to four tones per material and reuse the same ramps and sprite scale everywhere.
- Make the scene look assembled by a game engine from a compact production tileset and sprite sheet. Reuse ground, walls, paths, platforms, props, and interface motifs. Avoid one-off AI texture detail, random speckles, noisy micro-dithering, and a different treatment on every tile.
- Use a restrained palette of about 24-32 colours. Do not use smooth gradients, sub-pixel detail, anti-aliasing, 3D, vector-smooth shapes, painterly brushwork, glossy CGI, bloom, fog, lens effects, or depth-of-field.
- Keep characters and entities at normal gameplay sprite scale. Do not turn the player into a large illustration. Prefer a few clearly readable sprites and props over many tiny muddled objects. Leave quiet space so the playfield reads immediately.

GAMEPLAY SCREEN UI
%s

NO GENERATED TEXT
- Do not draw any words, letters, numerals, names, dialogue, tooltips, button labels, item labels, signs, logos, watermarks, or pseudo-text anywhere in the image. The project name is metadata and must not appear in the picture.
- Never imitate text with broken glyphs or decorative gibberish. Use icon-only indicators, simple bars, empty slots, and pictograms when interface feedback is necessary. Any real UI text and numbers will be rendered later by the game's UI layer with a real pixel font.

FINAL CHECK
Return one coherent playable screen with clear walkable or interactable spaces, consistent pixel scale, and a calm useful gameplay state. Remove anything that does not support the user's described game. The final image must contain zero generated text or pseudo-text.

REFERENCE IMAGE
If a reference image is supplied, use it only for visual language, palette relationships, sprite scale, or material treatment. Do not copy its words, logos, HUD layout, or accidental artifacts. Keep the user's game type and described activity in control of the composition.`,
		promptValue(project.Name, "Untitled game project"),
		gameType,
		perspective,
		platform,
		style,
		description,
		gameplayPlan,
		hudPlan,
	)
}

func gameplayPlanPrompt(project *Project) string {
	return fmt.Sprintf(`Interpret this as a %q game. Pick one concrete moment from the user's description, show the smallest set of game entities needed to understand the moment, and make the player's goal readable through placement and interaction. Do not replace the described loop with a more familiar genre and do not add extra mechanics to make the frame busy.`, project.GameType)
}

func hudPlanPrompt(*Project) string {
	return `Treat the interface as part of the described gameplay, not as a showcase overlay. Do not add a menu, inventory, equipment, loadout, or permanent side panel unless the user explicitly asks for it. If the user asks for a specific interface, show only that requested interface in a compact active-gameplay state, keep the playfield visible, and use iconography or empty slots instead of readable words, letters, numbers, or fake glyphs. Otherwise keep menus closed with no permanent side panel and use only small icon-only indicators that the described moment actually needs, such as a selected-object icon or an unlabeled health/resource bar. Do not draw counters, labels, dialogue, or interaction text. Keep the HUD small and subordinate to the playfield; if the game does not need it, show no HUD.`
}

// referenceForGeneration only forwards references in formats accepted by the
// image provider. Bare base64 values do not contain enough format information.
func referenceForGeneration(reference string) string {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "data:image/") ||
		strings.HasPrefix(reference, "https://") ||
		strings.HasPrefix(reference, "http://") {
		return reference
	}
	return ""
}

func gameTypePrompt(gameType GameType) string {
	label := strings.TrimSpace(string(gameType))
	if label == "" {
		return "an unspecified game type; rely on the user description"
	}
	return fmt.Sprintf("the user-described game type %q; rely on the user description for its actual activity", label)
}

func perspectivePrompt(perspective Perspective) string {
	switch perspective {
	case PerspectiveTopDown:
		return "top-down 2D pixel-art gameplay camera with a readable tile-based playfield, clear paths, interactables, and spatial relationships"
	case PerspectiveSideOn:
		return "side-on 2D pixel-art gameplay camera with layered parallax backgrounds, a clear traversal line, readable platforms, and strong sprite silhouettes"
	case PerspectiveIsometric:
		return "2D isometric pixel-art gameplay camera with a consistent pixel grid, clear depth, walkable tiles, elevation, and tactical readability"
	default:
		return "choose the 2D pixel-art gameplay camera best suited to the brief and keep it consistent with an authentic playable screenshot"
	}
}

func platformPrompt(platform PlatformType) string {
	switch platform {
	case PlatformTypePC:
		return "PC; use precise information hierarchy and keyboard/controller-friendly HUD conventions; choose portrait, landscape, or square from the user brief rather than assuming widescreen"
	case PlatformTypeMobile:
		return "mobile; use large touch-friendly controls, strong small-screen readability, safe margins, and restrained HUD density"
	case PlatformTypeWeb:
		return "web; use a responsive gameplay layout, concise HUD, and instantly understandable interactions; choose portrait, landscape, or square from the user brief rather than assuming widescreen"
	default:
		return "an unspecified target platform; infer suitable controls, orientation, and information density from the user brief"
	}
}

func promptValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
