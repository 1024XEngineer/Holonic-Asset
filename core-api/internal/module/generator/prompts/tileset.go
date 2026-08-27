package prompts

import "fmt"

const tileSetItemTemplate = `Create exactly one production-ready 2D pixel-art game Tileset Item.

NON-OVERRIDABLE STYLE RULES:
- These rules have higher priority than the project brief, metadata, and references.
- Render only classic low-resolution 2D pixel art. Never render 3D, 2.5D, photorealistic, painterly, vector, or smooth high-definition art.
- Use a coarse square pixel grid, crisp hard edges, stepped silhouettes, intentional pixel clusters, selective dithering, and a small deliberate colour palette.
- Do not use anti-aliasing, subpixel detail, smooth curves, gradients, soft shadows, ambient occlusion, depth of field, texture filtering, or smooth resampling.
- Express depth only through flat pixel clusters and hard-edged value groups in the requested game-camera perspective.
- Even when the requested output canvas is large, preserve the visual vocabulary of a genuinely low-resolution sprite enlarged with nearest-neighbour scaling. Never turn it into a high-definition illustration.

AUTHORITATIVE SHAPE GUIDE:
- The first reference image is a generated occupancy guide, not a style reference.
- Pure black #000000 is the only editable interior. Pure green #00ff00 is protected and must remain unchanged.
- Match the guide's orientation, aspect ratio, canvas edges, cell boundaries, and black/green coordinates exactly.
- Do not translate, rotate, flip, mirror, skew, crop, pad, rescale, expand, contract, or simplify the Shape.
- Render one complete continuous Item image across every black occupied region. The backend processor performs Tile cutting after generation; do not pre-cut, space apart, or visually separate Tiles.
- Draw the Item directly in the Shape. Do not draw a rectangle and rely on later cropping.
- Every occupied cell must contain meaningful connected Item content.
- Treat cell boundaries as placement coordinates only, never as design panels, visible gutters, spacing, frames, or separate cards.
- Adjacent occupied cells must share opaque artwork across their common edge. Material seams such as floorboards or paving joints must use coloured Item pixels, never green matte or empty space.
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
- Return exactly one complete Item image in PNG format whose canvas is the occupied cells' bounding rectangle.
- Preserve the guide grid and keep every occupied cell aligned to the same pixel density.
- Keep one connected opaque Item across all occupied cells; never separate neighbouring Tiles with matte or transparent lines.
- Fill all protected regions and any unoccupied background area outside the Item with one flat, opaque pure green #00ff00 matte for deterministic background removal. Do not draw a black background box, white glow, or checkerboard pattern.
- Replace every black occupied region with clearly visible, opaque Item artwork. Never return the black/green guide unchanged or leave an occupied cell as matte.
- Keep the outermost canvas border pure #00ff00 wherever it is protected so the processor can identify one border-connected matte region.
- Use colours with clear RGB separation from #00ff00 for every Item pixel; green design details must use a visibly different hue or value.
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

const tileSetEditTemplate = `Edit exactly one existing classic 2D pixel-art game asset.

NON-OVERRIDABLE RULES:
- Keep classic low-resolution 2D pixel art with crisp square pixels, hard edges, and no anti-aliasing, gradients, smooth resampling, 3D, photorealism, text, or scenery.
- The first reference is the authoritative current Tile. Preserve its exact canvas size, alpha silhouette, occupied footprint, camera perspective, pixel density, placement, and scale.
- Preserve every pixel on the outermost one-pixel canvas border exactly so this Tile continues to connect seamlessly to all neighbouring Tiles.
- You may redraw, reinterpret, recolour, relight, or replace the complete visible Tile interior when needed to apply the requested edit. Interior colour fidelity to the current Tile is not required.
- Keep all new artwork strictly inside the original alpha silhouette. Do not expand, contract, translate, rotate, flip, crop, pad, or rescale the silhouette.
- Render any transparent or unoccupied background area as exactly one flat, opaque pure green #00ff00 matte for deterministic background removal. Do not draw a white background, black box, or checkerboard pattern.
- Apply only this edit: %s
- Project context: %s
- Item name: %s
- Tile size: %dx%d pixels
- Perspective: %s
- Return exactly one Tile image in PNG format on the specified pure-green matte and no variants.`

// TileSetTileEdit constrains one independently edited Tile.
func TileSetTileEdit(
	creativeBrief string,
	projectContext string,
	itemName string,
	tileWidth int,
	tileHeight int,
	perspective string,
) string {
	return fmt.Sprintf(
		tileSetEditTemplate,
		creativeBrief,
		projectContext,
		itemName,
		tileWidth,
		tileHeight,
		perspective,
	)
}

const tileSetItemEditTemplate = `Edit one complete existing classic 2D pixel-art Tileset Item.

NON-OVERRIDABLE RULES:
- Keep classic low-resolution 2D pixel art with crisp square pixels, hard edges, and no anti-aliasing, gradients, smooth resampling, 3D, photorealism, text, or scenery.
- The first reference is an authoritative occupancy guide: black is editable Item area and green is protected. The second reference is the current complete Item.
- Preserve the exact occupied-cell footprint, canvas, camera perspective, pixel density, coherent cross-Tile seams, placement, and scale.
- Every occupied cell must contain meaningful connected content. No visible pixel may enter an omitted cell.
- Fill every protected region with one flat, opaque pure green #00ff00 matte.
- Replace every black guide region with clearly visible, opaque edited Item artwork. Never copy black guide pixels, return the guide unchanged, or leave an occupied cell black or matte-only.
- Keep Item colours clearly separated from pure green and keep the outer protected canvas edge connected to the green matte for deterministic background removal.
- Apply only this edit: %s
- Project context: %s
- Item name: %s
- Occupied cells: %s
- Tile size: %dx%d pixels
- Perspective: %s
- Return exactly one complete Item image in PNG format on the specified pure-green matte and no variants. Do not add transparency checkerboards, text, scenery, or extra previews.`

// TileSetItemEdit constrains regeneration of a complete persisted Item.
func TileSetItemEdit(
	creativeBrief string,
	projectContext string,
	itemName string,
	shape string,
	tileWidth int,
	tileHeight int,
	perspective string,
) string {
	return fmt.Sprintf(
		tileSetItemEditTemplate,
		creativeBrief,
		projectContext,
		itemName,
		shape,
		tileWidth,
		tileHeight,
		perspective,
	)
}
