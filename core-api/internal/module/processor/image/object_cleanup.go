package image

import (
	"image"
	"image/color"
)

// maxIsolatedComponentPixels returns the largest detached foreground island
// that is safe to treat as a quantization speck on an object prototype. Object
// props normally have one connected silhouette; keeping this limit small avoids
// deleting legitimate separated parts while removing isolated pixels produced
// by hard-alpha thresholding.
func maxIsolatedComponentPixels(bounds image.Rectangle) int {
	shortEdge := min(bounds.Dx(), bounds.Dy())
	switch {
	case shortEdge <= 32:
		return 2
	case shortEdge <= 64:
		return 3
	default:
		return 4
	}
}

// removeIsolatedAlphaComponents removes only small disconnected opaque islands
// and never edits the largest connected foreground component. It runs after
// hard-alpha cleanup, so semi-transparent source pixels cannot be promoted by
// this pass and intentional holes in the main silhouette remain untouched.
func removeIsolatedAlphaComponents(img *image.NRGBA, maxPixels int) {
	if img == nil || maxPixels <= 0 {
		return
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return
	}
	visited := make([]bool, bounds.Dx()*bounds.Dy())
	index := func(point image.Point) int {
		return (point.Y-bounds.Min.Y)*bounds.Dx() + point.X - bounds.Min.X
	}
	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	components := make([][]image.Point, 0)
	largest := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			start := image.Pt(x, y)
			if visited[index(start)] || img.NRGBAAt(x, y).A <= TransparentAlphaMax {
				continue
			}
			queue := []image.Point{start}
			visited[index(start)] = true
			for cursor := 0; cursor < len(queue); cursor++ {
				point := queue[cursor]
				for _, offset := range cardinal {
					neighbor := point.Add(offset)
					if !neighbor.In(bounds) || visited[index(neighbor)] ||
						img.NRGBAAt(neighbor.X, neighbor.Y).A <= TransparentAlphaMax {
						continue
					}
					visited[index(neighbor)] = true
					queue = append(queue, neighbor)
				}
			}
			if len(queue) > largest {
				largest = len(queue)
			}
			components = append(components, queue)
		}
	}

	if len(components) <= 1 || largest == 0 {
		return
	}
	for _, component := range components {
		if len(component) > maxPixels || len(component) == largest {
			continue
		}
		for _, point := range component {
			img.SetNRGBA(point.X, point.Y, color.NRGBA{})
		}
	}
}
