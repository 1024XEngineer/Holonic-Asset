package export

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/png"
	"io"
	"net/http"
	"path"
	"strings"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type exportAssetJSON struct {
	AssetID        uint                    `json:"assetId"`
	ProjectID      uint                    `json:"projectId"`
	RecordID       uint                    `json:"recordId,omitempty"`
	Version        uint                    `json:"version"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description,omitempty"`
	Type           assetdomain.AssetType   `json:"type"`
	Perspective    assetdomain.Perspective `json:"perspective"`
	Dimensions     json.RawMessage         `json:"dimensions"`
	DirectionCount uint                    `json:"directionCount,omitempty"`
	Prototype      []exportResource        `json:"prototype"`
	Animations     []exportAnimation       `json:"animations"`
	Items          []exportTileSetItem     `json:"items,omitempty"`
	Layers         []exportSceneryLayer    `json:"layers,omitempty"`
}

type exportResource struct {
	ID   uint   `json:"id"`
	Name string `json:"name,omitempty"`
	Path string `json:"path"`
}

type exportAnimation struct {
	ID         uint                                   `json:"id"`
	Name       string                                 `json:"name"`
	Direction  string                                 `json:"direction,omitempty"`
	Frames     []exportFrame                          `json:"frames"`
	Generation *assetdomain.AnimationGenerationConfig `json:"generation,omitempty"`
}

type exportFrame struct {
	ID       uint   `json:"id"`
	Path     string `json:"path"`
	Duration uint   `json:"duration,omitempty"`
}

type exportTileSetItem struct {
	Index uint                `json:"index"`
	Name  string              `json:"name,omitempty"`
	Tiles []exportTileSetTile `json:"tiles"`
}

type exportTileSetTile struct {
	Path     string                   `json:"path"`
	Position assetdomain.TilePosition `json:"position"`
}

type exportSceneryLayer struct {
	ID        uint                 `json:"id"`
	Name      string               `json:"name"`
	Path      string               `json:"path"`
	Position  assetdomain.Position `json:"position"`
	Transform json.RawMessage      `json:"transform,omitempty"`
	Visible   *bool                `json:"visible,omitempty"`
	Opacity   *float64             `json:"opacity,omitempty"`
	ZIndex    *int                 `json:"zIndex,omitempty"`
	Metadata  map[string]any       `json:"metadata,omitempty"`
}

type manifest struct {
	Format        string         `json:"format"`
	FormatVersion int            `json:"formatVersion"`
	Asset         manifestAsset  `json:"asset"`
	Files         []manifestFile `json:"files"`
}

type manifestAsset struct {
	AssetID  uint                  `json:"assetId"`
	RecordID uint                  `json:"recordId,omitempty"`
	Version  uint                  `json:"version"`
	Name     string                `json:"name"`
	Type     assetdomain.AssetType `json:"type"`
}

type manifestFile struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	SHA256      string `json:"sha256"`
}

