package generator

import (
	"errors"
	"math"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestDecodeSceneryLayoutsAssociatesUnorderedResponseByStableID(t *testing.T) {
	layouts, err := decodeSceneryLayouts([]byte(`{
		"layers":[
			{"id":2,"position":{"x":100,"y":40},"scale":{"x":0.8,"y":0.8},"rotation":15,"opacity":0.75,"zIndex":20},
			{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-10}
		]
	}`), sceneryLayoutTestLayers(), sceneryLayoutTestDimensions())
	if err != nil {
		t.Fatalf("decode valid scenery layout: %v", err)
	}
	if len(layouts) != 2 || layouts[1].ZIndex != -10 || layouts[2].Position.X != 100 ||
		layouts[2].Scale.X != 0.8 || layouts[2].Rotation != 15 || layouts[2].Opacity != 0.75 {
		t.Fatalf("unexpected layouts: %+v", layouts)
	}
}

func TestDecodeSceneryLayoutsRejectsInvalidModelOutput(t *testing.T) {
	first := `{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":0}`
	second := `{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}`
	tests := []struct {
		name string
		json string
	}{
		{name: "malformed", json: `{"layers":[`},
		{name: "missing layers", json: `{}`},
		{name: "missing layer", json: `{"layers":[` + first + `]}`},
		{name: "duplicate ID", json: `{"layers":[` + first + `,` + first + `]}`},
		{name: "unknown ID", json: `{"layers":[` + first + `,{"id":3,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "unknown root field", json: `{"layers":[` + first + `,` + second + `],"explanation":"ok"}`},
		{name: "unknown layer field", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1,"visible":true}]}`},
		{name: "unknown position field", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0,"anchor":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "unknown scale field", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1,"uniform":true},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing ID", json: `{"layers":[` + first + `,{"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing position", json: `{"layers":[` + first + `,{"id":2,"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing position coordinate", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing scale", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing scale coordinate", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing rotation", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"opacity":1,"zIndex":1}]}`},
		{name: "missing opacity", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"zIndex":1}]}`},
		{name: "missing zIndex", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1}]}`},
		{name: "non-integer zIndex", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1.5}]}`},
		{name: "number overflow", json: `{"layers":[` + first + `,{"id":2,"position":{"x":1e1000,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "zero scale", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":0,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "negative scale", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":-1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "opacity below range", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":-0.1,"zIndex":1}]}`},
		{name: "opacity above range", json: `{"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1.1,"zIndex":1}]}`},
		{name: "outside canvas", json: `{"layers":[` + first + `,{"id":2,"position":{"x":640,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "trailing data", json: `{"layers":[` + first + `,` + second + `]} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeSceneryLayouts([]byte(test.json), sceneryLayoutTestLayers(), sceneryLayoutTestDimensions())
			if !errors.Is(err, ErrInvalidSceneryLayout) {
				t.Fatalf("expected invalid scenery layout, got %v", err)
			}
		})
	}
}

func TestValidateSceneryLayoutRejectsInvalidCandidates(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*sceneryLayoutCandidate)
	}{
		{name: "zero ID", mutate: func(candidate *sceneryLayoutCandidate) { value := uint(0); candidate.ID = &value }},
		{name: "non-finite position", mutate: func(candidate *sceneryLayoutCandidate) { value := math.Inf(1); candidate.Position.X = &value }},
		{name: "non-finite scale", mutate: func(candidate *sceneryLayoutCandidate) { value := math.NaN(); candidate.Scale.Y = &value }},
		{name: "non-finite rotation", mutate: func(candidate *sceneryLayoutCandidate) { value := math.Inf(-1); candidate.Rotation = &value }},
		{name: "non-finite opacity", mutate: func(candidate *sceneryLayoutCandidate) { value := math.NaN(); candidate.Opacity = &value }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := validSceneryLayoutCandidate()
			test.mutate(&candidate)
			if _, _, err := validateSceneryLayoutCandidate(candidate, sceneryLayoutTestDimensions()); err == nil {
				t.Fatal("expected invalid candidate to be rejected")
			}
		})
	}
}

func TestDecodeSceneryLayoutsRejectsInvalidProcessedLayerIDs(t *testing.T) {
	valid := []byte(`{"layers":[{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":0},{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`)
	for name, layers := range map[string][]ProcessedSceneryLayer{
		"zero":      {{ID: 0}, {ID: 2}},
		"duplicate": {{ID: 1}, {ID: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeSceneryLayouts(valid, layers, sceneryLayoutTestDimensions()); !errors.Is(err, ErrInvalidSceneryLayout) {
				t.Fatalf("expected invalid processed layers, got %v", err)
			}
		})
	}
}

func validSceneryLayoutCandidate() sceneryLayoutCandidate {
	id := uint(1)
	zero, one := float64(0), float64(1)
	zIndex := 0
	return sceneryLayoutCandidate{
		ID: &id, Position: &sceneryLayoutVectorCandidate{X: &zero, Y: &zero},
		Scale: &sceneryLayoutVectorCandidate{X: &one, Y: &one}, Rotation: &zero, Opacity: &one, ZIndex: &zIndex,
	}
}

func sceneryLayoutTestLayers() []ProcessedSceneryLayer {
	return []ProcessedSceneryLayer{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}}
}

func sceneryLayoutTestDimensions() assetdomain.Size {
	return assetdomain.Size{Width: 640, Height: 360}
}
