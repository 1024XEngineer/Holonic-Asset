package image

import (
	"image"
	"image/color"
	"math"
)

// regularizeNearCircularObjectSilhouette repairs only single-component object
// silhouettes that are already strongly symmetric and close to an ellipse.
// Unlike the conservative generic repair below, it may expand a slightly
// squashed near-circle to a square logical bounding box. This targets round
// props whose high-resolution antialiasing otherwise becomes a pinched or oval
// sprite after hard-alpha reduction; non-round and asymmetric objects are left
// unchanged.
func regularizeNearCircularObjectSilhouette(
	img, smoothReference *image.NRGBA,
	palette []color.RGBA,
) {
	bounds, ok := alphaBounds(img, TransparentAlphaMax)
	if !ok || bounds.Dx() < 8 || bounds.Dy() < 8 || len(palette) == 0 {
		return
	}
	aspect := float64(bounds.Dx()) / float64(bounds.Dy())
	// A generated orthographic/top-down view can be a noticeably pinched
	// ellipse even when it is still a symmetric round prop. The lower bound is
	// deliberately conservative; asymmetric and strongly elongated objects are
	// rejected by the component/symmetry/ellipse checks below.
	if aspect < 0.60 || aspect > 1.67 || opaqueComponentCount(img, bounds) != 1 {
		return
	}

	area := bounds.Dx() * bounds.Dy()
	horizontalMismatch, verticalMismatch := silhouetteSymmetryMismatch(img, bounds)
	symmetryTolerance := max(2, area/16)
	if horizontalMismatch > symmetryTolerance || verticalMismatch > symmetryTolerance {
		return
	}
	if silhouetteEllipseIOU(img, bounds) < 0.88 {
		return
	}

	target := centredSquareBoundsForArea(bounds, img.Bounds(), opaquePixelArea(img, bounds))
	if target.Empty() {
		return
	}
	centreX := float64(target.Min.X+target.Max.X) / 2
	centreY := float64(target.Min.Y+target.Max.Y) / 2
	radius := float64(target.Dx()) / 2
	isCirclePixel := func(x, y int) bool {
		dx := (float64(x) + 0.5 - centreX) / radius
		dy := (float64(y) + 0.5 - centreY) / radius
		return dx*dx+dy*dy <= 1
	}

	region := bounds.Union(target).Intersect(img.Bounds())
	changed, union := 0, 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			current := img.NRGBAAt(x, y).A > TransparentAlphaMax
			ideal := x >= target.Min.X && x < target.Max.X &&
				y >= target.Min.Y && y < target.Max.Y && isCirclePixel(x, y)
			if current || ideal {
				union++
			}
			if current != ideal {
				changed++
			}
		}
	}
	// A strongly symmetric pinched view can need more than a handful of pixels
	// restored, but the target diameter is derived from opaque area so the repair
	// does not make one direction larger than the other views.
	// Keep a guard against turning arbitrary props into circles. A more strongly
	// pinched but still symmetric ellipse needs a larger correction than a mild
	// one, otherwise the common 13x20 logical basketball footprint is rejected
	// before it can be restored to a round silhouette.
	maxChangeRatio := 1.0 / 3.0
	if aspect < 0.75 || aspect > 1.0/0.75 {
		maxChangeRatio = 0.60
	}
	if union == 0 || changed == 0 || float64(changed)/float64(union) > maxChangeRatio {
		return
	}

	snapshot := cloneNRGBA(img)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			ideal := x >= target.Min.X && x < target.Max.X &&
				y >= target.Min.Y && y < target.Max.Y && isCirclePixel(x, y)
			current := snapshot.NRGBAAt(x, y)
			if !ideal {
				if current.A > TransparentAlphaMax {
					img.SetNRGBA(x, y, color.NRGBA{})
				}
				continue
			}
			if current.A > TransparentAlphaMax {
				continue
			}
			fill := ellipseBoundaryFillColour(snapshot, smoothReference, image.Pt(x, y), palette)
			img.SetNRGBA(x, y, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 255})
		}
	}
}

