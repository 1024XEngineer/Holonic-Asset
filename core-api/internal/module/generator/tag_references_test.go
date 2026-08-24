package generator

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type tagAssetReaderStub struct {
	assets    []assetdomain.Asset
	err       error
	projectID uint
}

type tagReferenceStoreStub struct {
	persisted []string
}

func (s *tagReferenceStoreStub) ResolveReference(_ context.Context, reference string) (string, error) {
	return reference, nil
}

func (s *tagReferenceStoreStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persisted = append(s.persisted, reference)
	return reference, nil
}

func (*tagReferenceStoreStub) NewObjectKey(string) (string, error) { return "generated.png", nil }

func (*tagReferenceStoreStub) PersistReferenceAt(context.Context, string, string) error { return nil }

func (*tagReferenceStoreStub) DeleteObjects(context.Context, []string) error { return nil }

func (s *tagAssetReaderStub) GetAssets(
	_ context.Context,
	projectID uint,
	_ assetdomain.AssetListFilter,
) ([]assetdomain.Asset, error) {
	s.projectID = projectID
	return s.assets, s.err
}

func TestBuildPrototypePayloadAcceptsLegacyAndStructuredTags(t *testing.T) {
	request := &Request{
		ProjectID:     42,
		Kind:          GenerateCharacterProtoType,
		CreativeBrief: "armored hero",
		Parameters: json.RawMessage(`{
			"asset_name":"hero",
			"dimensions":{"width":64,"height":64},
			"perspective":"Top-Down",
			"tags":["knight",{"name":"player","description":"controllable","color":"#123456"}]
		}`),
	}

	value, err := buildTaskPayload(request)
	if err != nil {
		t.Fatalf("build task payload: %v", err)
	}
	payload, ok := value.(CreateCharacterPrototypePayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", value)
	}
	want := []assetdomain.Tag{
		{Name: "knight", Color: assetdomain.DefaultTagColor},
		{Name: "player", Description: "controllable", Color: "#123456"},
	}
	if !reflect.DeepEqual(payload.Tags, want) {
		t.Fatalf("unexpected tags: got %+v want %+v", payload.Tags, want)
	}
}

func TestSelectNexusReferencesRanksOverlapThenVersionThenAssetID(t *testing.T) {
	assets := &tagAssetReaderStub{assets: []assetdomain.Asset{
		{ID: 1, ProjectID: 42, Version: 8, ThumbnailURL: "refs/one.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 2, ProjectID: 42, Version: 3, ThumbnailURL: "refs/two.png", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 4, ProjectID: 42, Version: 3, ThumbnailURL: "refs/four.png", Tags: []assetdomain.Tag{{Name: "KNIGHT"}, {Name: "player"}}},
		{ID: 7, ProjectID: 42, Version: 4, ThumbnailURL: "refs/four.png", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 5, ProjectID: 42, Version: 9, ThumbnailURL: "", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 6, ProjectID: 42, Version: 9, ThumbnailURL: "refs/unmatched.png", Tags: []assetdomain.Tag{{Name: "villager"}}},
		{ID: 8, ProjectID: 99, Version: 99, ThumbnailURL: "refs/cross-project.png", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
	}}
	engine := &Engine{assets: assets}

	got, err := engine.selectNexusReferences(context.Background(), 42, []assetdomain.Tag{
		{Name: " knight "},
		{Name: "player"},
		{Name: "player"},
	}, 3)
	if err != nil {
		t.Fatalf("select Nexus References: %v", err)
	}
	want := []string{"refs/four.png", "refs/two.png", "refs/one.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected references: got %v want %v", got, want)
	}
	if assets.projectID != 42 {
		t.Fatalf("unexpected project ID %d", assets.projectID)
	}

	got, err = engine.selectNexusReferences(
		context.Background(),
		42,
		[]assetdomain.Tag{{Name: "knight"}, {Name: "player"}},
		2,
		"refs/four.png",
	)
	if err != nil {
		t.Fatalf("select Nexus References with exclusion: %v", err)
	}
	if want := []string{"refs/two.png", "refs/one.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected references after exclusion: got %v want %v", got, want)
	}
}

func TestSelectNexusReferencesRequiresReaderAndPropagatesListFailure(t *testing.T) {
	tags := []assetdomain.Tag{{Name: "knight"}}
	if _, err := (&Engine{}).selectNexusReferences(context.Background(), 42, tags, 3); !errors.Is(err, ErrAssetReaderRequired) {
		t.Fatalf("expected asset reader error, got %v", err)
	}

	wantErr := errors.New("list failed")
	engine := &Engine{assets: &tagAssetReaderStub{err: wantErr}}
	if _, err := engine.selectNexusReferences(context.Background(), 42, tags, 3); !errors.Is(err, wantErr) {
		t.Fatalf("expected list error %v, got %v", wantErr, err)
	}
}

