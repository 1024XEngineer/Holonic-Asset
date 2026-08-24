package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func validateAddTilesetItemAsset(asset assetdomain.Asset, request AddTilesetItemPayload) error {
	if asset.ID == 0 {
		return invalidTaskPayload("Tileset asset %d not found", request.AssetID)
	}
	if asset.Type != assetdomain.AssetTypeTileSet {
		return invalidTaskPayload("asset %d must have type %s", request.AssetID, assetdomain.AssetTypeTileSet)
	}
	if asset.ProjectID != request.ProjectID {
		return invalidTaskPayload("asset %d does not belong to project %d", request.AssetID, request.ProjectID)
	}
	if err := assetdomain.ValidateDimensions(asset.Type, asset.Dimensions); err != nil {
		return invalidTaskPayload("Tileset asset %d dimensions are invalid: %v", request.AssetID, err)
	}
	var dimensions assetdomain.TileSetDimensions
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return invalidTaskPayload("decode Tileset asset %d dimensions: %v", request.AssetID, err)
	}
	if dimensions.TileSize.Width > maxTileEdge || dimensions.TileSize.Height > maxTileEdge ||
		uint64(dimensions.TileAmount.Columns)*uint64(dimensions.TileAmount.Rows) > maxTileSetGridTiles {
		return invalidTaskPayload("Tileset asset %d dimensions exceed processing limits", request.AssetID)
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return invalidTaskPayload("decode Tileset asset %d content: %v", request.AssetID, err)
	}
	_, err = resolveAddedTileSetItemPlacement(request, content, dimensions)
	return err
}

func (e *executor) addTileSetItem(
	ctx context.Context,
	request AddTilesetItemPayload,
) (json.RawMessage, error) {
	if err := validateAddTilesetItemPayload(&request); err != nil {
		return nil, err
	}
	addition, err := e.loadTileSetEditContext(ctx, request.AssetID, request.ProjectID)
	if err != nil {
		return nil, err
	}
	placement, err := resolveAddedTileSetItemPlacement(request, addition.content, addition.dimensions)
	if err != nil {
		return nil, err
	}
	references, err := e.resolveTileSetContextReferences(
		ctx, AddTilesetItem, addition.project, request.CreatingReference,
	)
	if err != nil {
		return nil, err
	}
	definition := TileSetItemDefinition{
		Name: request.Item.Name, Description: request.Item.Description, Shape: request.Item.Shape,
	}
	createRequest := CreateTileSetPayload{
		ProjectID: request.ProjectID, CreativeBrief: request.CreativeBrief,
		Dimensions: addition.dimensions, Items: []TileSetItemDefinition{definition},
	}
	processed, err := e.processTileSetItem(
		ctx, createRequest, addition.project, definition, 0, references,
	)
	if err != nil {
		return nil, err
	}
	return e.publishAddedTileSetItem(ctx, request.AssetID, addition.asset.Version, *processed, placement)
}

func (e *executor) publishAddedTileSetItem(
	ctx context.Context,
	assetID uint,
	version uint,
	item processedTileSetItem,
	placement tileSetPlacement,
) (json.RawMessage, error) {
	uploads, err := buildTileSetUploads(e.references, []processedTileSetItem{item}, []tileSetPlacement{placement})
	if err != nil {
		return nil, err
	}
	uploadedKeys, err := e.persistTileSetUploads(ctx, uploads)
	if err != nil {
		return nil, err
	}
	cleanup := func(cause error) error {
		if cleanupErr := e.references.DeleteObjects(context.WithoutCancel(ctx), uploadedKeys); cleanupErr != nil {
			return errors.Join(cause, fmt.Errorf("generator: clean up added Tileset Item uploads: %w", cleanupErr))
		}
		return cause
	}
	items := buildTileSetContentItems([]processedTileSetItem{item}, uploads)
	content, err := assetdomain.EncodeContent(assetdomain.AssetContent{Items: items})
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: encode added Tileset Item content: %w", err))
	}
	if len(items) != 1 || strings.TrimSpace(items[0].Name) == "" || len(items[0].Tiles) == 0 {
		return nil, cleanup(fmt.Errorf("generator: added Tileset Item candidate is empty"))
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID: assetID, Version: version, Content: content, GeneratedResources: uploadedKeys,
	})
}