func centredSquareBoundsForArea(bounds, canvas image.Rectangle, opaqueArea int) image.Rectangle {
	if opaqueArea <= 0 {
		return image.Rectangle{}
	}
	// Preserve the footprint of a squashed round prop instead of expanding it
	// to the longest axis. A 13x20 ellipse and a 17x17 circle can represent the
	// same object size; using max(width,height) would turn the former into a
	// much larger 20x20 sprite.
	side := max(1, int(math.Ceil(math.Sqrt(float64(opaqueArea)*4/math.Pi))))
	side = max(side, min(bounds.Dx(), bounds.Dy()))
	if side <= 0 || side > canvas.Dx() || side > canvas.Dy() {
		return image.Rectangle{}
	}
	minX := (bounds.Min.X + bounds.Max.X - side) / 2
	minY := (bounds.Min.Y + bounds.Max.Y - side) / 2
	minX = max(canvas.Min.X, min(minX, canvas.Max.X-side))
	minY = max(canvas.Min.Y, min(minY, canvas.Max.Y-side))
	return image.Rect(minX, minY, minX+side, minY+side)
}

func opaquePixelArea(img *image.NRGBA, bounds image.Rectangle) int {
	area := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.NRGBAAt(x, y).A > TransparentAlphaMax {
				area++
			}
		}
	}
	return area
}

func silhouetteSymmetryMismatch(img *image.NRGBA, bounds image.Rectangle) (int, int) {
	horizontalMismatch, verticalMismatch := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			opaque := img.NRGBAAt(x, y).A > TransparentAlphaMax
			mirrorX := bounds.Min.X + bounds.Max.X - 1 - x
			mirrorY := bounds.Min.Y + bounds.Max.Y - 1 - y
			if opaque != (img.NRGBAAt(mirrorX, y).A > TransparentAlphaMax) {
				horizontalMismatch++
			}
			if opaque != (img.NRGBAAt(x, mirrorY).A > TransparentAlphaMax) {
				verticalMismatch++
			}
		}
	}
	return horizontalMismatch, verticalMismatch
}

func silhouetteEllipseIOU(img *image.NRGBA, bounds image.Rectangle) float64 {
	centreX := float64(bounds.Min.X+bounds.Max.X) / 2
	centreY := float64(bounds.Min.Y+bounds.Max.Y) / 2
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	intersection, union := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := (float64(x) + 0.5 - centreX) / radiusX
			dy := (float64(y) + 0.5 - centreY) / radiusY
			ideal := dx*dx+dy*dy <= 1
			current := img.NRGBAAt(x, y).A > TransparentAlphaMax
			if current || ideal {
				union++
			}
			if current && ideal {
				intersection++
			}
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// regularizeNearEllipticalSilhouette snaps only a strongly symmetric,
// already-near-elliptical single component to the mathematical ellipse implied
// by its current bounds. This corrects threshold-induced flat spots on round
// props without rounding arbitrary characters, equipment, or asymmetric
// objects. Existing pixels keep their colours; only the few changed boundary
// pixels are removed or filled from neighbouring palette colours.
func regularizeNearEllipticalSilhouette(
	img, smoothReference *image.NRGBA,
	palette []color.RGBA,
) {
	bounds, ok := alphaBounds(img, TransparentAlphaMax)
	if !ok || bounds.Dx() < 8 || bounds.Dy() < 8 {
		return
	}
	aspect := float64(bounds.Dx()) / float64(bounds.Dy())
	if aspect < 0.9 || aspect > 1.1 || opaqueComponentCount(img, bounds) != 1 {
		return
	}

	area := bounds.Dx() * bounds.Dy()
	symmetryTolerance := max(2, area/32)
	horizontalMismatch, verticalMismatch := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			opaque := img.NRGBAAt(x, y).A > TransparentAlphaMax
			mirrorX := bounds.Min.X + bounds.Max.X - 1 - x
			mirrorY := bounds.Min.Y + bounds.Max.Y - 1 - y
			if opaque != (img.NRGBAAt(mirrorX, y).A > TransparentAlphaMax) {
				horizontalMismatch++
			}
			if opaque != (img.NRGBAAt(x, mirrorY).A > TransparentAlphaMax) {
				verticalMismatch++
			}
		}
	}
	if horizontalMismatch > symmetryTolerance || verticalMismatch > symmetryTolerance {
		return
	}

	centreX := float64(bounds.Min.X+bounds.Max.X) / 2
	centreY := float64(bounds.Min.Y+bounds.Max.Y) / 2
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	isEllipsePixel := func(x, y int) bool {
		dx := (float64(x) + 0.5 - centreX) / radiusX
		dy := (float64(y) + 0.5 - centreY) / radiusY
		return dx*dx+dy*dy <= 1
	}
	intersection, union := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			current := img.NRGBAAt(x, y).A > TransparentAlphaMax
			ideal := isEllipsePixel(x, y)
			if current || ideal {
				union++
			}
			if current && ideal {
				intersection++
			}
		}
	}
	if union == 0 {
		return
	}
	ellipseIOU := float64(intersection) / float64(union)
	if ellipseIOU < 0.94 || ellipseIOU >= 0.985 {
		return
	}

	snapshot := cloneNRGBA(img)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ideal := isEllipsePixel(x, y)
			current := snapshot.NRGBAAt(x, y)
			if !ideal {
				if current.A > TransparentAlphaMax {
					img.SetNRGBA(x, y, color.NRGBA{})
				}
				continue
			}
			if current.A > TransparentAlphaMax {
				continue
			}
			fill := ellipseBoundaryFillColour(snapshot, smoothReference, image.Pt(x, y), palette)
			img.SetNRGBA(x, y, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 255})
		}
	}
}

