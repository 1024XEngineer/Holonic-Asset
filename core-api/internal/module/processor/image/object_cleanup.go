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
// and never edits the largest connected foreground component. Optional smooth
// reference coverage protects small detached parts that were present strongly
// in the supersampled source; only unsupported hard-alpha specks are removed.
// It remains variadic so focused callers can exercise the geometry-only pass.
func removeIsolatedAlphaComponents(img *image.NRGBA, maxPixels int, references ...*image.NRGBA) {
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
	var reference *image.NRGBA
	if len(references) > 0 {
		reference = references[0]
	}
	for _, component := range components {
		if len(component) > maxPixels || len(component) == largest {
			continue
		}
		if detachedComponentHasStrongSourceSupport(component, reference) {
			continue
		}
		for _, point := range component {
			img.SetNRGBA(point.X, point.Y, color.NRGBA{})
		}
	}
}

func detachedComponentHasStrongSourceSupport(component []image.Point, reference *image.NRGBA) bool {
	if reference == nil {
		return false
	}
	for _, point := range component {
		if point.In(reference.Bounds()) && reference.NRGBAAt(point.X, point.Y).A > weakAlphaEdgeCeiling {
			return true
		}
	}
	return false
}

const weakAlphaEdgeCeiling = uint8(144)

// removeWeakAlphaEdgePixels removes only terminal silhouette pixels whose
// original area-resampled coverage was barely above the hard-alpha cutoff.
// These are usually antialias tips that become solid square protrusions after
// thresholding. Strong source pixels and pixels with two or more cardinal
// connections are preserved, so the pass cannot thin ordinary one-pixel lines
// or rewrite model-generated structure with full alpha evidence.
func removeWeakAlphaEdgePixels(img, smoothReference *image.NRGBA) {
	if img == nil || smoothReference == nil {
		return
	}
	bounds := img.Bounds().Intersect(smoothReference.Bounds())
	if bounds.Empty() {
		return
	}
	snapshot := cloneNRGBA(img)
	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	neighbors := [...]image.Point{
		{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1},
		{X: -1}, {X: 1},
		{X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1},
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if snapshot.NRGBAAt(x, y).A <= TransparentAlphaMax {
				continue
			}
			referenceAlpha := smoothReference.NRGBAAt(x, y).A
			if referenceAlpha < hardAlphaThreshold || referenceAlpha > weakAlphaEdgeCeiling {
				continue
			}

			point := image.Pt(x, y)
			cardinalCount := 0
			for _, offset := range cardinal {
				neighbor := point.Add(offset)
				if neighbor.In(bounds) && snapshot.NRGBAAt(neighbor.X, neighbor.Y).A > TransparentAlphaMax {
					cardinalCount++
				}
			}
			if cardinalCount > 1 {
				continue
			}
			opaqueNeighbors := 0
			for _, offset := range neighbors {
				neighbor := point.Add(offset)
				if neighbor.In(bounds) && snapshot.NRGBAAt(neighbor.X, neighbor.Y).A > TransparentAlphaMax {
					opaqueNeighbors++
				}
			}
			if opaqueNeighbors <= 3 {
				img.SetNRGBA(x, y, color.NRGBA{})
			}
		}
	}
}
