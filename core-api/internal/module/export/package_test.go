package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type resolverStub struct{}

func (resolverStub) ResolveReference(_ context.Context, reference string) (string, error) {
	return reference, nil
}

type failingResolver struct{}

func (failingResolver) ResolveReference(context.Context, string) (string, error) {
	return "http://127.0.0.1:1/missing.png", nil
}

func pngDataURL(t *testing.T, value color.Color) string {
	t.Helper()
	var data bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	for y := range 3 {
		for x := range 2 {
			img.Set(x, y, value)
		}
	}
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
}

func TestBuildPackageCharacter(t *testing.T) {
	prototypeURL := pngDataURL(t, color.RGBA{R: 255, A: 255})
	frameURL := pngDataURL(t, color.RGBA{B: 255, A: 255})
	snapshot := Snapshot{
		AssetID: 7, ProjectID: 11, RecordID: 19, Version: 3, Name: "Blue Hero",
		Type: assetdomain.AssetTypeCharacter, Perspective: assetdomain.PerspectiveTopDown,
		Dimensions: json.RawMessage(`{"width":2,"height":3}`),
		Content:    json.RawMessage(`{"directionCount":4,"prototype":[{"id":1,"url":"` + prototypeURL + `"}],"animations":[{"id":2,"name":"Idle","generation":{"direction":"front","frameCount":1,"columns":1,"frameWidth":2,"frameHeight":3,"fps":10,"resolution":"720p","duration":1,"aspectRatio":"1:1"},"frames":[{"id":3,"url":"` + frameURL + `","duration":120}]}]}`),
	}
	data, result, err := BuildPackage(context.Background(), snapshot, resolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FileName != "blue-hero-v3.zip" || result.FileSize != int64(len(data)) || result.SHA256 == "" {
		t.Fatalf("unexpected result: %+v", result)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = contents
	}
	for _, name := range []string{"asset.json", "manifest.json", "prototype/front.png", "animations/idle/front/000.png"} {
		if _, ok := entries[name]; !ok {
			t.Fatalf("missing zip entry %q; entries=%v", name, entries)
		}
	}
	var assetJSON exportAssetJSON
	if err := json.Unmarshal(entries["asset.json"], &assetJSON); err != nil {
		t.Fatal(err)
	}
	if assetJSON.Prototype[0].Path != "prototype/front.png" || assetJSON.Animations[0].Frames[0].Duration != 120 {
		t.Fatalf("unexpected asset metadata: %+v", assetJSON)
	}
	if !strings.Contains(string(entries["manifest.json"]), "holonic-asset-package") {
		t.Fatal("manifest format missing")
	}
}

func TestBuildPackageScenery(t *testing.T) {
	backdropURL := pngDataURL(t, color.RGBA{R: 80, G: 120, B: 160, A: 255})
	treeURL := pngDataURL(t, color.RGBA{G: 180, A: 255})
	visible := true
	opacity := 0.75
	zIndex := 2
	content := assetdomain.AssetContent{
		Layers: []assetdomain.SceneryLayer{
			{
				ID: 11, Name: "Backdrop", Resource: backdropURL,
				Position:  assetdomain.Position{X: 320, Y: 180},
				Transform: json.RawMessage(`{"scale":{"x":1,"y":1},"rotation":0}`),
				Visible:   &visible, Opacity: &opacity, ZIndex: &zIndex,
				Metadata: map[string]any{"role": "background"},
			},
			{
				ID: 12, Name: "Pine Trees", Resource: treeURL,
				Position: assetdomain.Position{X: 400, Y: 200},
			},
		},
	}
	contentData, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		AssetID: 9, ProjectID: 13, Version: 2, Name: "Forest Scene",
		Type:       assetdomain.AssetTypeScenery,
		Dimensions: json.RawMessage(`{"width":640,"height":360}`), Content: contentData,
	}

	data, result, err := BuildPackage(context.Background(), snapshot, resolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FileName != "forest-scene-v2.zip" || result.FileSize != int64(len(data)) {
		t.Fatalf("unexpected result: %+v", result)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = contents
	}
	for _, name := range []string{
		"asset.json",
		"manifest.json",
		"layers/000-backdrop.png",
		"layers/001-pine-trees.png",
	} {
		if _, ok := entries[name]; !ok {
			t.Fatalf("missing zip entry %q; entries=%v", name, entries)
		}
	}

	var assetJSON exportAssetJSON
	if err := json.Unmarshal(entries["asset.json"], &assetJSON); err != nil {
		t.Fatal(err)
	}
	if assetJSON.Type != assetdomain.AssetTypeScenery || len(assetJSON.Layers) != 2 {
		t.Fatalf("unexpected scenery metadata: %+v", assetJSON)
	}
	layer := assetJSON.Layers[0]
	if layer.ID != 11 || layer.Path != "layers/000-backdrop.png" || layer.Position != (assetdomain.Position{X: 320, Y: 180}) ||
		layer.Visible == nil || !*layer.Visible || layer.Opacity == nil || *layer.Opacity != opacity ||
		layer.ZIndex == nil || *layer.ZIndex != zIndex || layer.Metadata["role"] != "background" {
		t.Fatalf("unexpected scenery layer metadata: %+v", layer)
	}
}

func TestBuildPackageRejectsMissingSceneryLayer(t *testing.T) {
	snapshot := Snapshot{
		AssetID: 1, Version: 1, Name: "Scene", Type: assetdomain.AssetTypeScenery,
		Content: json.RawMessage(`{"layers":[{"id":1,"name":"Backdrop"}]}`),
	}
	_, _, err := BuildPackage(context.Background(), snapshot, resolverStub{})
	if err == nil || !strings.Contains(err.Error(), "has no image reference") {
		t.Fatalf("expected missing scenery layer error, got %v", err)
	}
}

func TestBuildPackageRejectsMissingFrame(t *testing.T) {
	snapshot := Snapshot{AssetID: 1, Version: 1, Name: "Hero", Type: assetdomain.AssetTypeObject, Content: json.RawMessage(`{"animations":[{"id":1,"name":"Idle","frames":[{"id":1,"url":"https://example.invalid/missing.png"}]}]}`)}
	_, _, err := BuildPackage(context.Background(), snapshot, failingResolver{})
	if err == nil || !strings.Contains(err.Error(), "download resource") {
		t.Fatalf("expected download error, got %v", err)
	}
}

func TestBuildPackageTileSet(t *testing.T) {
	tileURL := pngDataURL(t, color.RGBA{G: 255, A: 255})
	content := assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{
				Name: "Grass",
				Tiles: []assetdomain.Tile{
					{URL: &tileURL, Position: assetdomain.TilePosition{X: 0, Y: 1}},
					{URL: &tileURL, Position: assetdomain.TilePosition{X: 1, Y: 1}},
				},
			},
			{
				Name: "Stone",
				Tiles: []assetdomain.Tile{
					{URL: &tileURL, Position: assetdomain.TilePosition{X: 3, Y: 2}},
				},
			},
		},
	}
	contentData, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		AssetID: 8, ProjectID: 12, Version: 4, Name: "Forest Tiles",
		Type:       assetdomain.AssetTypeTileSet,
		Dimensions: json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":8,"rows":8}}`),
		Content:    contentData,
	}

	data, result, err := BuildPackage(context.Background(), snapshot, resolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FileName != "forest-tiles-v4.zip" || result.FileSize != int64(len(data)) {
		t.Fatalf("unexpected result: %+v", result)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = contents
	}
	for _, name := range []string{
		"asset.json",
		"manifest.json",
		"tiles/items/000-grass/tile-000-001.png",
		"tiles/items/000-grass/tile-001-001.png",
		"tiles/items/001-stone/tile-003-002.png",
	} {
		if _, ok := entries[name]; !ok {
			t.Fatalf("missing zip entry %q; entries=%v", name, entries)
		}
	}

	var assetJSON exportAssetJSON
	if err := json.Unmarshal(entries["asset.json"], &assetJSON); err != nil {
		t.Fatal(err)
	}
	if assetJSON.Type != assetdomain.AssetTypeTileSet || len(assetJSON.Items) != 2 || len(assetJSON.Items[0].Tiles) != 2 {
		t.Fatalf("unexpected tileSet metadata: %+v", assetJSON)
	}
	if assetJSON.Items[0].Tiles[1].Position != (assetdomain.TilePosition{X: 1, Y: 1}) ||
		assetJSON.Items[1].Tiles[0].Path != "tiles/items/001-stone/tile-003-002.png" {
		t.Fatalf("unexpected tileSet tile metadata: %+v", assetJSON.Items)
	}
}

