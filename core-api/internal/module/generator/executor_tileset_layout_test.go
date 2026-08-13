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