func opaqueComponentCount(img *image.NRGBA, bounds image.Rectangle) int {
	visited := make([]bool, bounds.Dx()*bounds.Dy())
	index := func(point image.Point) int {
		return (point.Y-bounds.Min.Y)*bounds.Dx() + point.X - bounds.Min.X
	}
	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	components := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			start := image.Pt(x, y)
			if visited[index(start)] || img.NRGBAAt(x, y).A <= TransparentAlphaMax {
				continue
			}
			components++
			if components > 1 {
				return components
			}
			queue := []image.Point{start}
			visited[index(start)] = true
			for len(queue) > 0 {
				point := queue[0]
				queue = queue[1:]
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
		}
	}
	return components
}

func ellipseBoundaryFillColour(
	img, smoothReference *image.NRGBA,
	point image.Point,
	palette []color.RGBA,
) color.RGBA {
	// The area-resampled source is the authority for interior colour. Looking
	// only at neighbours makes newly added boundary pixels inherit an arbitrary
	// outline/highlight colour and is a common cause of stray blocks on props.
	reference := smoothReference.NRGBAAt(point.X, point.Y)
	if reference.A > TransparentAlphaMax {
		best := palette[0]
		bestDistance := perceptualColourDistance(nrgbaToOKLab(reference), rgbaToOKLab(best))
		for _, candidate := range palette[1:] {
			distance := perceptualColourDistance(nrgbaToOKLab(reference), rgbaToOKLab(candidate))
			if distance < bestDistance || (distance == bestDistance && rgbaKey(candidate) < rgbaKey(best)) {
				best, bestDistance = candidate, distance
			}
		}
		return best
	}

	neighbors := [...]image.Point{
		{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1},
		{X: -1}, {X: 1},
		{X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1},
	}
	counts := make(map[pixelColourKey]int)
	for _, offset := range neighbors {
		neighbor := point.Add(offset)
		if !neighbor.In(img.Bounds()) {
			continue
		}
		pixel := img.NRGBAAt(neighbor.X, neighbor.Y)
		if pixel.A > TransparentAlphaMax {
			counts[colourKey(pixel)]++
		}
	}
	if key, count := dominantBoundaryColour(counts); count > 0 {
		candidate := colourFromKey(key)
		return color.RGBA{R: candidate.R, G: candidate.G, B: candidate.B, A: 255}
	}
	best := palette[0]
	bestDistance := math.MaxFloat64
	referenceLab := nrgbaToOKLab(reference)
	for _, candidate := range palette {
		distance := perceptualColourDistance(referenceLab, rgbaToOKLab(candidate))
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}