func TestBuildPackageRejectsMissingTile(t *testing.T) {
	snapshot := Snapshot{
		AssetID: 1, Version: 1, Name: "Tiles", Type: assetdomain.AssetTypeTileSet,
		Content: json.RawMessage(`{"items":[{"name":"Grass","tiles":[{"position":{"x":0,"y":0}}]}]}`),
	}
	_, _, err := BuildPackage(context.Background(), snapshot, resolverStub{})
	if err == nil || !strings.Contains(err.Error(), "has no image reference") {
		t.Fatalf("expected missing tile error, got %v", err)
	}
}

type resolverFunc func(context.Context, string) (string, error)

func (f resolverFunc) ResolveReference(ctx context.Context, reference string) (string, error) {
	return f(ctx, reference)
}

func TestBuildPackageValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		snapshot Snapshot
		resolver ReferenceResolver
		want     string
	}{
		{
			name:     "resolver required",
			snapshot: Snapshot{Type: assetdomain.AssetTypeObject},
			want:     "reference resolver is required",
		},
		{
			name:     "unsupported asset",
			snapshot: Snapshot{Type: assetdomain.AssetTypeAudio},
			resolver: resolverStub{},
			want:     ErrUnsupportedAsset.Error(),
		},
		{
			name:     "invalid content",
			snapshot: Snapshot{Type: assetdomain.AssetTypeObject, Content: json.RawMessage(`{"prototype":`)},
			resolver: resolverStub{},
			want:     "decode asset content",
		},
		{
			name: string(assetdomain.AssetTypeCharacter) + " prototype reference required",
			snapshot: Snapshot{
				Type:    assetdomain.AssetTypeCharacter,
				Content: json.RawMessage(`{"prototype":[{"id":1}]}`),
			},
			resolver: resolverStub{},
			want:     "prototype 0 has no image reference",
		},
		{
			name: "animation frame reference required",
			snapshot: Snapshot{
				Type:    assetdomain.AssetTypeObject,
				Content: json.RawMessage(`{"animations":[{"name":"Idle","frames":[{}]}]}`),
			},
			resolver: resolverStub{},
			want:     "animation \"Idle\" frame 0 has no image reference",
		},
		{
			name: "tile reference required",
			snapshot: Snapshot{
				Type:    assetdomain.AssetTypeTileSet,
				Content: json.RawMessage(`{"items":[{"tiles":[{}]}]}`),
			},
			resolver: resolverStub{},
			want:     "tileSet item 0 tile 0 has no image reference",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := BuildPackage(context.Background(), test.snapshot, test.resolver)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("BuildPackage error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestFetchPNG(t *testing.T) {
	validPNG := pngDataURL(t, color.RGBA{R: 255, A: 255})
	validPNGData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(validPNG, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(validPNGData)
		case "/missing.png":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name      string
		reference string
		resolver  ReferenceResolver
		want      string
	}{
		{name: "data URL", reference: validPNG, resolver: resolverStub{}},
		{name: "resolver error", reference: "asset-key", resolver: resolverFunc(func(context.Context, string) (string, error) {
			return "", errors.New("resolver unavailable")
		}), want: "resolver unavailable"},
		{name: "invalid resolved URL", reference: "asset-key", resolver: resolverFunc(func(context.Context, string) (string, error) {
			return "://invalid", nil
		}), want: "create resource request"},
		{name: "HTTP error", reference: "asset-key", resolver: resolverFunc(func(context.Context, string) (string, error) {
			return server.URL + "/missing.png", nil
		}), want: "HTTP 404"},
		{name: "HTTP success", reference: "asset-key", resolver: resolverFunc(func(context.Context, string) (string, error) {
			return server.URL + "/ok.png", nil
		})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := fetchPNG(context.Background(), test.resolver, test.reference)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("fetchPNG error = %v, want substring %q", err, test.want)
				}
				return
			}
			if err != nil || len(data) == 0 {
				t.Fatalf("fetchPNG returned %d bytes, err=%v", len(data), err)
			}
			if _, err := png.Decode(bytes.NewReader(data)); err != nil {
				t.Fatalf("fetchPNG returned invalid PNG: %v", err)
			}
		})
	}
}

