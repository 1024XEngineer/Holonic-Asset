package generator

import (
	"reflect"
	"strings"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestAssignTileSetLayoutUsesRequestOrderAndFirstRowMajorOrigin(t *testing.T) {
	request := CreateTileSetPayload{
		Dimensions: assetdomain.TileSetDimensions{
			TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 3},
		},
		Items: []TileSetItemDefinition{
			{Name: "Corner", Shape: []TileSetCoordinate{{2, 4}, {3, 4}, {2, 5}}},
			{Name: "Single", Shape: []TileSetCoordinate{{8, 7}}},
			{Name: "Edge", Shape: []TileSetCoordinate{{0, 0}, {1, 0}}},
		},
	}

	placements, err := assignTileSetLayout(request)
	if err != nil {
		t.Fatalf("assign layout: %v", err)
	}
	want := [][]TileSetCoordinate{
		{{0, 0}, {1, 0}, {0, 1}},
		{{2, 0}},
		{{1, 1}, {2, 1}},
	}
	for index, placement := range placements {
		if placement.ItemIndex != index || !reflect.DeepEqual(placement.Positions, want[index]) {
			t.Fatalf("Item %d placement: got %+v want %v", index, placement, want[index])
		}
	}
}

func TestAssignTileSetLayoutUsesRowMajorOrderInsteadOfDistance(t *testing.T) {
	request := CreateTileSetPayload{
		Dimensions: assetdomain.TileSetDimensions{
			TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 2},
		},
		Items: []TileSetItemDefinition{
			{Name: "First", Shape: []TileSetCoordinate{{0, 0}, {1, 0}, {2, 0}}},
			{Name: "Second", Shape: []TileSetCoordinate{{0, 0}}},
		},
	}

	placements, err := assignTileSetLayout(request)
	if err != nil {
		t.Fatalf("assign row-major layout: %v", err)
	}
	if got, want := placements[1].Origin, (TileSetCoordinate{3, 0}); got != want {
		t.Fatalf("second Item origin: got %v want row-major origin %v", got, want)
	}
}

func TestAssignTileSetLayoutUsesOnlyOccupiedShapeCells(t *testing.T) {
	request := CreateTileSetPayload{
		Dimensions: assetdomain.TileSetDimensions{
			TileAmount: assetdomain.TileAmount{Columns: 3, Rows: 2},
		},
		Items: []TileSetItemDefinition{
			{Name: "U", Shape: []TileSetCoordinate{{0, 0}, {2, 0}, {0, 1}, {2, 1}}},
			{Name: "Inside gap", Shape: []TileSetCoordinate{{0, 0}}},
		},
	}

	placements, err := assignTileSetLayout(request)
	if err != nil {
		t.Fatalf("assign sparse layout: %v", err)
	}
	if got := placements[1].Positions; !reflect.DeepEqual(got, []TileSetCoordinate{{1, 0}}) {
		t.Fatalf("sparse Shape gap was treated as occupied: %v", got)
	}
}

func TestAssignTileSetLayoutRejectsOverlapWhenNoShapePlacementFits(t *testing.T) {
	request := CreateTileSetPayload{
		Dimensions: assetdomain.TileSetDimensions{
			TileAmount: assetdomain.TileAmount{Columns: 2, Rows: 2},
		},
		Items: []TileSetItemDefinition{
			{Name: "Diagonal", Shape: []TileSetCoordinate{{0, 0}, {1, 1}}},
			{Name: "Horizontal", Shape: []TileSetCoordinate{{0, 0}, {1, 0}}},
		},
	}

	_, err := assignTileSetLayout(request)
	if err == nil || !strings.Contains(err.Error(), `Tileset Item 1 ("Horizontal") does not fit`) {
		t.Fatalf("expected no-fit error, got %v", err)
	}
}