func BuildPackage(ctx context.Context, snapshot Snapshot, resolver ReferenceResolver) ([]byte, taskResult, error) {
	if resolver == nil {
		return nil, taskResult{}, fmt.Errorf("export: reference resolver is required")
	}
	if !isSupportedAssetType(snapshot.Type) {
		return nil, taskResult{}, ErrUnsupportedAsset
	}
	var content assetdomain.AssetContent
	if len(snapshot.Content) > 0 {
		if err := json.Unmarshal(snapshot.Content, &content); err != nil {
			return nil, taskResult{}, fmt.Errorf("export: decode asset content: %w", err)
		}
	}
	files := make([]packageFile, 0)
	seenPaths := make(map[string]struct{})
	assetJSON := exportAssetJSON{AssetID: snapshot.AssetID, ProjectID: snapshot.ProjectID, RecordID: snapshot.RecordID, Version: snapshot.Version, Name: snapshot.Name, Description: snapshot.Description, Type: snapshot.Type, Perspective: snapshot.Perspective, Dimensions: append([]byte(nil), snapshot.Dimensions...), DirectionCount: content.DirectionCount, Prototype: []exportResource{}, Animations: []exportAnimation{}}
	if content.Prototype != nil {
		directions := directions(snapshot.Perspective, len(*content.Prototype))
		for index, resource := range *content.Prototype {
			if resource.URL == nil || strings.TrimSpace(*resource.URL) == "" {
				return nil, taskResult{}, fmt.Errorf("export: prototype %d has no image reference", index)
			}
			filePath := fmt.Sprintf("prototype/%03d.png", index)
			if index < len(directions) {
				filePath = fmt.Sprintf("prototype/%s.png", directions[index])
			}
			if _, exists := seenPaths[filePath]; exists {
				return nil, taskResult{}, fmt.Errorf("export: duplicate package path %q", filePath)
			}
			seenPaths[filePath] = struct{}{}
			data, err := fetchPNG(ctx, resolver, *resource.URL)
			if err != nil {
				return nil, taskResult{}, fmt.Errorf("export: prototype %d: %w", index, err)
			}
			files = append(files, packageFile{Path: filePath, Kind: "prototype-frame", ContentType: "image/png", Data: data})
			name := ""
			if index < len(directions) {
				name = directions[index]
			}
			assetJSON.Prototype = append(assetJSON.Prototype, exportResource{ID: resource.ID, Name: name, Path: filePath})
		}
	}
	for _, animation := range content.Animations {
		out := exportAnimation{ID: animation.ID, Name: animation.Name, Frames: []exportFrame{}}
		if animation.Generation != nil {
			out.Direction, out.Generation = animation.Generation.Direction, animation.Generation
		}
		directory := path.Join("animations", slug(animation.Name))
		if out.Direction != "" {
			directory = path.Join(directory, slug(out.Direction))
		}
		for index, frame := range animation.Frames {
			if frame.URL == nil || strings.TrimSpace(*frame.URL) == "" {
				return nil, taskResult{}, fmt.Errorf("export: animation %q frame %d has no image reference", animation.Name, index)
			}
			filePath := path.Join(directory, fmt.Sprintf("%03d.png", index))
			if _, exists := seenPaths[filePath]; exists {
				return nil, taskResult{}, fmt.Errorf("export: duplicate package path %q", filePath)
			}
			seenPaths[filePath] = struct{}{}
			data, err := fetchPNG(ctx, resolver, *frame.URL)
			if err != nil {
				return nil, taskResult{}, fmt.Errorf("export: animation %q frame %d: %w", animation.Name, index, err)
			}
			files = append(files, packageFile{Path: filePath, Kind: "animation-frame", ContentType: "image/png", Data: data})
			out.Frames = append(out.Frames, exportFrame{ID: frame.ID, Path: filePath, Duration: frame.Duration})
		}
		assetJSON.Animations = append(assetJSON.Animations, out)
	}
	if snapshot.Type == assetdomain.AssetTypeScenery {
		for layerIndex, layer := range content.Layers {
			resource := strings.TrimSpace(layer.Resource)
			if resource == "" {
				return nil, taskResult{}, fmt.Errorf("export: scenery layer %d has no image reference", layerIndex)
			}
			filePath := sceneryLayerPath(layerIndex, layer.Name)
			if _, exists := seenPaths[filePath]; exists {
				return nil, taskResult{}, fmt.Errorf("export: duplicate package path %q", filePath)
			}
			seenPaths[filePath] = struct{}{}
			data, err := fetchPNG(ctx, resolver, resource)
			if err != nil {
				return nil, taskResult{}, fmt.Errorf("export: scenery layer %d: %w", layerIndex, err)
			}
			files = append(files, packageFile{Path: filePath, Kind: "scenery-layer", ContentType: "image/png", Data: data})
			assetJSON.Layers = append(assetJSON.Layers, exportSceneryLayer{
				ID: layer.ID, Name: layer.Name, Path: filePath, Position: layer.Position,
				Transform: append(json.RawMessage(nil), layer.Transform...), Visible: layer.Visible,
				Opacity: layer.Opacity, ZIndex: layer.ZIndex, Metadata: layer.Metadata,
			})
		}
	}

	if snapshot.Type == assetdomain.AssetTypeTileSet {
		for itemIndex, item := range content.Items {
			exportedItem := exportTileSetItem{Index: uint(itemIndex), Name: item.Name, Tiles: []exportTileSetTile{}}
			itemDirectory := tileSetItemDirectory(itemIndex, item.Name)
			for tileIndex, tile := range item.Tiles {
				if tile.URL == nil || strings.TrimSpace(*tile.URL) == "" {
					return nil, taskResult{}, fmt.Errorf("export: tileSet item %d tile %d has no image reference", itemIndex, tileIndex)
				}
				filePath := path.Join(itemDirectory, fmt.Sprintf("tile-%03d-%03d.png", tile.Position.X, tile.Position.Y))
				if _, exists := seenPaths[filePath]; exists {
					return nil, taskResult{}, fmt.Errorf("export: duplicate package path %q", filePath)
				}
				seenPaths[filePath] = struct{}{}
				data, err := fetchPNG(ctx, resolver, *tile.URL)
				if err != nil {
					return nil, taskResult{}, fmt.Errorf("export: tileSet item %d tile %d: %w", itemIndex, tileIndex, err)
				}
				files = append(files, packageFile{Path: filePath, Kind: "tileset-tile", ContentType: "image/png", Data: data})
				exportedItem.Tiles = append(exportedItem.Tiles, exportTileSetTile{Path: filePath, Position: tile.Position})
			}
			assetJSON.Items = append(assetJSON.Items, exportedItem)
		}
	}

	assetJSONData, err := json.MarshalIndent(assetJSON, "", "  ")
	if err != nil {
		return nil, taskResult{}, fmt.Errorf("export: encode asset metadata: %w", err)
	}
	files = append(files, packageFile{Path: "asset.json", Kind: "asset-metadata", ContentType: "application/json", Data: append(assetJSONData, '\n')})
	manifestData, err := encodeManifest(snapshot, files)
	if err != nil {
		return nil, taskResult{}, err
	}
	files = append(files, packageFile{Path: "manifest.json", Kind: "manifest", ContentType: "application/json", Data: manifestData})
	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	for _, file := range files {
		if err := writeZipFile(zw, file); err != nil {
			_ = zw.Close()
			return nil, taskResult{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, taskResult{}, fmt.Errorf("export: close zip: %w", err)
	}
	name := slug(snapshot.Name)
	if name == "" {
		name = fmt.Sprintf("asset-%d", snapshot.AssetID)
	}
	return output.Bytes(), taskResult{AssetID: snapshot.AssetID, RecordID: snapshot.RecordID, Version: snapshot.Version, FileName: fmt.Sprintf("%s-v%d.zip", name, snapshot.Version), FileSize: int64(output.Len()), SHA256: digest(output.Bytes())}, nil
}

type packageFile struct {
	Path, Kind, ContentType string
	Data                    []byte
}

func isSupportedAssetType(assetType assetdomain.AssetType) bool {
	return assetType == assetdomain.AssetTypeCharacter || assetType == assetdomain.AssetTypeObject || assetType == assetdomain.AssetTypeTileSet || assetType == assetdomain.AssetTypeScenery
}

func sceneryLayerPath(index int, name string) string {
	filePath := fmt.Sprintf("layers/%03d.png", index)
	if value := slug(name); value != "" {
		filePath = fmt.Sprintf("layers/%03d-%s.png", index, value)
	}
	return filePath
}

func tileSetItemDirectory(index int, name string) string {
	directory := fmt.Sprintf("tiles/items/%03d", index)
	if value := slug(name); value != "" {
		directory += "-" + value
	}
	return directory
}

func encodeManifest(snapshot Snapshot, files []packageFile) ([]byte, error) {
	items := make([]manifestFile, 0, len(files)+1)
	for _, file := range files {
		items = append(items, manifestFile{Path: file.Path, Kind: file.Kind, ContentType: file.ContentType, Size: len(file.Data), SHA256: digest(file.Data)})
	}
	value := manifest{Format: "holonic-asset-package", FormatVersion: FormatVersion, Asset: manifestAsset{AssetID: snapshot.AssetID, RecordID: snapshot.RecordID, Version: snapshot.Version, Name: snapshot.Name, Type: snapshot.Type}, Files: items}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("export: encode manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func writeZipFile(zw *zip.Writer, file packageFile) error {
	if file.Path == "" || path.IsAbs(file.Path) || strings.HasPrefix(path.Clean(file.Path), "../") {
		return fmt.Errorf("export: invalid package path %q", file.Path)
	}
	writer, err := zw.Create(file.Path)
	if err != nil {
		return fmt.Errorf("export: create zip entry %q: %w", file.Path, err)
	}
	if _, err := writer.Write(file.Data); err != nil {
		return fmt.Errorf("export: write zip entry %q: %w", file.Path, err)
	}
	return nil
}

func fetchPNG(ctx context.Context, resolver ReferenceResolver, reference string) ([]byte, error) {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "data:") {
		return decodeAndEncodePNG(reference)
	}
	resolved, err := resolver.ResolveReference(ctx, reference)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved, nil)
	if err != nil {
		return nil, fmt.Errorf("create resource request: %w", err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download resource: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download resource: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read resource: %w", err)
	}
	return encodePNG(data)
}

func decodeAndEncodePNG(dataURL string) ([]byte, error) {
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 || !strings.Contains(strings.ToLower(dataURL[:comma]), ";base64") {
		return nil, fmt.Errorf("invalid image data URL")
	}
	data, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return nil, fmt.Errorf("decode image data URL: %w", err)
	}
	return encodePNG(data)
}

func encodePNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return output.Bytes(), nil
}

func digest(data []byte) string { value := sha256.Sum256(data); return hex.EncodeToString(value[:]) }

func directions(p assetdomain.Perspective, count int) []string {
	var value []string
	switch p {
	case assetdomain.PerspectiveSideOn:
		value = []string{"left", "right"}
	case assetdomain.PerspectiveTopDown:
		value = []string{"front", "right", "back", "left"}
	case assetdomain.PerspectiveIsometric:
		value = []string{"front", "front_right", "right", "back_right", "back", "back_left", "left", "front_left"}
	}
	if count < len(value) {
		return value[:count]
	}
	return value
}
