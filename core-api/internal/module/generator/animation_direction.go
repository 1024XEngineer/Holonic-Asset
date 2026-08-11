package generator

import (
	"fmt"
	"slices"
	"strings"
)

const (
	AnimationDirectionFront      = "front"
	AnimationDirectionFrontRight = "front_right"
	AnimationDirectionRight      = "right"
	AnimationDirectionBackRight  = "back_right"
	AnimationDirectionBack       = "back"
	AnimationDirectionBackLeft   = "back_left"
	AnimationDirectionLeft       = "left"
	AnimationDirectionFrontLeft  = "front_left"
)

var animationDirectionLayouts = map[uint][]string{
	2: {
		AnimationDirectionLeft,
		AnimationDirectionRight,
	},
	4: {
		AnimationDirectionFront,
		AnimationDirectionRight,
		AnimationDirectionBack,
		AnimationDirectionLeft,
	},
	8: {
		AnimationDirectionFront,
		AnimationDirectionFrontRight,
		AnimationDirectionRight,
		AnimationDirectionBackRight,
		AnimationDirectionBack,
		AnimationDirectionBackLeft,
		AnimationDirectionLeft,
		AnimationDirectionFrontLeft,
	},
}

func animationDirectionIndex(direction string, directionCount uint) (int, error) {
	if directionCount > 8 {
		return 0, fmt.Errorf("generator: animation supports at most 8 directions, asset has %d", directionCount)
	}
	layout, ok := animationDirectionLayouts[directionCount]
	if !ok {
		return 0, fmt.Errorf("generator: animation asset direction count must be one of 2, 4, or 8, got %d", directionCount)
	}

	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		return 0, fmt.Errorf("generator: animation direction is required; available directions: %s", strings.Join(layout, ", "))
	}
	index := slices.Index(layout, direction)
	if index < 0 {
		return 0, fmt.Errorf(
			"generator: animation direction %q is unavailable for an asset with %d directions; available directions: %s",
			direction,
			directionCount,
			strings.Join(layout, ", "),
		)
	}
	return index, nil
}