func TestDecodeAndEncodePNGRejectsInvalidData(t *testing.T) {
	for _, dataURL := range []string{
		"data:image/png,not-base64",
		"data:image/png;base64,%%%",
		"data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image")),
		"not-a-data-url",
	} {
		t.Run(dataURL, func(t *testing.T) {
			if _, err := decodeAndEncodePNG(dataURL); err == nil {
				t.Fatal("expected invalid data URL error")
			}
		})
	}
	if _, err := encodePNG([]byte("not an image")); err == nil {
		t.Fatal("expected image decode error")
	}
}

func TestWriteZipFileRejectsUnsafePath(t *testing.T) {
	for _, filePath := range []string{"", "/absolute.png", "../outside.png", "nested/../../outside.png"} {
		t.Run(filePath, func(t *testing.T) {
			zw := zip.NewWriter(io.Discard)
			if err := writeZipFile(zw, packageFile{Path: filePath, Data: []byte("x")}); err == nil {
				t.Fatal("expected invalid package path error")
			}
			_ = zw.Close()
		})
	}
}

func TestDirections(t *testing.T) {
	tests := []struct {
		name  string
		value assetdomain.Perspective
		count int
		want  []string
	}{
		{name: "side on truncated", value: assetdomain.PerspectiveSideOn, count: 1, want: []string{"left"}},
		{name: "isometric full", value: assetdomain.PerspectiveIsometric, count: 20, want: []string{"front", "front_right", "right", "back_right", "back", "back_left", "left", "front_left"}},
		{name: "unknown", value: assetdomain.Perspective("unknown"), count: 2, want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := directions(test.value, test.count); !slices.Equal(got, test.want) {
				t.Fatalf("directions() = %v, want %v", got, test.want)
			}
		})
	}
}
