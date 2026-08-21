package image

import (
	"image"
	"image/color"
)

const (
	// contourFillAlphaFloor only promotes a transparent logical pixel when the
	// supersampled reduction still contains meaningful foreground coverage. A
	// lower value would turn deliberate one-pixel holes into filled corners.
	contourFillAlphaFloor = uint8(88)
	// contourRemoveAlphaCeiling rejects only weakly supported terminal pixels.
	// Strong source coverage is kept even if the binary mask makes a rough step.
	contourRemoveAlphaCeiling = uint8(192)
)

// regularizePixelContour removes the two most common hard-alpha contour defects:
// a one-pixel convex tooth and a one-pixel concave notch. It deliberately does
// not run a blur or a neighbourhood majority filter. Only pixels on the
// exterior background are eligible for filling, and every decision is backed by
// alpha coverage from the untouched supersampled reduction.
//
// This is a pixel-grid cleanup, not a shape generator. It changes at most one
// logical pixel at a time, keeps all colours inside the existing palette, and
// leaves enclosed holes and high-contrast interior details alone.
func regularizePixelContour(img, smoothReference *image.NRGBA, palette []color.RGBA) {
	if img == nil || smoothReference == nil || len(palette) == 0 {
		return
	}
	bounds := img.Bounds().Intersect(smoothReference.Bounds())
	if bounds.Empty() {
		return
	}

	snapshot := cloneNRGBA(img)
	exterior := exteriorTransparentPixels(snapshot, bounds)
	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	diagonal := [...]image.Point{
		{X: -1, Y: -1}, {X: 1, Y: -1},
		{X: -1, Y: 1}, {X: 1, Y: 1},
	}

	fill := make([]image.Point, 0)
	remove := make([]image.Point, 0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			point := image.Pt(x, y)
			pixel := snapshot.NRGBAAt(x, y)
			if pixel.A <= TransparentAlphaMax {
				if !exterior[contourPointIndex(point, bounds)] {
					continue
				}
				reference := smoothReference.NRGBAAt(x, y)
				if reference.A < contourFillAlphaFloor {
					continue
				}
				cardinalCount := countOpaqueNeighbours(snapshot, point, bounds, cardinal[:])
				if cardinalCount < 3 {
					continue
				}
				fill = append(fill, point)
				continue
			}

			reference := smoothReference.NRGBAAt(x, y)
			if reference.A > contourRemoveAlphaCeiling {
				continue
			}
			cardinalCount := countOpaqueNeighbours(snapshot, point, bounds, cardinal[:])
			if cardinalCount > 1 {
				continue
			}
			// A terminal pixel with no nearby diagonal support is handled by the
			// existing isolated-component pass. Requiring diagonal support here
			// limits this pass to contour teeth rather than deleting tiny props.
			diagonalCount := countOpaqueNeighbours(snapshot, point, bounds, diagonal[:])
			if diagonalCount == 0 {
				continue
			}
			if countExteriorNeighbours(exterior, point, bounds, diagonal[:]) < 2 {
				continue
			}
			remove = append(remove, point)
		}
	}

	for _, point := range fill {
		fillColour := contourFillColour(snapshot, smoothReference, point, palette)
		img.SetNRGBA(point.X, point.Y, color.NRGBA{
			R: fillColour.R, G: fillColour.G, B: fillColour.B, A: 255,
		})
	}
	for _, point := range remove {
		img.SetNRGBA(point.X, point.Y, color.NRGBA{})
	}
	regularizeBoundaryRuns(img, smoothReference, palette)
}

