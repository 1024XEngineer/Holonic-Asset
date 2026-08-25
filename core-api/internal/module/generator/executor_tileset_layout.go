package generator

import (
	"fmt"
	"strings"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type tileSetPlacement struct {
	ItemIndex int
	Origin    TileSetCoordinate
	Positions []TileSetCoordinate
}

// assignTileSetLayout places Items in request order using only their occupied
// Shape cells. Model output never influences the resulting grid positions.
func assignTileSetLayout(request CreateTileSetPayload) ([]tileSetPlacement, error) {
	tileAmount := request.Dimensions.TileAmount
	if tileAmount.Columns == 0 || tileAmount.Rows == 0 {
		return nil, fmt.Errorf("generator: Tileset layout requires a positive grid")
	}
	if tileAmount.Rows > maxTileSetGridTiles ||
		tileAmount.Columns > maxTileSetGridTiles/tileAmount.Rows {
		return nil, fmt.Errorf(
			"generator: Tileset layout must not exceed %d cells",
			maxTileSetGridTiles,
		)
	}
	if len(request.Items) == 0 || len(request.Items) > maxTileSetItems {
		return nil, fmt.Errorf(
			"generator: Tileset layout requires between 1 and %d Items",
			maxTileSetItems,
		)
	}
	columns := int(tileAmount.Columns)
	rows := int(tileAmount.Rows)
	capacity := columns * rows

	occupied := make(map[TileSetCoordinate]struct{}, capacity)
	placements := make([]tileSetPlacement, len(request.Items))
	totalTiles := 0
	for itemIndex, item := range request.Items {
		if len(item.Shape) == 0 || len(item.Shape) > maxTilesPerItem {
			return nil, fmt.Errorf(
				"generator: Tileset Item %d Shape must contain between 1 and %d cells",
				itemIndex,
				maxTilesPerItem,
			)
		}
		totalTiles += len(item.Shape)
		if totalTiles > capacity {
			return nil, fmt.Errorf("generator: Tileset layout has more occupied cells than the grid")
		}
		localShape, err := normalizeTileSetShape(item.Shape)
		if err != nil {
			return nil, fmt.Errorf("generator: normalize Tileset Item %d Shape: %w", itemIndex, err)
		}
		for _, cell := range localShape {
			if cell[0] >= columns || cell[1] >= rows {
				return nil, fmt.Errorf(
					"generator: Tileset Item %d (%q) Shape cannot fit inside the %dx%d grid",
					itemIndex,
					item.Name,
					columns,
					rows,
				)
			}
		}
		origin, positions, found := findFirstTileSetPlacement(localShape, occupied, columns, rows)
		if !found {
			return nil, fmt.Errorf(
				"generator: Tileset Item %d (%q) does not fit in the remaining %dx%d grid",
				itemIndex,
				item.Name,
				columns,
				rows,
			)
		}
		for _, position := range positions {
			occupied[position] = struct{}{}
		}
		placements[itemIndex] = tileSetPlacement{
			ItemIndex: itemIndex,
			Origin:    origin,
			Positions: positions,
		}
	}
	return placements, nil
}

func normalizeTileSetShape(shape []TileSetCoordinate) ([]TileSetCoordinate, error) {
	if len(shape) == 0 {
		return nil, fmt.Errorf("shape is required")
	}
	minX, minY := shape[0][0], shape[0][1]
	seen := make(map[TileSetCoordinate]struct{}, len(shape))
	for _, cell := range shape {
		if cell[0] < 0 || cell[1] < 0 {
			return nil, fmt.Errorf("shape contains a negative cell [%d,%d]", cell[0], cell[1])
		}
		if _, duplicate := seen[cell]; duplicate {
			return nil, fmt.Errorf("shape contains duplicate cell [%d,%d]", cell[0], cell[1])
		}
		seen[cell] = struct{}{}
		minX = min(minX, cell[0])
		minY = min(minY, cell[1])
	}
	local := make([]TileSetCoordinate, len(shape))
	for index, cell := range shape {
		local[index] = TileSetCoordinate{cell[0] - minX, cell[1] - minY}
	}
	return local, nil
}

// findFirstTileSetPlacement returns the first valid origin in row-major order,
// scanning top-to-bottom and left-to-right.
func findFirstTileSetPlacement(
	shape []TileSetCoordinate,
	occupied map[TileSetCoordinate]struct{},
	columns int,
	rows int,
) (TileSetCoordinate, []TileSetCoordinate, bool) {
	if len(shape) == 0 || columns <= 0 || rows <= 0 {
		return TileSetCoordinate{}, nil, false
	}
	for y := range rows {
		for x := range columns {
			origin := TileSetCoordinate{x, y}
			valid := true
			for _, cell := range shape {
				position := TileSetCoordinate{origin[0] + cell[0], origin[1] + cell[1]}
				if position[0] < 0 || position[0] >= columns || position[1] < 0 || position[1] >= rows {
					valid = false
					break
				}
				if _, collision := occupied[position]; collision {
					valid = false
					break
				}
			}
			if valid {
				positions := make([]TileSetCoordinate, len(shape))
				for index, cell := range shape {
					positions[index] = TileSetCoordinate{origin[0] + cell[0], origin[1] + cell[1]}
				}
				return origin, positions, true
			}
		}
	}
	return TileSetCoordinate{}, nil, false
}

func resolveAddedTileSetItemPlacement(
	request AddTilesetItemPayload,
	content assetdomain.AssetContent,
	dimensions assetdomain.TileSetDimensions,
) (tileSetPlacement, error) {
	if request.Item == nil {
		return tileSetPlacement{}, invalidTaskPayload("item is required")
	}
	if len(content.Items) >= maxTileSetItems {
		return tileSetPlacement{}, invalidTaskPayload("Tileset already contains the maximum of %d Items", maxTileSetItems)
	}
	name := strings.TrimSpace(request.Item.Name)
	for _, existing := range content.Items {
		if strings.EqualFold(strings.TrimSpace(existing.Name), name) {
			return tileSetPlacement{}, invalidTaskPayload("Tileset Item name %q already exists", name)
		}
	}
	definition := TileSetItemDefinition{
		Name: request.Item.Name, Description: request.Item.Description, Shape: request.Item.Shape,
	}
	if err := validateTileSetItemDefinition("item", definition, dimensions); err != nil {
		return tileSetPlacement{}, err
	}
	occupied, err := indexTileSetContent(content, dimensions)
	if err != nil {
		return tileSetPlacement{}, err
	}
	occupiedCells := make(map[TileSetCoordinate]struct{}, len(occupied))
	for position := range occupied {
		occupiedCells[TileSetCoordinate{position.X, position.Y}] = struct{}{}
	}
	localShape, err := normalizeTileSetShape(request.Item.Shape)
	if err != nil {
		return tileSetPlacement{}, invalidTaskPayload("item.shape is invalid: %v", err)
	}
	columns, rows := int(dimensions.TileAmount.Columns), int(dimensions.TileAmount.Rows)
	if request.Item.Origin != nil {
		origin := TileSetCoordinate{*request.Item.Origin.X, *request.Item.Origin.Y}
		positions, placeErr := placeTileSetShapeAtOrigin(localShape, occupiedCells, origin, columns, rows)
		if placeErr != nil {
			return tileSetPlacement{}, placeErr
		}
		return tileSetPlacement{Origin: origin, Positions: positions}, nil
	}
	origin, positions, found := findFirstTileSetPlacement(localShape, occupiedCells, columns, rows)
	if !found {
		return tileSetPlacement{}, invalidTaskPayload(
			"Tileset Item %q does not fit in the remaining %dx%d grid", name, columns, rows,
		)
	}
	return tileSetPlacement{Origin: origin, Positions: positions}, nil
}

func placeTileSetShapeAtOrigin(
	shape []TileSetCoordinate,
	occupied map[TileSetCoordinate]struct{},
	origin TileSetCoordinate,
	columns int,
	rows int,
) ([]TileSetCoordinate, error) {
	positions := make([]TileSetCoordinate, len(shape))
	for index, cell := range shape {
		position := TileSetCoordinate{origin[0] + cell[0], origin[1] + cell[1]}
		if position[0] < 0 || position[0] >= columns || position[1] < 0 || position[1] >= rows {
			return nil, invalidTaskPayload(
				"item.origin (%d,%d) places item.shape outside the %dx%d grid",
				origin[0], origin[1], columns, rows,
			)
		}
		if _, collision := occupied[position]; collision {
			return nil, invalidTaskPayload(
				"item.origin (%d,%d) collides with occupied Tile position (%d,%d)",
				origin[0], origin[1], position[0], position[1],
			)
		}
		positions[index] = position
	}
	return positions, nil
}
