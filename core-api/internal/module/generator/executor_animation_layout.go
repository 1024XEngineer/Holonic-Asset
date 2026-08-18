package generator

import "math"

// animationGridColumns picks the most square grid possible. For non-square
// frame counts the final row may have unused cells, but only real frames are
// emitted and persisted.
func animationGridColumns(frameCount int) int {
	if frameCount <= 1 {
		return 1
	}
	columns := int(math.Ceil(math.Sqrt(float64(frameCount))))
	if columns > 8 {
		return 8
	}
	return columns
}
