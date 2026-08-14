package repository

import (
	"bytes"
	"strings"
	"testing"
)

func TestMigrateAnimationGenerationEdgeCases(t *testing.T) {
	const replacement = `{"animations":[{"id":1,"name":"replacement"}],"custom":true}`

	tests := []struct {
		name        string
		current     string
		replacement string
		want        string
		wantErr     string
	}{
		{name: "empty current", current: "", replacement: replacement, want: replacement},
		{name: "empty replacement", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: "", want: ""},
		{name: "invalid replacement", current: `{}`, replacement: `{`, wantErr: "decode replacement asset content"},
		{name: "null replacement", current: `{}`, replacement: `null`, want: `null`},
		{name: "replacement without animations", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"prototype":[]}`, want: `{"prototype":[]}`},
		{name: "null replacement animations", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":null}`, want: `{"animations":null}`},
		{name: "invalid current", current: `{`, replacement: replacement, wantErr: "decode current asset content"},
		{name: "null current", current: `null`, replacement: replacement, want: replacement},
		{name: "current without animations", current: `{}`, replacement: replacement, want: replacement},
		{name: "null current animations", current: `{"animations":null}`, replacement: replacement, want: replacement},
		{name: "invalid current animations", current: `{"animations":{}}`, replacement: replacement, wantErr: "decode current asset animations"},
		{name: "invalid current animation", current: `{"animations":[null]}`, replacement: replacement, wantErr: "decode current asset animation 0"},
		{name: "current animation without id", current: `{"animations":[{"generation":{"fps":12}}]}`, replacement: replacement, want: replacement},
		{name: "current animation with zero id", current: `{"animations":[{"id":0,"generation":{"fps":12}}]}`, replacement: replacement, want: replacement},
		{name: "current animation with invalid id", current: `{"animations":[{"id":"1","generation":{"fps":12}}]}`, replacement: replacement, want: replacement},
		{name: "current animation with null generation", current: `{"animations":[{"id":1,"generation":null}]}`, replacement: replacement, want: replacement},
		{name: "replacement invalid animations", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":{}}`, wantErr: "decode replacement asset animations"},
		{name: "replacement invalid animation", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":[null]}`, wantErr: "decode replacement asset animation 0"},
		{name: "replacement animation without id", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":[{"name":"new"}]}`, want: `{"animations":[{"name":"new"}]}`},
		{name: "replacement animation with invalid id", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":[{"id":"1"}]}`, want: `{"animations":[{"id":"1"}]}`},
		{name: "replacement animation with zero id", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":[{"id":0}]}`, want: `{"animations":[{"id":0}]}`},
		{name: "no matching id", current: `{"animations":[{"id":2,"generation":{"fps":12}}]}`, replacement: replacement, want: replacement},
		{name: "explicit null generation is preserved", current: `{"animations":[{"id":1,"generation":{"fps":12}}]}`, replacement: `{"animations":[{"id":1,"generation":null}]}`, want: `{"animations":[{"id":1,"generation":null}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrateAnimationGeneration([]byte(tt.current), []byte(tt.replacement))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("migrate animation generation: %v", err)
			}
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Fatalf("unexpected content: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMigrateAnimationGenerationOnlyChangesGeneration(t *testing.T) {
	current := []byte(`{"animations":[{"id":1,"generation":{"fps":12,"future":12345678901234567890}}]}`)
	replacement := []byte(`{"animations":[{"id":1,"name":"walk","frames":[],"futureValue":12345678901234567890}]}`)

	got, err := migrateAnimationGeneration(current, replacement)
	if err != nil {
		t.Fatalf("migrate animation generation: %v", err)
	}
	want := `{"animations":[{"frames":[],"futureValue":12345678901234567890,"generation":{"fps":12,"future":12345678901234567890},"id":1,"name":"walk"}]}`
	if string(got) != want {
		t.Fatalf("unexpected migrated content: %s", got)
	}
}

func TestDecodeAssetContentHelpers(t *testing.T) {
	if _, err := decodeAssetContentObject([]byte(`[]`)); err == nil {
		t.Fatal("expected non-object JSON to be rejected")
	}
	if _, err := decodeAssetContentObject([]byte(`null`)); err == nil || !strings.Contains(err.Error(), "expected JSON object") {
		t.Fatalf("expected null object error, got %v", err)
	}

	for _, raw := range []string{"", `"1"`, `0`, `null`} {
		if id, ok := decodeAssetContentID([]byte(raw)); ok || id != 0 {
			t.Fatalf("expected invalid animation ID %q, got %d, %v", raw, id, ok)
		}
	}
	if id, ok := decodeAssetContentID([]byte(`7`)); !ok || id != 7 {
		t.Fatalf("expected valid animation ID, got %d, %v", id, ok)
	}
}