func TestPreparePrototypePayloadPersistsOnlyAdaptiveNexusReferenceLimit(t *testing.T) {
	assets := &tagAssetReaderStub{assets: []assetdomain.Asset{
		{ID: 1, Version: 3, ThumbnailURL: "refs/one.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 2, Version: 2, ThumbnailURL: "refs/two.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 3, Version: 1, ThumbnailURL: "refs/three.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
	}}
	references := &tagReferenceStoreStub{}
	engine := &Engine{assets: assets, references: references}

	prepared, err := engine.prepareTaskPayload(context.Background(), 42, CreateCharacterPrototypePayload{
		CreatingReference: "user.png",
		Tags:              []assetdomain.Tag{{Name: "knight"}},
	})
	if err != nil {
		t.Fatalf("prepare task payload: %v", err)
	}
	payload := prepared.(CreateCharacterPrototypePayload)
	if want := []string{"refs/one.png", "refs/two.png"}; !reflect.DeepEqual(payload.NexusReferences, want) {
		t.Fatalf("unexpected Nexus References: got %v want %v", payload.NexusReferences, want)
	}
	if want := []string{"user.png", "refs/one.png", "refs/two.png"}; !reflect.DeepEqual(references.persisted, want) {
		t.Fatalf("unexpected persisted references: got %v want %v", references.persisted, want)
	}
}

func TestPrototypeReferenceInputsPreservesPriorityAndCapsAtFive(t *testing.T) {
	got, state := prototypeReferenceInputs(
		"project.png",
		"user.png",
		[]string{"tag-1.png", "", "tag-2.png", "tag-3.png", "tag-4.png"},
	)
	want := []string{"project.png", "user.png", "tag-1.png", "tag-2.png", "tag-3.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ordered references: got %v want %v", got, want)
	}
	if !state.HasProjectReference || !state.HasCreatingReference || state.NexusReferenceCount != 3 {
		t.Fatalf("unexpected reference state: %+v", state)
	}
}

func TestNewPrototypeAssetKeepsStructuredTags(t *testing.T) {
	tags := []assetdomain.Tag{{Name: "knight", Description: "armored", Color: "#123456"}}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeCharacter,
		"hero",
		42,
		"brief",
		tags,
		assetdomain.PerspectiveTopDown,
		assetdomain.Size{Width: 64, Height: 64},
		4,
		nil,
	)
	if err != nil {
		t.Fatalf("create prototype asset: %v", err)
	}
	if !reflect.DeepEqual(value.Tags, tags) {
		t.Fatalf("unexpected asset tags: got %+v want %+v", value.Tags, tags)
	}
}

func TestBuildObjectPrototypePayloadAcceptsTags(t *testing.T) {
	request := &Request{
		ProjectID:     42,
		Kind:          GenerateObjectProtoType,
		CreativeBrief: "ancient artifact",
		Parameters: json.RawMessage(`{
			"asset_name":"artifact",
			"dimensions":{"width":64,"height":64},
			"perspective":"Side-On",
			"tags":["magic",{"name":"relic","description":"holy item","color":"#ABCDEF"}]
		}`),
	}

	value, err := buildTaskPayload(request)
	if err != nil {
		t.Fatalf("build task payload: %v", err)
	}
	payload, ok := value.(CreateObjectPrototypePayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", value)
	}
	want := []assetdomain.Tag{
		{Name: "magic", Color: assetdomain.DefaultTagColor},
		{Name: "relic", Description: "holy item", Color: "#ABCDEF"},
	}
	if !reflect.DeepEqual(payload.Tags, want) {
		t.Fatalf("unexpected tags: got %+v want %+v", payload.Tags, want)
	}
}

func TestPrepareObjectPrototypePayloadPersistsNexusReferences(t *testing.T) {
	assets := &tagAssetReaderStub{assets: []assetdomain.Asset{
		{ID: 1, Version: 2, ThumbnailURL: "refs/relic1.png", Tags: []assetdomain.Tag{{Name: "magic"}}},
		{ID: 2, Version: 1, ThumbnailURL: "refs/relic2.png", Tags: []assetdomain.Tag{{Name: "magic"}}},
	}}
	references := &tagReferenceStoreStub{}
	engine := &Engine{assets: assets, references: references}

	prepared, err := engine.prepareTaskPayload(context.Background(), 42, CreateObjectPrototypePayload{
		CreatingReference: "user_relic.png",
		Tags:              []assetdomain.Tag{{Name: "magic"}},
	})
	if err != nil {
		t.Fatalf("prepare task payload: %v", err)
	}
	payload := prepared.(CreateObjectPrototypePayload)
	if want := []string{"refs/relic1.png", "refs/relic2.png"}; !reflect.DeepEqual(payload.NexusReferences, want) {
		t.Fatalf("unexpected Nexus References: got %v want %v", payload.NexusReferences, want)
	}
	if want := []string{"user_relic.png", "refs/relic1.png", "refs/relic2.png"}; !reflect.DeepEqual(references.persisted, want) {
		t.Fatalf("unexpected persisted references: got %v want %v", references.persisted, want)
	}
}

func TestSelectNexusReferencesZeroAndEdgeInputs(t *testing.T) {
	engine := &Engine{assets: &tagAssetReaderStub{}}

	// projectID == 0
	got, err := engine.selectNexusReferences(context.Background(), 0, []assetdomain.Tag{{Name: "tag"}}, 3)
	if err != nil || got != nil {
		t.Fatalf("expected nil for zero project ID, got %v, %v", got, err)
	}

	// len(tags) == 0
	got, err = engine.selectNexusReferences(context.Background(), 42, nil, 3)
	if err != nil || got != nil {
		t.Fatalf("expected nil for empty tags, got %v, %v", got, err)
	}

	// blank tag names only
	got, err = engine.selectNexusReferences(context.Background(), 42, []assetdomain.Tag{{Name: "  "}, {Name: ""}}, 3)
	if err != nil || got != nil {
		t.Fatalf("expected nil for blank tag names, got %v, %v", got, err)
	}

	// limit <= 0
	got, err = engine.selectNexusReferences(context.Background(), 42, []assetdomain.Tag{{Name: "tag"}}, 0)
	if err != nil || got != nil {
		t.Fatalf("expected nil for zero limit, got %v, %v", got, err)
	}
}

var _ AssetReader = (*tagAssetReaderStub)(nil)
var _ ReferenceStore = (*tagReferenceStoreStub)(nil)
