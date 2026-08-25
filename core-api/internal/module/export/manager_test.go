package export

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type assetReaderStub struct {
	detail     assetdomain.Asset
	records    []assetdomain.AssetRecord
	detailErr  error
	historyErr error
}

func (s assetReaderStub) GetDetail(context.Context, uint) (assetdomain.Asset, error) {
	return s.detail, s.detailErr
}
func (s assetReaderStub) GetRecordHistory(context.Context, uint) ([]assetdomain.AssetRecord, error) {
	return s.records, s.historyErr
}

type taskManagerStub struct {
	task       *taskdomain.Task
	publishErr error
	getErr     error
}

func (s *taskManagerStub) Publish(_ context.Context, task *taskdomain.Task) (uint, error) {
	if s.publishErr != nil {
		return 0, s.publishErr
	}
	task.ID = 42
	s.task = task
	return task.ID, nil
}
func (s *taskManagerStub) GetDetail(context.Context, uint) (*taskdomain.Task, error) {
	return s.task, s.getErr
}

type artifactStoreStub struct {
	data       []byte
	key        string
	putErr     error
	resolveErr error
}

func (s *artifactStoreStub) PutArtifact(_ context.Context, key, _ string, data []byte) error {
	s.key, s.data = key, append([]byte(nil), data...)
	return s.putErr
}
func (s *artifactStoreStub) ResolveReference(_ context.Context, key string) (string, error) {
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
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

func TestManagerCreateRejectsInvalidDependenciesAndPublishesErrors(t *testing.T) {
	validAssets := assetReaderStub{detail: assetdomain.Asset{ID: 1, Type: assetdomain.AssetTypeObject, Version: 1}}
	validTasks := &taskManagerStub{}
	validArtifacts := &artifactStoreStub{}
	validResolver := managerResolverStub{}

	tests := []struct {
		name    string
		manager Manager
		request CreateRequest
		want    error
	}{
		{name: "zero asset id", manager: NewManager(validAssets, validResolver, validArtifacts, validTasks), request: CreateRequest{}, want: ErrInvalidRequest},
		{name: "missing asset reader", manager: NewManager(nil, validResolver, validArtifacts, validTasks), request: CreateRequest{AssetID: 1}, want: ErrInvalidRequest},
		{name: "missing reference resolver", manager: NewManager(validAssets, nil, validArtifacts, validTasks), request: CreateRequest{AssetID: 1}, want: ErrInvalidRequest},
		{name: "missing artifact store", manager: NewManager(validAssets, validResolver, nil, validTasks), request: CreateRequest{AssetID: 1}, want: ErrInvalidRequest},
		{name: "missing task manager", manager: NewManager(validAssets, validResolver, validArtifacts, nil), request: CreateRequest{AssetID: 1}, want: ErrInvalidRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.manager.Create(context.Background(), test.request)
			if !errors.Is(err, test.want) {
				t.Fatalf("Create error = %v, want %v", err, test.want)
			}
		})
	}

	publishErr := errors.New("queue unavailable")
	manager := NewManager(validAssets, validResolver, validArtifacts, &taskManagerStub{publishErr: publishErr})
	if _, err := manager.Create(context.Background(), CreateRequest{AssetID: 1}); !errors.Is(err, publishErr) {
		t.Fatalf("Create error = %v, want %v", err, publishErr)
	}

	assetErr := errors.New("asset reader unavailable")
	manager = NewManager(assetReaderStub{detailErr: assetErr}, validResolver, validArtifacts, validTasks)
	if _, err := manager.Create(context.Background(), CreateRequest{AssetID: 1}); !errors.Is(err, assetErr) {
		t.Fatalf("Create error = %v, want %v", err, assetErr)
	}

	manager = NewManager(assetReaderStub{detail: assetdomain.Asset{ID: 1, Type: assetdomain.AssetTypeAudio}}, validResolver, validArtifacts, validTasks)
	if _, err := manager.Create(context.Background(), CreateRequest{AssetID: 1}); !errors.Is(err, ErrUnsupportedAsset) {
		t.Fatalf("Create error = %v, want %v", err, ErrUnsupportedAsset)
	}
}

func TestManagerGetValidationAndDecodingErrors(t *testing.T) {
	if _, err := NewManager(nil, nil, nil, nil).Get(context.Background(), 0); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("zero export id error = %v", err)
	}
	if _, err := NewManager(nil, nil, nil, nil).Get(context.Background(), 1); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("nil task manager error = %v", err)
	}

	getErr := errors.New("task unavailable")
	manager := NewManager(nil, nil, nil, &taskManagerStub{getErr: getErr})
	if _, err := manager.Get(context.Background(), 1); !errors.Is(err, getErr) {
		t.Fatalf("task lookup error = %v, want %v", err, getErr)
	}

	for _, task := range []*taskdomain.Task{
		nil,
		{Type: "other", Payload: json.RawMessage(`{}`)},
		{Type: string(TaskType), Payload: json.RawMessage(`not-json`)},
	} {
		manager = NewManager(nil, nil, nil, &taskManagerStub{task: task})
		_, err := manager.Get(context.Background(), 1)
		if err == nil {
			t.Fatalf("Get(%+v) unexpectedly succeeded", task)
		}
		if task != nil && task.Type != string(TaskType) && !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("wrong task type error = %v", err)
		}
	}

	payload, err := json.Marshal(taskPayload{Snapshot: Snapshot{AssetID: 7, Version: 2}})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []json.RawMessage{json.RawMessage(`not-json`)} {
		task := &taskdomain.Task{Type: string(TaskType), Payload: payload, Result: result}
		manager = NewManager(nil, nil, nil, &taskManagerStub{task: task})
		if _, err := manager.Get(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "decode task result") {
			t.Fatalf("result decode error = %v", err)
		}
	}
}

