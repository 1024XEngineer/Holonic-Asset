package generator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type tileSetTileUpload struct {
	itemIndex      int
	tileIndex      int
	position       TileSetCoordinate
	region         imageprocessor.ImageRegion
	objectKey      string
	rawImageBase64 string
	rawMediaType   string
}

func buildTileSetContentItems(
	items []processedTileSetItem,
	uploads []tileSetTileUpload,
) []assetdomain.TileSetItem {
	contentItems := make([]assetdomain.TileSetItem, len(items))
	for _, upload := range uploads {
		if contentItems[upload.itemIndex].Name == "" {
			contentItems[upload.itemIndex].Name = items[upload.itemIndex].Name
		}
		key := upload.objectKey
		contentItems[upload.itemIndex].Tiles = append(contentItems[upload.itemIndex].Tiles, assetdomain.Tile{
			URL:      &key,
			Position: assetdomain.TilePosition{X: upload.position[0], Y: upload.position[1]},
		})
	}
	return contentItems
}

func buildTileSetUploads(
	references ReferenceStore,
	items []processedTileSetItem,
	layout []tileSetPlacement,
) ([]tileSetTileUpload, error) {
	if len(items) != len(layout) {
		return nil, fmt.Errorf("generator: Tileset layout count does not match Item count")
	}
	total := 0
	for _, item := range items {
		total += len(item.Tiles)
	}
	uploads := make([]tileSetTileUpload, 0, total)
	allocated := make(map[string]struct{}, total)
	for itemIndex, item := range items {
		placement := layout[itemIndex]
		if placement.ItemIndex != itemIndex || len(item.Tiles) != len(placement.Positions) {
			return nil, fmt.Errorf("generator: Tileset Item %d layout does not match processed Tiles", itemIndex)
		}
		for tileIndex, region := range item.Tiles {
			key, err := references.NewObjectKey("image/png")
			if err != nil {
				return nil, fmt.Errorf("generator: allocate Tileset Item %d Tile %d key: %w", itemIndex, tileIndex, err)
			}
			if _, duplicate := allocated[key]; duplicate {
				return nil, fmt.Errorf("generator: allocate Tileset Item %d Tile %d key: duplicate object key %q", itemIndex, tileIndex, key)
			}
			allocated[key] = struct{}{}
			var rawB64, rawMIME string
			if tileIndex == 0 {
				rawB64 = item.RawImageBase64
				rawMIME = item.RawMediaType
				if rawMIME == "" {
					rawMIME = "image/png"
				}
			}
			uploads = append(uploads, tileSetTileUpload{
				itemIndex: itemIndex, tileIndex: tileIndex, position: placement.Positions[tileIndex],
				region: region, objectKey: key, rawImageBase64: rawB64, rawMediaType: rawMIME,
			})
		}
	}
	return uploads, nil
}

func (e *executor) persistTileSetUploads(
	ctx context.Context,
	uploads []tileSetTileUpload,
) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	semaphore := make(chan struct{}, maxTileSetItemConcurrency)
	uploaded := make([]bool, len(uploads))
	var group sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	for index := range uploads {
		group.Go(func() {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			upload := uploads[index]
			mimeType := upload.region.MIMEType
			if mimeType == "" {
				mimeType = "image/png"
			}
			dataURL := "data:" + mimeType + ";base64," + upload.region.ImageBase64
			if err := e.references.PersistReferenceAt(ctx, upload.objectKey, dataURL); err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("generator: upload Tileset Item %d Tile %d: %w", upload.itemIndex, upload.tileIndex, err)
					cancel()
				})
				return
			}
			if upload.rawImageBase64 != "" {
				rawType := upload.rawMediaType
				if rawType == "" {
					rawType = "image/png"
				}
				unprocessedKey := addObjectKeySuffix(upload.objectKey, "-unprocessed")
				unprocessedDataURL := "data:" + rawType + ";base64," + upload.rawImageBase64
				if err := e.references.PersistReferenceAt(ctx, unprocessedKey, unprocessedDataURL); err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("generator: upload Tileset Item %d unprocessed: %w", upload.itemIndex, err)
						cancel()
					})
					return
				}
			}
			uploaded[index] = true
		})
	}
	group.Wait()
	keys := make([]string, 0, len(uploads)*2)
	for index, ok := range uploaded {
		if ok {
			keys = append(keys, uploads[index].objectKey)
			if uploads[index].rawImageBase64 != "" {
				keys = append(keys, addObjectKeySuffix(uploads[index].objectKey, "-unprocessed"))
			}
		}
	}
	if firstErr != nil {
		if cleanupErr := e.references.DeleteObjects(context.WithoutCancel(ctx), keys); cleanupErr != nil {
			return nil, errors.Join(firstErr, fmt.Errorf("generator: clean up partial Tileset uploads: %w", cleanupErr))
		}
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		if cleanupErr := e.references.DeleteObjects(context.WithoutCancel(ctx), keys); cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("generator: clean up canceled Tileset uploads: %w", cleanupErr))
		}
		return nil, err
	}
	return keys, nil
}
