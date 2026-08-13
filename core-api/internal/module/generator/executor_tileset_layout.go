package generator

import "fmt"

type tileSetPlacement struct {
	ItemIndex int
	Origin    TileSetCoordinate
	Positions []TileSetCoordinate
}

// assignTileSetLayout places Items in request order using only their occupied
// Shape cells. Model output never influences the resulting grid positions.
func assignTileSetLayout(request CreateTileSetPayload) ([]tileSetPlacement, error) {
	columns := int(request.Dimensions.TileAmount.Columns)
	rows := int(request.Dimensions.TileAmount.Rows)
	if columns <= 0 || rows <= 0 {
		return nil, fmt.Errorf("generator: Tileset layout requires a positive grid")
	}

	occupied := make(map[TileSetCoordinate]struct{})
	placements := make([]tileSetPlacement, len(request.Items))
	for itemIndex, item := range request.Items {
		localShape, err := normalizeTileSetShape(item.Shape)
		if err != nil {
			return nil, fmt.Errorf("generator: normalize Tileset Item %d Shape: %w", itemIndex, err)
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
