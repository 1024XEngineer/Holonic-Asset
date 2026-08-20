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

func TestSelectTagReferencesRanksOverlapThenVersionThenAssetID(t *testing.T) {
	assets := &tagAssetReaderStub{assets: []assetdomain.Asset{
		{ID: 1, Version: 8, ThumbnailURL: "refs/one.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 2, Version: 3, ThumbnailURL: "refs/two.png", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 4, Version: 3, ThumbnailURL: "refs/four.png", Tags: []assetdomain.Tag{{Name: "KNIGHT"}, {Name: "player"}}},
		{ID: 7, Version: 4, ThumbnailURL: "refs/four.png", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 5, Version: 9, ThumbnailURL: "", Tags: []assetdomain.Tag{{Name: "knight"}, {Name: "player"}}},
		{ID: 6, Version: 9, ThumbnailURL: "refs/unmatched.png", Tags: []assetdomain.Tag{{Name: "villager"}}},
	}}
	engine := &Engine{assets: assets}

	got, err := engine.selectTagReferences(context.Background(), 42, []assetdomain.Tag{
		{Name: " knight "},
		{Name: "player"},
		{Name: "player"},
	}, 3)
	if err != nil {
		t.Fatalf("select tag references: %v", err)
	}
	want := []string{"refs/four.png", "refs/two.png", "refs/one.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected references: got %v want %v", got, want)
	}
	if assets.projectID != 42 {
		t.Fatalf("unexpected project ID %d", assets.projectID)
	}

	got, err = engine.selectTagReferences(
		context.Background(),
		42,
		[]assetdomain.Tag{{Name: "knight"}, {Name: "player"}},
		2,
		"refs/four.png",
	)
	if err != nil {
		t.Fatalf("select tag references with exclusion: %v", err)
	}
	if want := []string{"refs/two.png", "refs/one.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected references after exclusion: got %v want %v", got, want)
	}
}

func TestSelectTagReferencesRequiresReaderAndPropagatesListFailure(t *testing.T) {
	tags := []assetdomain.Tag{{Name: "knight"}}
	if _, err := (&Engine{}).selectTagReferences(context.Background(), 42, tags, 3); !errors.Is(err, ErrAssetReaderRequired) {
		t.Fatalf("expected asset reader error, got %v", err)
	}

	wantErr := errors.New("list failed")
	engine := &Engine{assets: &tagAssetReaderStub{err: wantErr}}
	if _, err := engine.selectTagReferences(context.Background(), 42, tags, 3); !errors.Is(err, wantErr) {
		t.Fatalf("expected list error %v, got %v", wantErr, err)
	}
}

func TestPreparePrototypePayloadPersistsOnlyAdaptiveTagReferenceLimit(t *testing.T) {
	assets := &tagAssetReaderStub{assets: []assetdomain.Asset{
		{ID: 1, Version: 3, ThumbnailURL: "refs/one.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 2, Version: 2, ThumbnailURL: "refs/two.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
		{ID: 3, Version: 1, ThumbnailURL: "refs/three.png", Tags: []assetdomain.Tag{{Name: "knight"}}},
	}}
	references := &tagReferenceStoreStub{}
	engine := &Engine{assets: assets, references: references}

	prepared, err := engine.prepareTaskPayload(context.Background(), 42, CreateCharacterPrototypePayload{
		Reference: "user.png",
		Tags:      []assetdomain.Tag{{Name: "knight"}},
	})
	if err != nil {
		t.Fatalf("prepare task payload: %v", err)
	}
	payload := prepared.(CreateCharacterPrototypePayload)
	if want := []string{"refs/one.png", "refs/two.png"}; !reflect.DeepEqual(payload.TagReferences, want) {
		t.Fatalf("unexpected tag references: got %v want %v", payload.TagReferences, want)
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
	if !state.HasProjectReference || !state.HasUserReference || state.TagReferenceCount != 3 {
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

var _ AssetReader = (*tagAssetReaderStub)(nil)
var _ ReferenceStore = (*tagReferenceStoreStub)(nil)
