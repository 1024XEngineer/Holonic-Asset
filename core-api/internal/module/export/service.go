package export

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

var (
	ErrInvalidRequest   = errors.New("export: invalid request")
	ErrUnsupportedAsset = errors.New("export: only character, object, and tileSet assets are supported")
)

type Service struct {
	assets     AssetReader
	references ReferenceResolver
	artifacts  ArtifactStore
	tasks      TaskManager
	now        func() time.Time
}

func NewService(assets AssetReader, references ReferenceResolver, artifacts ArtifactStore, tasks TaskManager) *Service {
	return &Service{assets: assets, references: references, artifacts: artifacts, tasks: tasks, now: time.Now}
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (CreateResponse, error) {
	if request.AssetID == 0 || s.assets == nil || s.tasks == nil || s.artifacts == nil || s.references == nil {
		return CreateResponse{}, ErrInvalidRequest
	}
	snapshot, err := s.snapshot(ctx, request.AssetID, request.Version)
	if err != nil {
		return CreateResponse{}, err
	}
	key, err := s.objectKey(snapshot)
	if err != nil {
		return CreateResponse{}, err
	}
	payload, err := json.Marshal(taskPayload{Snapshot: snapshot, ObjectKey: key})
	if err != nil {
		return CreateResponse{}, fmt.Errorf("export: encode task payload: %w", err)
	}
	t := &taskdomain.Task{Type: string(TaskType), Payload: payload, CreatedAt: s.now().UTC(), UpdatedAt: s.now().UTC()}
	taskID, err := s.tasks.Publish(ctx, t)
	if err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{ExportID: taskID, TaskID: taskID, Status: taskdomain.StatusPending.String()}, nil
}

func (s *Service) Get(ctx context.Context, exportID uint) (ExportResponse, error) {
	if exportID == 0 || s.tasks == nil {
		return ExportResponse{}, ErrInvalidRequest
	}
	t, err := s.tasks.GetDetail(ctx, exportID)
	if err != nil {
		return ExportResponse{}, err
	}
	if t == nil || t.Type != string(TaskType) {
		return ExportResponse{}, ErrInvalidRequest
	}
	var payload taskPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return ExportResponse{}, fmt.Errorf("export: decode task payload: %w", err)
	}
	response := ExportResponse{ExportID: exportID, AssetID: payload.Snapshot.AssetID, RecordID: payload.Snapshot.RecordID, Version: payload.Snapshot.Version, Status: t.Status.String(), Error: t.Error}
	if len(t.Result) == 0 {
		return response, nil
	}
	var result taskResult
	if err := json.Unmarshal(t.Result, &result); err != nil {
		return ExportResponse{}, fmt.Errorf("export: decode task result: %w", err)
	}
	response.FileName, response.FileSize, response.SHA256 = result.FileName, result.FileSize, result.SHA256
	response.DownloadURL = result.DownloadURL
	if result.ObjectKey != "" && s.artifacts != nil {
		// Resolve on every status read so private object URLs are refreshed after
		// their configured expiry instead of returning a stale task result URL.
		response.DownloadURL, err = s.artifacts.ResolveReference(ctx, result.ObjectKey)
		if err != nil {
			return ExportResponse{}, fmt.Errorf("export: resolve package URL: %w", err)
		}
	}
	return response, nil
}

func (s *Service) Handle(ctx context.Context, t *taskdomain.Task) (any, error) {
	if t == nil {
		return nil, ErrInvalidRequest
	}
	var payload taskPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return nil, fmt.Errorf("export: decode payload: %w", err)
	}
	packageData, result, err := BuildPackage(ctx, payload.Snapshot, s.references)
	if err != nil {
		return nil, err
	}
	if err := s.artifacts.PutArtifact(ctx, payload.ObjectKey, "application/zip", packageData); err != nil {
		return nil, fmt.Errorf("export: upload package: %w", err)
	}
	url, err := s.artifacts.ResolveReference(ctx, payload.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("export: resolve package URL: %w", err)
	}
	result.ObjectKey, result.DownloadURL = payload.ObjectKey, url
	return result, nil
}

func (s *Service) snapshot(ctx context.Context, assetID, version uint) (Snapshot, error) {
	asset, err := s.assets.GetDetail(ctx, assetID)
	if err != nil {
		return Snapshot{}, err
	}
	if asset.ID == 0 {
		return Snapshot{}, fmt.Errorf("export: asset %d not found", assetID)
	}
	if !isSupportedAssetType(asset.Type) {
		return Snapshot{}, ErrUnsupportedAsset
	}
	if version == 0 || version == asset.Version {
		return Snapshot{AssetID: asset.ID, ProjectID: asset.ProjectID, Version: asset.Version, Name: asset.Name, Description: asset.Description, Type: asset.Type, Perspective: asset.Perspective, Dimensions: append([]byte(nil), asset.Dimensions...), Content: append([]byte(nil), asset.Content...)}, nil
	}
	records, err := s.assets.GetRecordHistory(ctx, assetID)
	if err != nil {
		return Snapshot{}, err
	}
	for _, record := range records {
		if record.Version == version {
			return Snapshot{AssetID: record.AssetID, ProjectID: asset.ProjectID, RecordID: record.ID, Version: record.Version, Name: record.Name, Description: record.Description, Type: asset.Type, Perspective: record.Perspective, Dimensions: append([]byte(nil), record.Dimensions...), Content: append([]byte(nil), record.Content...)}, nil
		}
	}
	return Snapshot{}, fmt.Errorf("export: asset %d version %d not found", assetID, version)
}

func (s *Service) objectKey(snapshot Snapshot) (string, error) {
	var randomID [8]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return "", fmt.Errorf("export: generate object key: %w", err)
	}
	return fmt.Sprintf("exports/%d/%s-%s.zip", snapshot.AssetID, s.now().UTC().Format("20060102T150405Z"), hex.EncodeToString(randomID[:])), nil
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
