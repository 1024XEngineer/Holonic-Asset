package generator

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func animationReference(asset assetdomain.Asset, direction string) (string, bool, error) {
	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return "", false, fmt.Errorf("generator: asset type %q does not support animation generation", asset.Type)
	}
	// Select the requested direction and resolve its -unprocessed image-hosting
	// reference.
	content, err := asset.DecodeContent()
	if err != nil {
		return "", false, fmt.Errorf("generator: decode animation asset %d content: %w", asset.ID, err)
	}
	prototypeIndex, err := animationDirectionIndex(direction, content.DirectionCount)
	if err != nil {
		return "", false, err
	}

	if content.Prototype == nil || prototypeIndex >= len(*content.Prototype) {
		return "", false, fmt.Errorf("generator: animation asset %d has no prototype for direction %q", asset.ID, direction)
	}
	prototype := (*content.Prototype)[prototypeIndex]
	if prototype.URL == nil || strings.TrimSpace(*prototype.URL) == "" {
		return "", false, fmt.Errorf("generator: animation asset %d prototype direction %q has no image URL", asset.ID, direction)
	}
	unprocessedURL := animationUnprocessedImageURL(strings.TrimSpace(*prototype.URL))
	return unprocessedURL, false, nil
}

func animationUnprocessedImageURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "data:") {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return addObjectKeySuffix(value, "-unprocessed")
	}
	parsed.Path = addObjectKeySuffix(parsed.Path, "-unprocessed")
	return parsed.String()
}

func (e *executor) persistAnimationFrames(
	ctx context.Context,
	result *AnimationGenerationResult,
) ([]assetdomain.Frame, error) {
	if e.references == nil {
		return nil, ErrAnimationReferenceStoreRequired
	}
	if result == nil {
		return nil, fmt.Errorf("generator: animation generation result is required")
	}
	if len(result.RawFrames) != 0 && len(result.RawFrames) != len(result.Frames) {
		return nil, fmt.Errorf("generator: raw animation frame count %d does not match processed frame count %d", len(result.RawFrames), len(result.Frames))
	}
	frames := make([]assetdomain.Frame, len(result.Frames))
	for index, frame := range result.Frames {
		mediaType := strings.TrimSpace(frame.MIMEType)
		if mediaType == "" {
			mediaType = "image/png"
		}
		dataURL := "data:" + mediaType + ";base64," + frame.ImageBase64
		objectKey, err := e.references.PersistReference(ctx, dataURL)
		if err != nil {
			return nil, fmt.Errorf("generator: persist animation frame %d: %w", index+1, err)
		}
		objectKey = strings.TrimSpace(objectKey)
		if objectKey == "" {
			return nil, fmt.Errorf("generator: persist animation frame %d: empty object key", index+1)
		}
		if strings.HasPrefix(objectKey, "data:") ||
			strings.HasPrefix(objectKey, "http://") ||
			strings.HasPrefix(objectKey, "https://") {
			return nil, fmt.Errorf("generator: persist animation frame %d: storage returned a non-object-key reference", index+1)
		}
		if len(result.RawFrames) != 0 {
			raw := result.RawFrames[index]
			if strings.TrimSpace(raw.ImageBase64) == "" {
				return nil, fmt.Errorf("generator: raw animation frame %d is empty", index+1)
			}
			rawMediaType := strings.TrimSpace(raw.MIMEType)
			if rawMediaType == "" {
				rawMediaType = "image/png"
			}
			rawDataURL := "data:" + rawMediaType + ";base64," + raw.ImageBase64
			if err := e.references.PersistReferenceAt(ctx, addObjectKeySuffix(objectKey, "-unprocessed"), rawDataURL); err != nil {
				return nil, fmt.Errorf("generator: persist raw animation frame %d: %w", index+1, err)
			}
		}
		frames[index] = assetdomain.Frame{
			ID:       uint(index + 1),
			URL:      &objectKey,
			Duration: result.FrameDurationMS,
		}
	}
	return frames, nil
}
