package export

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"testing"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type assetReaderStub struct {
	detail  assetdomain.Asset
	records []assetdomain.AssetRecord
}

func (s assetReaderStub) GetDetail(context.Context, uint) (assetdomain.Asset, error) {
	return s.detail, nil
}
func (s assetReaderStub) GetRecordHistory(context.Context, uint) ([]assetdomain.AssetRecord, error) {
	return s.records, nil
}

type taskManagerStub struct{ task *taskdomain.Task }

func (s *taskManagerStub) Publish(_ context.Context, task *taskdomain.Task) (uint, error) {
	task.ID = 42
	s.task = task
	return task.ID, nil
}
func (s *taskManagerStub) GetDetail(context.Context, uint) (*taskdomain.Task, error) {
	return s.task, nil
}

type artifactStoreStub struct {
	data []byte
	key  string
}

func (s *artifactStoreStub) PutArtifact(_ context.Context, key, _ string, data []byte) error {
	s.key, s.data = key, append([]byte(nil), data...)
	return nil
}
func (s *artifactStoreStub) ResolveReference(_ context.Context, key string) (string, error) {
	return "https://download.example/" + key, nil
}

type managerResolverStub struct{}

func (managerResolverStub) ResolveReference(_ context.Context, reference string) (string, error) {
	return reference, nil
}

func managerPNGDataURL(t *testing.T) string {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
}

func TestManagerCreateSnapshotsRequestedVersionAndHandle(t *testing.T) {
	content := json.RawMessage(`{"prototype":[{"id":1,"url":"` + managerPNGDataURL(t) + `"}]}`)
	assets := assetReaderStub{
		detail:  assetdomain.Asset{ID: 7, ProjectID: 9, Type: assetdomain.AssetTypeObject, Name: "Chest", Version: 3, Content: json.RawMessage(`{"prototype":[]}`)},
		records: []assetdomain.AssetRecord{{ID: 20, AssetID: 7, Version: 2, Name: "Old Chest", Content: content}},
	}
	tasks := &taskManagerStub{}
	artifacts := &artifactStoreStub{}
	manager := NewManager(assets, managerResolverStub{}, artifacts, tasks)

	created, err := manager.Create(context.Background(), CreateRequest{AssetID: 7, Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	if created.ExportID != 42 || created.Status != "pending" {
		t.Fatalf("unexpected create response: %+v", created)
	}
	var payload taskPayload
	if err := json.Unmarshal(tasks.task.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Snapshot.Version != 2 || payload.Snapshot.RecordID != 20 || payload.Snapshot.Name != "Old Chest" {
		t.Fatalf("unexpected snapshot: %+v", payload.Snapshot)
	}

	result, err := manager.Handle(context.Background(), tasks.task)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	tasks.task.Status = taskdomain.StatusCompleted
	tasks.task.Result = encoded
	response, err := manager.Get(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != "completed" || response.DownloadURL == "" || len(artifacts.data) == 0 {
		t.Fatalf("unexpected export response: %+v", response)
	}
}

func TestManagerCreateReturnsNotFoundForMissingAsset(t *testing.T) {
	manager := NewManager(
		assetReaderStub{},
		managerResolverStub{},
		&artifactStoreStub{},
		&taskManagerStub{},
	)

	_, err := manager.Create(context.Background(), CreateRequest{AssetID: 7})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestManagerCreateReturnsNotFoundForMissingVersion(t *testing.T) {
	manager := NewManager(
		assetReaderStub{detail: assetdomain.Asset{ID: 7, Version: 3, Type: assetdomain.AssetTypeObject}},
		managerResolverStub{},
		&artifactStoreStub{},
		&taskManagerStub{},
	)

	_, err := manager.Create(context.Background(), CreateRequest{AssetID: 7, Version: 2})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