func exteriorTransparentPixels(img *image.NRGBA, bounds image.Rectangle) []bool {
	exterior := make([]bool, bounds.Dx()*bounds.Dy())
	queue := make([]image.Point, 0, bounds.Dx()+bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x != bounds.Min.X && x != bounds.Max.X-1 &&
				y != bounds.Min.Y && y != bounds.Max.Y-1 {
				continue
			}
			point := image.Pt(x, y)
			if img.NRGBAAt(x, y).A <= TransparentAlphaMax {
				index := contourPointIndex(point, bounds)
				if !exterior[index] {
					exterior[index] = true
					queue = append(queue, point)
				}
			}
		}
	}

	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	for cursor := 0; cursor < len(queue); cursor++ {
		point := queue[cursor]
		for _, offset := range cardinal {
			neighbor := point.Add(offset)
			if !neighbor.In(bounds) || img.NRGBAAt(neighbor.X, neighbor.Y).A > TransparentAlphaMax {
				continue
			}
			index := contourPointIndex(neighbor, bounds)
			if exterior[index] {
				continue
			}
			exterior[index] = true
			queue = append(queue, neighbor)
		}
	}
	return exterior
}

func contourPointIndex(point image.Point, bounds image.Rectangle) int {
	return (point.Y-bounds.Min.Y)*bounds.Dx() + point.X - bounds.Min.X
}

func countOpaqueNeighbours(
	img *image.NRGBA,
	point image.Point,
	bounds image.Rectangle,
	offsets []image.Point,
) int {
	count := 0
	for _, offset := range offsets {
		neighbor := point.Add(offset)
		if neighbor.In(bounds) && img.NRGBAAt(neighbor.X, neighbor.Y).A > TransparentAlphaMax {
			count++
		}
	}
	return count
}

func countExteriorNeighbours(
	exterior []bool,
	point image.Point,
	bounds image.Rectangle,
	offsets []image.Point,
) int {
	count := 0
	for _, offset := range offsets {
		neighbor := point.Add(offset)
		if neighbor.In(bounds) && exterior[contourPointIndex(neighbor, bounds)] {
			count++
		}
	}
	return count
}

func contourFillColour(
	img, smoothReference *image.NRGBA,
	point image.Point,
	palette []color.RGBA,
) color.RGBA {
	reference := smoothReference.NRGBAAt(point.X, point.Y)
	if reference.A > TransparentAlphaMax {
		return nearestPaletteColour(reference, palette)
	}

	cardinal := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	counts := make(map[pixelColourKey]int)
	for _, offset := range cardinal {
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
		pixel := colourFromKey(key)
		return color.RGBA{R: pixel.R, G: pixel.G, B: pixel.B, A: 255}
	}
	return nearestPaletteColour(reference, palette)
}

func nearestPaletteColour(reference color.NRGBA, palette []color.RGBA) color.RGBA {
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

// regularizeBoundaryRuns removes isolated one-pixel fluctuations from otherwise
// continuous scanline boundaries. A median of the previous, current, and next
// run is used only when every row/column contains one sufficiently wide run;
// this preserves limbs, holes, and separated accessories. The high-resolution
// reference still has veto power over every add/remove operation.
func regularizeBoundaryRuns(img, smoothReference *image.NRGBA, palette []color.RGBA) {
	bounds := img.Bounds().Intersect(smoothReference.Bounds())
	if bounds.Empty() {
		return
	}
	snapshot := cloneNRGBA(img)
	exterior := exteriorTransparentPixels(snapshot, bounds)
	fill := make([]image.Point, 0)
	remove := make([]image.Point, 0)

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		previous, previousOK := singleOpaqueRowSpan(snapshot, bounds, y-1)
		current, currentOK := singleOpaqueRowSpan(snapshot, bounds, y)
		next, nextOK := singleOpaqueRowSpan(snapshot, bounds, y+1)
		if !previousOK || !currentOK || !nextOK ||
			min(previous.width(), current.width(), next.width()) < 3 {
			continue
		}
		left := median3(previous.min, current.min, next.min)
		right := median3(previous.max, current.max, next.max)
		queueBoundaryShift(snapshot, smoothReference, exterior, bounds, palette,
			current.min, left, current.max, true, y, &fill, &remove)
		queueBoundaryShift(snapshot, smoothReference, exterior, bounds, palette,
			current.max, right, current.min, false, y, &fill, &remove)
	}

	for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
		previous, previousOK := singleOpaqueColumnSpan(snapshot, bounds, x-1)
		current, currentOK := singleOpaqueColumnSpan(snapshot, bounds, x)
		next, nextOK := singleOpaqueColumnSpan(snapshot, bounds, x+1)
		if !previousOK || !currentOK || !nextOK ||
			min(previous.width(), current.width(), next.width()) < 3 {
			continue
		}
		top := median3(previous.min, current.min, next.min)
		bottom := median3(previous.max, current.max, next.max)
		queueBoundaryShift(snapshot, smoothReference, exterior, bounds, palette,
			current.min, top, current.max, true, x, &fill, &remove)
		queueBoundaryShift(snapshot, smoothReference, exterior, bounds, palette,
			current.max, bottom, current.min, false, x, &fill, &remove)
	}

	for _, point := range fill {
		fillColour := contourFillColour(snapshot, smoothReference, point, palette)
		img.SetNRGBA(point.X, point.Y, color.NRGBA{
			R: fillColour.R, G: fillColour.G, B: fillColour.B, A: 255,
		})
	}
	for _, point := range remove {
		img.SetNRGBA(point.X, point.Y, color.NRGBA{})
	}
}