func TestManagerGetResolvesArtifactAndReturnsPendingStatus(t *testing.T) {
	payload, err := json.Marshal(taskPayload{Snapshot: Snapshot{AssetID: 7, RecordID: 9, Version: 2}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := json.Marshal(taskResult{FileName: "asset.zip", FileSize: 12, SHA256: "hash", ObjectKey: "exports/7/result.zip", DownloadURL: "stale-url"})
	if err != nil {
		t.Fatal(err)
	}
	task := &taskdomain.Task{Type: string(TaskType), Status: taskdomain.StatusProcessing, Payload: payload, Result: result, Error: ""}
	artifacts := &artifactStoreStub{}
	manager := NewManager(nil, nil, artifacts, &taskManagerStub{task: task})
	response, err := manager.Get(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if response.AssetID != 7 || response.RecordID != 9 || response.Version != 2 || response.Status != "processing" || response.DownloadURL != "https://download.example/exports/7/result.zip" || response.FileName != "asset.zip" {
		t.Fatalf("unexpected response: %+v", response)
	}

	task.Result = nil
	response, err = manager.Get(context.Background(), 42)
	if err != nil || response.DownloadURL != "" || response.Status != "processing" {
		t.Fatalf("unexpected pending response: %+v, err=%v", response, err)
	}

	resolveErr := errors.New("signing unavailable")
	artifacts.resolveErr = resolveErr
	task.Result = result
	if _, err := manager.Get(context.Background(), 42); !errors.Is(err, resolveErr) {
		t.Fatalf("resolve error = %v, want %v", err, resolveErr)
	}
}

func TestManagerHandleValidationAndArtifactErrors(t *testing.T) {
	manager := NewManager(nil, managerResolverStub{}, &artifactStoreStub{}, &taskManagerStub{})
	if _, err := manager.Handle(context.Background(), nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("nil task error = %v", err)
	}
	if _, err := manager.Handle(context.Background(), &taskdomain.Task{Payload: json.RawMessage(`not-json`)}); err == nil || !strings.Contains(err.Error(), "decode payload") {
		t.Fatalf("payload decode error = %v", err)
	}

	payload, err := json.Marshal(taskPayload{Snapshot: Snapshot{AssetID: 1, Version: 1, Type: assetdomain.AssetTypeAudio}, ObjectKey: "exports/1/result.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Handle(context.Background(), &taskdomain.Task{Payload: payload}); !errors.Is(err, ErrUnsupportedAsset) {
		t.Fatalf("unsupported asset error = %v", err)
	}

	payload, err = json.Marshal(taskPayload{Snapshot: Snapshot{AssetID: 1, Version: 1, Type: assetdomain.AssetTypeObject}, ObjectKey: "exports/1/result.zip"})
	if err != nil {
		t.Fatal(err)
	}
	putErr := errors.New("storage write failed")
	artifacts := &artifactStoreStub{putErr: putErr}
	manager = NewManager(nil, managerResolverStub{}, artifacts, &taskManagerStub{})
	if _, err := manager.Handle(context.Background(), &taskdomain.Task{Payload: payload}); !errors.Is(err, putErr) {
		t.Fatalf("put artifact error = %v, want %v", err, putErr)
	}

	resolveErr := errors.New("artifact URL unavailable")
	artifacts = &artifactStoreStub{resolveErr: resolveErr}
	manager = NewManager(nil, managerResolverStub{}, artifacts, &taskManagerStub{})
	if _, err := manager.Handle(context.Background(), &taskdomain.Task{Payload: payload}); !errors.Is(err, resolveErr) {
		t.Fatalf("resolve artifact error = %v, want %v", err, resolveErr)
	}
}

func TestManagerSnapshotCurrentAndHistoryErrors(t *testing.T) {
	current := assetdomain.Asset{ID: 7, ProjectID: 8, Version: 3, Type: assetdomain.AssetTypeObject, Name: "Current", Content: json.RawMessage(`{"prototype":[]}`)}
	managerUseCase := NewManager(assetReaderStub{detail: current}, managerResolverStub{}, &artifactStoreStub{}, &taskManagerStub{})
	concrete := managerUseCase.(*manager)
	concrete.now = func() time.Time { return time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC) }
	snapshot, err := concrete.snapshot(context.Background(), 7, 0)
	if err != nil || snapshot.Version != 3 || snapshot.RecordID != 0 || string(snapshot.Content) != string(current.Content) {
		t.Fatalf("current snapshot = %+v, err=%v", snapshot, err)
	}

	historyErr := errors.New("history unavailable")
	concrete.assets = assetReaderStub{detail: current, historyErr: historyErr}
	if _, err := concrete.snapshot(context.Background(), 7, 2); !errors.Is(err, historyErr) {
		t.Fatalf("history error = %v, want %v", err, historyErr)
	}
}
