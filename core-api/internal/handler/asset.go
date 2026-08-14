package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type referencePersister interface {
	PersistReference(context.Context, string) (string, error)
}

type Handler struct {
	AssetManager domain.Manager
	references   referenceResolver
	persister    referencePersister
}

func NewHandler(manager domain.Manager, references ...referenceResolver) *Handler {
	var resolver referenceResolver
	var persister referencePersister
	if len(references) > 0 {
		resolver = references[0]
		persister, _ = resolver.(referencePersister)
	}
	return &Handler{AssetManager: manager, references: resolver, persister: persister}
}

func (h *Handler) GetAssets(
	x context.Context,
	request dto.GetAssetsRequest,
) (dto.SuccessResponse[dto.GetAssetsResponse], error) {
	projectID := request.ProjectID
	if projectID == 0 {
		return dto.SuccessResponse[dto.GetAssetsResponse]{}, echo.ErrBadRequest
	}

	assets, err := h.AssetManager.GetAssets(x, projectID, domain.AssetListFilter{
		Query: request.Query,
		Tags:  request.Tags,
		Types: request.Types,
	})
	if err != nil {
		return dto.SuccessResponse[dto.GetAssetsResponse]{}, err
	}

	items := make([]dto.AssetListItemResponse, len(assets))
	for index, asset := range assets {
		items[index] = dto.AssetListItemResponse{
			AssetID:     asset.ID,
			Name:        asset.Name,
			ProjectID:   asset.ProjectID,
			Type:        asset.Type,
			Description: asset.Description,
			Perspective: asset.Perspective,
			Dimensions:  append([]byte(nil), asset.Dimensions...),
			Tags:        asset.Tags,
			Version:     asset.Version,
		}
	}

	return dto.NewTypedSuccessResponse(dto.GetAssetsResponse{Assets: items}), nil
}

func (h *Handler) Detail(
	x context.Context,
	request dto.AssetDetailRequest,
) (dto.SuccessResponse[dto.AssetDetailResponse], error) {
	assetID := request.AssetID
	if assetID == 0 {
		return dto.SuccessResponse[dto.AssetDetailResponse]{}, echo.ErrBadRequest
	}

	asset, err := h.AssetManager.GetDetail(x, assetID)
	if err != nil {
		return dto.SuccessResponse[dto.AssetDetailResponse]{}, err
	}

	content, err := h.resolveAssetContent(x, asset.Content)
	if err != nil {
		return dto.SuccessResponse[dto.AssetDetailResponse]{}, err
	}

	return dto.NewTypedSuccessResponse(dto.AssetDetailResponse{
		AssetID:     asset.ID,
		Name:        asset.Name,
		ProjectID:   asset.ProjectID,
		Type:        asset.Type,
		Description: asset.Description,
		Perspective: asset.Perspective,
		Dimensions:  append([]byte(nil), asset.Dimensions...),
		Tags:        asset.Tags,
		Content:     content,
		Version:     asset.Version,
	}), nil
}

