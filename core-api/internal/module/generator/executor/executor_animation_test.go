package executor

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"

	"encoding/json"
	"strings"
	"testing"
)

func TestAnimationDirectionIndexUsesAssetDirectionLayout(t *testing.T) {
	tests := []struct {
		name           string
		direction      string
		directionCount uint
		want           int
	}{
		{name: "two left", direction: generator.AnimationDirectionLeft, directionCount: 2, want: 0},
		{name: "two right", direction: generator.AnimationDirectionRight, directionCount: 2, want: 1},
		{name: "four right", direction: generator.AnimationDirectionRight, directionCount: 4, want: 1},
		{name: "four left", direction: generator.AnimationDirectionLeft, directionCount: 4, want: 3},
		{name: "eight front right", direction: generator.AnimationDirectionFrontRight, directionCount: 8, want: 1},
		{name: "eight back right", direction: generator.AnimationDirectionBackRight, directionCount: 8, want: 3},
		{name: "eight front left", direction: generator.AnimationDirectionFrontLeft, directionCount: 8, want: 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := animationDirectionIndex(test.direction, test.directionCount)
			if err != nil {
				t.Fatalf("resolve direction: %v", err)
			}
			if got != test.want {
				t.Fatalf("index = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAnimationDirectionIndexRejectsInvalidLayoutsAndNames(t *testing.T) {
	tests := []struct {
		name           string
		direction      string
		directionCount uint
		want           string
	}{
		{name: "missing multi direction", directionCount: 8, want: "direction is required"},
		{name: "diagonal unavailable in four directions", direction: generator.AnimationDirectionFrontRight, directionCount: 4, want: "is unavailable"},
		{name: "unknown direction", direction: "up", directionCount: 8, want: "is unavailable"},
		{name: "more than eight", direction: generator.AnimationDirectionFront, directionCount: 9, want: "at most 8 directions"},
		{name: "single direction unsupported", direction: generator.AnimationDirectionFront, directionCount: 1, want: "must be one of 2, 4, or 8"},
		{name: "unsupported count", direction: generator.AnimationDirectionFront, directionCount: 3, want: "must be one of 2, 4, or 8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := animationDirectionIndex(test.direction, test.directionCount)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCreateAnimationPayloadRejectsNumericDirection(t *testing.T) {
	var payload generator.CreateAnimationPayload
	err := json.Unmarshal([]byte(`{"direction":3}`), &payload)
	if err == nil {
		t.Fatal("numeric animation direction should be rejected")
	}
}

func TestAnimationUnprocessedImageURLAddsSuffixWithoutChangingReferenceSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "URL with query",
			value: "https://cdn.example.com/hero/front.png?version=7",
			want:  "https://cdn.example.com/hero/front-unprocessed.png?version=7",
		},
		{name: "object key", value: "uploads/hero/front.png", want: "uploads/hero/front-unprocessed.png"},
		{name: "no extension", value: "uploads/hero/front", want: "uploads/hero/front-unprocessed"},
		{name: "data URL", value: "data:image/png;base64,parent", want: "data:image/png;base64,parent"},
		{name: "blank", value: "  ", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := animationUnprocessedImageURL(test.value); got != test.want {
				t.Fatalf("unprocessed URL = %q, want %q", got, test.want)
			}
		})
	}
}
