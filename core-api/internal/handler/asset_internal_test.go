package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestTransformReferenceFieldSkipsMissingAndNonStringReferences(t *testing.T) {
	transform := func(_ context.Context, reference string) (string, error) {
		return "transformed:" + reference, nil
	}

	if err := transformReferenceField(context.Background(), map[string]json.RawMessage{}, "url", transform); err != nil {
		t.Fatalf("missing reference: %v", err)
	}

	for _, raw := range []string{"", `null`, `123`, `true`, `[]`, `{}`} {
		object := map[string]json.RawMessage{"url": json.RawMessage(raw)}
		if err := transformReferenceField(context.Background(), object, "url", transform); err != nil {
			t.Fatalf("raw reference %q: %v", raw, err)
		}
		if string(object["url"]) != raw {
			t.Fatalf("raw reference %q was changed to %s", raw, object["url"])
		}
	}

	object := map[string]json.RawMessage{"url": json.RawMessage(`"   "`)}
	if err := transformReferenceField(context.Background(), object, "url", transform); err != nil {
		t.Fatalf("blank reference: %v", err)
	}
	if string(object["url"]) != `"   "` {
		t.Fatalf("blank reference was changed to %s", object["url"])
	}
}

func TestTransformReferenceFieldReturnsDecodeError(t *testing.T) {
	object := map[string]json.RawMessage{"url": json.RawMessage(`"unterminated`)}
	err := transformReferenceField(context.Background(), object, "url", func(context.Context, string) (string, error) {
		return "unused", nil
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected end") {
		t.Fatalf("expected reference decode error, got %v", err)
	}
}

func TestIsURLReferenceRecognizesDataAndProtocolRelativeURLs(t *testing.T) {
	for _, reference := range []string{"data:image/png;base64,abc", "DATA:image/png;base64,abc", "//cdn.example/image.png"} {
		if !isURLReference(reference) {
			t.Fatalf("expected URL reference %q", reference)
		}
	}
}

func TestTransformReferenceFieldPropagatesTransformError(t *testing.T) {
	wantErr := errors.New("persist failed")
	object := map[string]json.RawMessage{"url": json.RawMessage(`"uploads/image.png"`)}
	err := transformReferenceField(context.Background(), object, "url", func(context.Context, string) (string, error) {
		return "", wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected transform error %v, got %v", wantErr, err)
	}
}