func (h *Handler) Record(
	x context.Context,
	asset dto.RecordAssetRequest,
) (dto.SuccessResponse[dto.RecordAssetResponse], error) {
	content := bytes.TrimSpace(asset.Content)
	if asset.AssetID == 0 || len(content) == 0 || bytes.Equal(content, []byte("null")) {
		return dto.SuccessResponse[dto.RecordAssetResponse]{}, echo.ErrBadRequest
	}
	persistedContent, err := h.persistAssetContent(x, content)
	if err != nil {
		return dto.SuccessResponse[dto.RecordAssetResponse]{}, err
	}
	record, err := h.AssetManager.CreateRecord(x, &domain.AssetRecord{
		AssetID: asset.AssetID,
		Content: persistedContent,
	}, 0)
	if err != nil {
		return dto.SuccessResponse[dto.RecordAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.RecordAssetResponse{
		RecordID:    record.ID,
		AssetID:     record.AssetID,
		Version:     record.Version,
		ContentID:   record.ContentID,
		CreatedAt:   record.CreatedAt,
		Name:        record.Name,
		Description: record.Description,
		Perspective: record.Perspective,
		Dimensions:  append([]byte(nil), record.Dimensions...),
	}), nil
}

func (h *Handler) Records(
	x context.Context,
	request dto.GetAssetRecordsRequest,
) (dto.SuccessResponse[dto.GetAssetRecordsResponse], error) {
	assetID := request.AssetID
	if assetID == 0 {
		return dto.SuccessResponse[dto.GetAssetRecordsResponse]{}, echo.ErrBadRequest
	}
	records, err := h.AssetManager.GetRecordHistory(x, assetID)
	if err != nil {
		return dto.SuccessResponse[dto.GetAssetRecordsResponse]{}, err
	}
	items := make([]dto.AssetRecordResponse, len(records))
	for index, record := range records {
		content, resolveErr := h.resolveAssetContent(x, record.Content)
		if resolveErr != nil {
			return dto.SuccessResponse[dto.GetAssetRecordsResponse]{}, resolveErr
		}
		items[index] = dto.AssetRecordResponse{
			RecordID:    record.ID,
			AssetID:     record.AssetID,
			Version:     record.Version,
			ContentID:   record.ContentID,
			CreatedAt:   record.CreatedAt,
			Name:        record.Name,
			Description: record.Description,
			Perspective: record.Perspective,
			Dimensions:  append([]byte(nil), record.Dimensions...),
			Content:     content,
		}
	}
	return dto.NewTypedSuccessResponse(dto.GetAssetRecordsResponse{Records: items}), nil
}

type referenceTransform func(context.Context, string) (string, error)

func (h *Handler) resolveAssetContent(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var transform referenceTransform
	if h.references != nil {
		transform = h.references.ResolveReference
	}
	return h.transformAssetContentReferences(ctx, raw, "resolve", transform)
}

func (h *Handler) persistAssetContent(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	return h.transformAssetContentReferences(ctx, raw, "persist", h.persistAssetReference)
}

func (h *Handler) transformAssetContentReferences(
	ctx context.Context,
	raw json.RawMessage,
	operation string,
	transform referenceTransform,
) (json.RawMessage, error) {
	var content map[string]json.RawMessage
	if err := json.Unmarshal(raw, &content); err != nil {
		return nil, fmt.Errorf("handler: decode asset content: %w", err)
	}
	if content == nil {
		return raw, nil
	}

	if operation == "resolve" {
		if value, ok := content["animations"]; ok {
			sanitized, err := stripAnimationGeneration(value)
			if err != nil {
				return nil, err
			}
			content["animations"] = sanitized
		}
	}

	if transform != nil {
		imageArrays := []struct {
			field      string
			childField string
		}{
			{field: "prototype"},
			{field: "animations", childField: "frames"},
			{field: "items", childField: "tiles"},
		}
		for _, imageArray := range imageArrays {
			value, ok := content[imageArray.field]
			if !ok {
				continue
			}
			transformed, err := transformReferenceArray(
				ctx,
				value,
				imageArray.field,
				imageArray.childField,
				"url",
				operation,
				transform,
			)
			if err != nil {
				return nil, err
			}
			content[imageArray.field] = transformed
		}
		if value, ok := content["layers"]; ok {
			transformed, err := transformReferenceArray(
				ctx,
				value,
				"layers",
				"",
				"resource",
				operation,
				transform,
			)
			if err != nil {
				return nil, err
			}
			content["layers"] = transformed
		}
	}

	encoded, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("handler: encode asset content: %w", err)
	}
	return json.RawMessage(encoded), nil
}

func stripAnimationGeneration(raw json.RawMessage) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	var animations []json.RawMessage
	if err := json.Unmarshal(raw, &animations); err != nil {
		return nil, fmt.Errorf("handler: decode asset content animations: %w", err)
	}
	for index, value := range animations {
		animation, err := decodeJSONObject(value)
		if err != nil {
			return nil, fmt.Errorf("handler: decode asset content animations[%d]: %w", index, err)
		}
		delete(animation, "generation")
		animations[index], err = json.Marshal(animation)
		if err != nil {
			return nil, fmt.Errorf("handler: encode asset content animations[%d]: %w", index, err)
		}
	}
	encoded, err := json.Marshal(animations)
	if err != nil {
		return nil, fmt.Errorf("handler: encode asset content animations: %w", err)
	}
	return encoded, nil
}

func transformReferenceArray(
	ctx context.Context,
	raw json.RawMessage,
	field string,
	childField string,
	referenceField string,
	operation string,
	transform referenceTransform,
) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("handler: decode asset content %s: %w", field, err)
	}
	for index, value := range values {
		object, err := decodeJSONObject(value)
		if err != nil {
			return nil, fmt.Errorf("handler: decode asset content %s[%d]: %w", field, index, err)
		}
		if childField == "" {
			if err := transformReferenceField(ctx, object, referenceField, transform); err != nil {
				return nil, fmt.Errorf(
					"handler: %s asset content %s[%d].%s: %w",
					operation,
					field,
					index,
					referenceField,
					err,
				)
			}
		} else if child, ok := object[childField]; ok {
			transformed, err := transformReferenceArray(
				ctx,
				child,
				fmt.Sprintf("%s[%d].%s", field, index, childField),
				"",
				referenceField,
				operation,
				transform,
			)
			if err != nil {
				return nil, err
			}
			object[childField] = transformed
		}
		values[index], err = json.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("handler: encode asset content %s[%d]: %w", field, index, err)
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("handler: encode asset content %s: %w", field, err)
	}
	return encoded, nil
}

func decodeJSONObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		if err == nil {
			err = fmt.Errorf("expected JSON object")
		}
		return nil, err
	}
	return object, nil
}

func transformReferenceField(
	ctx context.Context,
	object map[string]json.RawMessage,
	field string,
	transform referenceTransform,
) error {
	rawReference, ok := object[field]
	if !ok || len(rawReference) == 0 || rawReference[0] != '"' {
		return nil
	}
	var reference string
	if err := json.Unmarshal(rawReference, &reference); err != nil {
		return err
	}
	if strings.TrimSpace(reference) == "" {
		return nil
	}
	transformed, err := transform(ctx, reference)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(transformed)
	if err != nil {
		return err
	}
	object[field] = encoded
	return nil
}

func (h *Handler) persistAssetReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	persisted := reference
	var err error
	if h.persister != nil {
		persisted, err = h.persister.PersistReference(ctx, reference)
		if err != nil {
			return "", err
		}
		persisted = strings.TrimSpace(persisted)
	}
	if persisted == "" || strings.HasPrefix(persisted, "/") || isURLReference(persisted) {
		cause := fmt.Errorf("asset image reference %q is not a persisted object key", reference)
		return "", echo.NewHTTPError(http.StatusBadRequest, cause.Error()).SetInternal(cause)
	}
	return persisted, nil
}

func isURLReference(reference string) bool {
	if strings.HasPrefix(strings.ToLower(reference), "data:") || strings.HasPrefix(reference, "//") {
		return true
	}
	parsed, err := url.Parse(reference)
	return err == nil && (parsed.IsAbs() || parsed.Host != "")
}

func (h *Handler) CopyAsset(
	ctx context.Context,
	asset dto.CopyAssetRequest,
) (dto.SuccessResponse[dto.CopyAssetResponse], error) {
	if asset.AssetID == 0 {
		return dto.SuccessResponse[dto.CopyAssetResponse]{}, echo.ErrBadRequest
	}
	newAssetID, err := h.AssetManager.Copy(ctx, asset.AssetID, 0)
	if err != nil {
		return dto.SuccessResponse[dto.CopyAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.CopyAssetResponse{NewAssetID: newAssetID}), nil
}

func (h *Handler) RollBackAsset(
	ctx context.Context,
	asset dto.RollBackAssetRequest,
) (dto.SuccessResponse[dto.RollBackAssetResponse], error) {
	if asset.AssetID == 0 || asset.Version == 0 {
		return dto.SuccessResponse[dto.RollBackAssetResponse]{}, echo.ErrBadRequest
	}
	record, err := h.AssetManager.RollBackRecord(ctx, asset.AssetID, asset.Version)
	if err != nil {
		return dto.SuccessResponse[dto.RollBackAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.RollBackAssetResponse{
		AssetID:   record.AssetID,
		Version:   record.Version,
		ContentID: record.ContentID,
	}), nil
}

func (h *Handler) Delete(
	ctx context.Context,
	request dto.DeleteAssetRequest,
) (dto.SuccessResponse[dto.DeleteAssetResponse], error) {
	if request.AssetID == 0 {
		return dto.SuccessResponse[dto.DeleteAssetResponse]{}, echo.ErrBadRequest
	}
	if err := h.AssetManager.Delete(ctx, request.AssetID); err != nil {
		return dto.SuccessResponse[dto.DeleteAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.DeleteAssetResponse{Success: true}), nil
}

func (h *Handler) UpdateAsset(
	ctx context.Context,
	req dto.UpdateAssetRequest,
) (dto.SuccessResponse[dto.UpdateAssetResponse], error) {
	if req.AssetID == 0 {
		return dto.SuccessResponse[dto.UpdateAssetResponse]{}, echo.ErrBadRequest
	}

	asset, err := h.AssetManager.UpdateAsset(ctx, req.AssetID, &domain.AssetUpdate{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		Perspective: req.Perspective,
		Dimensions:  req.Dimensions,
	})
	if err != nil {
		return dto.SuccessResponse[dto.UpdateAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.UpdateAssetResponse{
		AssetID:     asset.ID,
		Name:        asset.Name,
		ProjectID:   asset.ProjectID,
		Type:        asset.Type,
		Description: asset.Description,
		Perspective: asset.Perspective,
		Dimensions:  append([]byte(nil), asset.Dimensions...),
		Tags:        asset.Tags,
		Version:     asset.Version,
	}), nil
}
