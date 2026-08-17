package generator

import (
	"fmt"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

const (
	tileSetNearBlackGuideThreshold       = 20
	minTileSetNearBlackComponentSolidity = 0.80
	tileSetGuideBoundsTolerance          = 1
)

func verifyTileSetNoGuideLeak(imageBase64 string) error {
	img, err := imageprocessor.DecodeBase64Image(imageBase64)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("empty image")
	}
	nearBlack := make([]bool, width*height)
	for y := range height {
		for x := range width {
			pixel := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			nearBlack[y*width+x] = pixel.A > imageprocessor.TransparentAlphaMax &&
				pixel.R <= tileSetNearBlackGuideThreshold &&
				pixel.G <= tileSetNearBlackGuideThreshold &&
				pixel.B <= tileSetNearBlackGuideThreshold
		}
	}
	queue := make([]int, 0, width*height)
	for start := range nearBlack {
		if !nearBlack[start] {
			continue
		}
		nearBlack[start] = false
		queue = append(queue[:0], start)
		size := 0
		minX, minY := start%width, start/width
		maxX, maxY := minX, minY
		for len(queue) > 0 {
			current := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			size++
			x, y := current%width, current/width
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
			neighbors := [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}}
			for _, neighbor := range neighbors {
				nx, ny := neighbor[0], neighbor[1]
				if nx < 0 || ny < 0 || nx >= width || ny >= height {
					continue
				}
				index := ny*width + nx
				if !nearBlack[index] {
					continue
				}
				nearBlack[index] = false
				queue = append(queue, index)
			}
		}
		boundingBoxPixels := (maxX - minX + 1) * (maxY - minY + 1)
		solidity := float64(size) / float64(boundingBoxPixels)
		matchesCellBounds := minX <= tileSetGuideBoundsTolerance && minY <= tileSetGuideBoundsTolerance &&
			maxX >= width-1-tileSetGuideBoundsTolerance && maxY >= height-1-tileSetGuideBoundsTolerance
		if matchesCellBounds && solidity >= minTileSetNearBlackComponentSolidity {
			return fmt.Errorf(
				"near-black occupancy-guide component matches the Tile bounds at %.1f%% solidity",
				solidity*100,
			)
		}
	}
	return nil
}

func formatTileSetProjectContext(project *projectdomain.Project) string {
	return fmt.Sprintf(
		"Name: %s\nGame type: %s\nDescription: %s\nVisual style: %s\nPlatform: %s\nPerspective: %s",
		project.Name,
		project.GameType,
		project.Description,
		project.Style,
		project.TargetPlatform,
		project.Perspective,
	)
}

func tileSetItemBounds(shape []TileSetCoordinate) (int, int, int, int) {
	minX, minY := shape[0][0], shape[0][1]
	maxX, maxY := minX, minY
	for _, coordinate := range shape[1:] {
		minX = min(minX, coordinate[0])
		minY = min(minY, coordinate[1])
		maxX = max(maxX, coordinate[0])
		maxY = max(maxY, coordinate[1])
	}
	return minX, minY, maxX, maxY
}