type opaqueSpan struct {
	min int
	max int
}

func (span opaqueSpan) width() int {
	return span.max - span.min + 1
}

func singleOpaqueRowSpan(img *image.NRGBA, bounds image.Rectangle, y int) (opaqueSpan, bool) {
	return singleOpaqueSpan(img, bounds, y, true)
}

func singleOpaqueColumnSpan(img *image.NRGBA, bounds image.Rectangle, x int) (opaqueSpan, bool) {
	return singleOpaqueSpan(img, bounds, x, false)
}

func singleOpaqueSpan(img *image.NRGBA, bounds image.Rectangle, line int, horizontal bool) (opaqueSpan, bool) {
	lineMin, lineMax := bounds.Min.X, bounds.Max.X
	if !horizontal {
		lineMin, lineMax = bounds.Min.Y, bounds.Max.Y
	}
	runs := 0
	inRun := false
	span := opaqueSpan{}
	for coordinate := lineMin; coordinate < lineMax; coordinate++ {
		var point image.Point
		if horizontal {
			point = image.Pt(coordinate, line)
		} else {
			point = image.Pt(line, coordinate)
		}
		opaque := img.NRGBAAt(point.X, point.Y).A > TransparentAlphaMax
		if !opaque {
			inRun = false
			continue
		}
		if !inRun {
			runs++
			span.min = coordinate
			inRun = true
		}
		span.max = coordinate
	}
	if runs != 1 {
		return opaqueSpan{}, false
	}
	return span, true
}

func queueBoundaryShift(
	snapshot, smoothReference *image.NRGBA,
	exterior []bool,
	bounds image.Rectangle,
	palette []color.RGBA,
	current, target, opposite int,
	leading bool,
	line int,
	fill, remove *[]image.Point,
) {
	if target == current || absInt(target-current) > 1 || current == opposite {
		return
	}
	if leading && target > current || !leading && target < current {
		// The median would move the boundary inward. Remove one weak edge pixel.
		point := boundaryPoint(line, current, leading)
		if snapshot.NRGBAAt(point.X, point.Y).A <= TransparentAlphaMax ||
			smoothReference.NRGBAAt(point.X, point.Y).A > contourRemoveAlphaCeiling {
			return
		}
		*remove = append(*remove, point)
		return
	}

	// The median would move the boundary outward. Add one pixel only if it is
	// exterior background and the smooth reference supports foreground coverage.
	point := boundaryPoint(line, target, leading)
	if !point.In(bounds) || !exterior[contourPointIndex(point, bounds)] ||
		snapshot.NRGBAAt(point.X, point.Y).A > TransparentAlphaMax ||
		smoothReference.NRGBAAt(point.X, point.Y).A < contourFillAlphaFloor {
		return
	}
	*fill = append(*fill, point)
}

func boundaryPoint(line, coordinate int, leading bool) image.Point {
	if leading {
		return image.Pt(coordinate, line)
	}
	return image.Pt(line, coordinate)
}

func median3(a, b, c int) int {
	if (a <= b && b <= c) || (c <= b && b <= a) {
		return b
	}
	if (b <= a && a <= c) || (c <= a && a <= b) {
		return a
	}
	return c
}
