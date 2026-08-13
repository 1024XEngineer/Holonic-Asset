package prompts

import "fmt"

const tileSetItemTemplate = `Create exactly one production-ready 2D pixel-art game Tileset Item.

NON-OVERRIDABLE STYLE RULES:
- These rules have higher priority than the project brief, metadata, and references.
- Render only classic low-resolution 2D pixel art. Never render 3D, 2.5D, photorealistic, painterly, vector, or smooth high-definition art.
- Use a coarse square pixel grid, crisp hard edges, stepped silhouettes, intentional pixel clusters, selective dithering, and a small deliberate colour palette.
- Do not use anti-aliasing, subpixel detail, smooth curves, gradients, soft shadows, ambient occlusion, depth of field, texture filtering, or smooth resampling.
- Express depth only through flat pixel clusters and hard-edged value groups in the requested game-camera perspective.

AUTHORITATIVE SHAPE GUIDE:
- The first reference image is a generated occupancy guide, not a style reference.
- Pure black #000000 is the only editable interior. Pure green #00ff00 is protected and must remain unchanged.
- Match the guide's orientation, aspect ratio, canvas edges, cell boundaries, and black/green coordinates exactly.
- Do not translate, rotate, flip, mirror, skew, crop, pad, rescale, expand, contract, or simplify the Shape.
- Draw one coherent Item directly across every black occupied region. Do not draw a rectangle and rely on later cropping.
- Every occupied cell must contain meaningful connected Item content.
- Keep every subject pixel, outline, highlight, shadow, and decoration inside black regions. No Item pixel may enter a green region.
- References after the guide are Project style references. Use palette, material, scale, lighting, and perspective cues only; never treat them as Shape authority.

PROJECT BRIEF:
%s

PROJECT CONTEXT:
%s

ITEM:
- Name: %s
- Description: %s
- Local occupied cells: %s
- Tile size: %dx%d pixels
- Perspective: %s

OUTPUT CONTRACT:
- Return exactly one complete Item image whose canvas is the occupied cells' bounding rectangle.
- Preserve the guide grid and keep every occupied cell aligned to the same pixel density.
- Fill all protected regions with one flat, opaque pure green #00ff00 matte.
- Do not use #00ff00 inside the Item or add green spill, fringes, labels, borders, text, watermarks, scenery, ground planes, unrelated objects, or extra variants.`

// TileSetItem builds the mandatory prompt for one independently generated Item.
func TileSetItem(
	creativeBrief string,
	projectContext string,
	itemName string,
	itemDescription string,
	shape string,
	tileWidth int,
	tileHeight int,
	perspective string,
) string {
	return fmt.Sprintf(
		tileSetItemTemplate,
		creativeBrief,
		projectContext,
		itemName,
		itemDescription,
		shape,
		tileWidth,
		tileHeight,
		perspective,
	)
}