func TestAssignTileSetLayoutRejectsUnboundedInputs(t *testing.T) {
	tests := []struct {
		name    string
		request CreateTileSetPayload
		want    string
	}{
		{
			name: "oversized grid",
			request: CreateTileSetPayload{
				Dimensions: assetdomain.TileSetDimensions{
					TileAmount: assetdomain.TileAmount{Columns: maxTileSetGridTiles, Rows: 2},
				},
				Items: []TileSetItemDefinition{{Shape: []TileSetCoordinate{{0, 0}}}},
			},
			want: "must not exceed 4096 cells",
		},
		{
			name: "too many Items",
			request: CreateTileSetPayload{
				Dimensions: assetdomain.TileSetDimensions{
					TileAmount: assetdomain.TileAmount{Columns: 8, Rows: 8},
				},
				Items: make([]TileSetItemDefinition, maxTileSetItems+1),
			},
			want: "between 1 and 64 Items",
		},
		{
			name: "oversized Shape",
			request: CreateTileSetPayload{
				Dimensions: assetdomain.TileSetDimensions{
					TileAmount: assetdomain.TileAmount{Columns: 64, Rows: 64},
				},
				Items: []TileSetItemDefinition{{Shape: make([]TileSetCoordinate, maxTilesPerItem+1)}},
			},
			want: "between 1 and 256 cells",
		},
		{
			name: "Shape wider than grid",
			request: CreateTileSetPayload{
				Dimensions: assetdomain.TileSetDimensions{
					TileAmount: assetdomain.TileAmount{Columns: 2, Rows: 2},
				},
				Items: []TileSetItemDefinition{{Name: "Wide", Shape: []TileSetCoordinate{{4, 3}, {6, 3}}}},
			},
			want: "Shape cannot fit inside the 2x2 grid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := assignTileSetLayout(test.request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestNormalizeTileSetShapeRejectsInvalidCells(t *testing.T) {
	for _, shape := range [][]TileSetCoordinate{
		{},
		{{-1, 0}},
		{{1, 1}, {1, 1}},
	} {
		if _, err := normalizeTileSetShape(shape); err == nil {
			t.Fatalf("expected Shape %v to fail", shape)
		}
	}
}

func TestResolveAddedTileSetItemPlacementUsesExistingOccupancy(t *testing.T) {
	existing := "existing.png"
	content := assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{
		Name: "U shape",
		Tiles: []assetdomain.Tile{
			{URL: &existing, Position: assetdomain.TilePosition{X: 0, Y: 0}},
			{URL: &existing, Position: assetdomain.TilePosition{X: 2, Y: 0}},
			{URL: &existing, Position: assetdomain.TilePosition{X: 0, Y: 1}},
			{URL: &existing, Position: assetdomain.TilePosition{X: 2, Y: 1}},
		},
	}}}
	request := AddTilesetItemPayload{Item: &AddTileSetItemDefinition{
		Name: "Center", Description: "fills the sparse gap", Shape: []TileSetCoordinate{{0, 0}},
	}}
	dimensions := assetdomain.TileSetDimensions{
		TileSize:   assetdomain.Size{Width: 16, Height: 16},
		TileAmount: assetdomain.TileAmount{Columns: 3, Rows: 2},
	}

	placement, err := resolveAddedTileSetItemPlacement(request, content, dimensions)
	if err != nil {
		t.Fatalf("resolve added Item placement: %v", err)
	}
	if got := placement.Positions; !reflect.DeepEqual(got, []TileSetCoordinate{{1, 0}}) {
		t.Fatalf("unexpected incremental placement: %v", got)
	}
}

func TestResolveAddedTileSetItemPlacementHonorsExplicitOrigin(t *testing.T) {
	x, y := 2, 1
	request := AddTilesetItemPayload{Item: &AddTileSetItemDefinition{
		Name: "Shelf", Description: "wide shelf", Shape: []TileSetCoordinate{{0, 0}, {1, 0}},
		Origin: &TileSetOrigin{X: &x, Y: &y},
	}}
	dimensions := assetdomain.TileSetDimensions{
		TileSize:   assetdomain.Size{Width: 16, Height: 16},
		TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 3},
	}

	placement, err := resolveAddedTileSetItemPlacement(request, assetdomain.AssetContent{}, dimensions)
	if err != nil {
		t.Fatalf("resolve explicit origin: %v", err)
	}
	want := []TileSetCoordinate{{2, 1}, {3, 1}}
	if placement.Origin != (TileSetCoordinate{2, 1}) || !reflect.DeepEqual(placement.Positions, want) {
		t.Fatalf("unexpected explicit placement: %+v", placement)
	}
}

func TestResolveAddedTileSetItemPlacementRejectsConflicts(t *testing.T) {
	existing := "existing.png"
	content := assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{
		Name:  "Pot",
		Tiles: []assetdomain.Tile{{URL: &existing, Position: assetdomain.TilePosition{X: 0, Y: 0}}},
	}}}
	dimensions := assetdomain.TileSetDimensions{
		TileSize:   assetdomain.Size{Width: 16, Height: 16},
		TileAmount: assetdomain.TileAmount{Columns: 2, Rows: 1},
	}
	x, y := 0, 0
	tests := []struct {
		name string
		item AddTileSetItemDefinition
		want string
	}{
		{
			name: "duplicate name",
			item: AddTileSetItemDefinition{Name: " pot ", Description: "duplicate", Shape: []TileSetCoordinate{{0, 0}}},
			want: "already exists",
		},
		{
			name: "explicit collision",
			item: AddTileSetItemDefinition{Name: "Tree", Description: "tree", Shape: []TileSetCoordinate{{0, 0}}, Origin: &TileSetOrigin{X: &x, Y: &y}},
			want: "collides",
		},
		{
			name: "no automatic fit",
			item: AddTileSetItemDefinition{Name: "Bench", Description: "wide", Shape: []TileSetCoordinate{{0, 0}, {1, 0}}},
			want: "does not fit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := resolveAddedTileSetItemPlacement(
				AddTilesetItemPayload{Item: &test.item}, content, dimensions,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}
