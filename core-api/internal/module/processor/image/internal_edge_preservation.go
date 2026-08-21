package image

import (
	"image"
	"image/color"
)

const (
	internalEdgeMinimumContrast    = 0.012
	internalEdgeBridgeContrast     = 0.006
	internalEdgeMaximumFlankSpread = 0.035
	internalEdgeMinimumLightness   = 0.035
)

type internalEdgeEvidence struct {
	contrast float64
	flankL   float64
	normal   image.Point
	dark     bool
}

// stabilizeInternalHardEdges turns source-supported, continuous one- or
// two-pixel internal ridges into coherent palette lines. Area reduction often
// leaves a thin seam represented by several blended shades; ordinary nearest
// palette mapping then makes that seam patchy or partially invisible. This
// pass detects only local colour extrema surrounded by foreground on both
// sides, so broad shading boundaries and the outer silhouette are not traced.
// It recolours existing opaque pixels only and never changes alpha or geometry.
func stabilizeInternalHardEdges(
	img, smoothReference *image.NRGBA,
	palette []color.RGBA,
) {
	if img == nil || smoothReference == nil || len(palette) == 0 {
		return
	}
	bounds := img.Bounds().Intersect(smoothReference.Bounds())
	if bounds.Dx() < 5 || bounds.Dy() < 5 {
		return
	}

	evidence := make([]internalEdgeEvidence, bounds.Dx()*bounds.Dy())
	candidates := make([]bool, len(evidence))
	index := func(point image.Point) int {
		return (point.Y-bounds.Min.Y)*bounds.Dx() + point.X - bounds.Min.X
	}
	for y := bounds.Min.Y + 2; y < bounds.Max.Y-2; y++ {
		for x := bounds.Min.X + 2; x < bounds.Max.X-2; x++ {
			point := image.Pt(x, y)
			current, ok := strongestInternalEdgeEvidence(smoothReference, bounds, point)
			if !ok {
				continue
			}
			evidence[index(point)] = current
			candidates[index(point)] = current.contrast >= internalEdgeMinimumContrast
		}
	}
	bridgeInternalEdgeGaps(candidates, evidence, bounds, index)
	thinInternalEdgeCandidates(candidates, evidence, smoothReference, bounds, index)
	stabilizeInternalEdgeComponents(img, smoothReference, palette, bounds, candidates, evidence, index)
}

func strongestInternalEdgeEvidence(
	reference *image.NRGBA,
	bounds image.Rectangle,
	point image.Point,
) (internalEdgeEvidence, bool) {
	center := reference.NRGBAAt(point.X, point.Y)
	if center.A < hardAlphaThreshold || opaqueNeighbourCount(reference, bounds, point) < 7 {
		return internalEdgeEvidence{}, false
	}
	centerLab := nrgbaToOKLab(center)
	normals := [...]image.Point{{X: 1}, {Y: 1}, {X: 1, Y: 1}, {X: 1, Y: -1}}
	best := internalEdgeEvidence{}
	found := false
	for _, normal := range normals {
		for radius := 1; radius <= 2; radius++ {
			offset := image.Pt(normal.X*radius, normal.Y*radius)
			leftPoint, rightPoint := point.Sub(offset), point.Add(offset)
			if !leftPoint.In(bounds) || !rightPoint.In(bounds) {
				continue
			}
			left := reference.NRGBAAt(leftPoint.X, leftPoint.Y)
			right := reference.NRGBAAt(rightPoint.X, rightPoint.Y)
			if left.A < hardAlphaThreshold || right.A < hardAlphaThreshold {
				continue
			}
			leftLab, rightLab := nrgbaToOKLab(left), nrgbaToOKLab(right)
			flankSpread := perceptualColourDistance(leftLab, rightLab)
			if flankSpread > internalEdgeMaximumFlankSpread {
				continue
			}
			leftContrast := perceptualColourDistance(centerLab, leftLab)
			rightContrast := perceptualColourDistance(centerLab, rightLab)
			contrast := min(leftContrast, rightContrast) - flankSpread*0.35
			flankL := (leftLab.l + rightLab.l) / 2
			lightnessDelta := centerLab.l - flankL
			if contrast <= 0 || absFloat(lightnessDelta) < internalEdgeMinimumLightness {
				continue
			}
			if !found || contrast > best.contrast {
				best = internalEdgeEvidence{
					contrast: contrast,
					flankL:   flankL,
					normal:   normal,
					dark:     lightnessDelta < 0,
				}
				found = true
			}
		}
	}
	return best, found
}

func opaqueNeighbourCount(reference *image.NRGBA, bounds image.Rectangle, point image.Point) int {
	count := 0
	for y := -1; y <= 1; y++ {
		for x := -1; x <= 1; x++ {
			if x == 0 && y == 0 {
				continue
			}
			neighbor := point.Add(image.Pt(x, y))
			if neighbor.In(bounds) && reference.NRGBAAt(neighbor.X, neighbor.Y).A >= hardAlphaThreshold {
				count++
			}
		}
	}
	return count
}

func bridgeInternalEdgeGaps(
	candidates []bool,
	evidence []internalEdgeEvidence,
	bounds image.Rectangle,
	index func(image.Point) int,
) {
	snapshot := append([]bool(nil), candidates...)
	pairs := [...][2]image.Point{
		{{X: -1}, {X: 1}},
		{{Y: -1}, {Y: 1}},
		{{X: -1, Y: -1}, {X: 1, Y: 1}},
		{{X: 1, Y: -1}, {X: -1, Y: 1}},
	}
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			point := image.Pt(x, y)
			pointIndex := index(point)
			if snapshot[pointIndex] || evidence[pointIndex].contrast < internalEdgeBridgeContrast {
				continue
			}
			for _, pair := range pairs {
				left, right := point.Add(pair[0]), point.Add(pair[1])
				if !left.In(bounds) || !right.In(bounds) ||
					!snapshot[index(left)] || !snapshot[index(right)] {
					continue
				}
				if evidence[index(left)].dark == evidence[index(right)].dark {
					candidates[pointIndex] = true
					break
				}
			}
		}
	}
}

func thinInternalEdgeCandidates(
	candidates []bool,
	evidence []internalEdgeEvidence,
	reference *image.NRGBA,
	bounds image.Rectangle,
	index func(image.Point) int,
) {
	snapshot := append([]bool(nil), candidates...)
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			point := image.Pt(x, y)
			pointIndex := index(point)
			if !snapshot[pointIndex] {
				continue
			}
			current := evidence[pointIndex]
			currentL := nrgbaToOKLab(reference.NRGBAAt(x, y)).l
			for _, sign := range [...]int{-1, 1} {
				neighbor := point.Add(image.Pt(current.normal.X*sign, current.normal.Y*sign))
				if !neighbor.In(bounds) || !snapshot[index(neighbor)] ||
					!internalEdgeOrientationsCompatible(current.normal, evidence[index(neighbor)].normal) {
					continue
				}
				neighborEvidence := evidence[index(neighbor)]
				if neighborEvidence.dark != current.dark {
					continue
				}
				neighborL := nrgbaToOKLab(reference.NRGBAAt(neighbor.X, neighbor.Y)).l
				neighborIsStronger := neighborEvidence.contrast > current.contrast
				if current.dark {
					neighborIsStronger = neighborIsStronger ||
						(neighborEvidence.contrast == current.contrast && neighborL < currentL)
				} else {
					neighborIsStronger = neighborIsStronger ||
						(neighborEvidence.contrast == current.contrast && neighborL > currentL)
				}
				if neighborIsStronger ||
					(neighborEvidence.contrast == current.contrast && neighborL == currentL && index(neighbor) < pointIndex) {
					candidates[pointIndex] = false
					break
				}
			}
		}
	}
}

func internalEdgeOrientationsCompatible(left, right image.Point) bool {
	return absInt(left.X*right.X+left.Y*right.Y) > 0
}

func internalEdgeCandidatesConnect(
	left, right image.Point,
	evidence []internalEdgeEvidence,
	index func(image.Point) int,
) bool {
	leftEvidence, rightEvidence := evidence[index(left)], evidence[index(right)]
	if leftEvidence.dark != rightEvidence.dark ||
		!internalEdgeOrientationsCompatible(leftEvidence.normal, rightEvidence.normal) {
		return false
	}
	offset := right.Sub(left)
	return internalEdgeRunsAlongTangent(leftEvidence.normal, offset) ||
		internalEdgeRunsAlongTangent(rightEvidence.normal, offset)
}

func internalEdgeRunsAlongTangent(normal, offset image.Point) bool {
	normalMovement := absInt(normal.X*offset.X + normal.Y*offset.Y)
	tangentMovement := absInt(-normal.Y*offset.X + normal.X*offset.Y)
	return tangentMovement >= normalMovement
}

func stabilizeInternalEdgeComponents(
	img, reference *image.NRGBA,
	palette []color.RGBA,
	bounds image.Rectangle,
	candidates []bool,
	evidence []internalEdgeEvidence,
	index func(image.Point) int,
) {
	visited := make([]bool, len(candidates))
	neighbors := [...]image.Point{
		{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1},
		{X: -1}, {X: 1},
		{X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1},
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			start := image.Pt(x, y)
			startIndex := index(start)
			if visited[startIndex] || !candidates[startIndex] {
				continue
			}
			component := []image.Point{start}
			visited[startIndex] = true
			for cursor := 0; cursor < len(component); cursor++ {
				point := component[cursor]
				for _, offset := range neighbors {
					neighbor := point.Add(offset)
					if !neighbor.In(bounds) {
						continue
					}
					neighborIndex := index(neighbor)
					if visited[neighborIndex] || !candidates[neighborIndex] ||
						!internalEdgeCandidatesConnect(point, neighbor, evidence, index) {
						continue
					}
					visited[neighborIndex] = true
					component = append(component, neighbor)
				}
			}
			if len(component) < 3 || !coherentInternalEdgeComponent(component, evidence, index) {
				continue
			}
			target, ok := internalEdgePaletteColour(reference, palette, component, evidence, index)
			if !ok {
				continue
			}
			for _, point := range component {
				pixel := img.NRGBAAt(point.X, point.Y)
				if pixel.A <= TransparentAlphaMax {
					continue
				}
				pixel.R, pixel.G, pixel.B = target.R, target.G, target.B
				img.SetNRGBA(point.X, point.Y, pixel)
			}
		}
	}
}

func coherentInternalEdgeComponent(
	component []image.Point,
	evidence []internalEdgeEvidence,
	index func(image.Point) int,
) bool {
	dark, contrast := 0, 0.0
	for _, point := range component {
		item := evidence[index(point)]
		if item.dark {
			dark++
		}
		contrast += item.contrast
	}
	minority := min(dark, len(component)-dark)
	return minority*5 <= len(component) && contrast/float64(len(component)) >= internalEdgeMinimumContrast
}

func internalEdgePaletteColour(
	reference *image.NRGBA,
	palette []color.RGBA,
	component []image.Point,
	evidence []internalEdgeEvidence,
	index func(image.Point) int,
) (color.RGBA, bool) {
	darkVotes := 0
	averageFlankL := 0.0
	for _, point := range component {
		item := evidence[index(point)]
		if item.dark {
			darkVotes++
		}
		averageFlankL += item.flankL
	}
	dark := darkVotes*2 >= len(component)
	averageFlankL /= float64(len(component))

	bestIndex := -1
	bestScore := 0.0
	for paletteIndex, candidate := range palette {
		candidateLab := rgbaToOKLab(candidate)
		if dark && candidateLab.l >= averageFlankL-internalEdgeMinimumLightness {
			continue
		}
		if !dark && candidateLab.l <= averageFlankL+internalEdgeMinimumLightness {
			continue
		}
		score := 0.0
		for _, point := range component {
			source := reference.NRGBAAt(point.X, point.Y)
			score += perceptualColourDistance(nrgbaToOKLab(source), candidateLab)
		}
		if bestIndex < 0 || score < bestScore ||
			(score == bestScore && rgbaKey(candidate) < rgbaKey(palette[bestIndex])) {
			bestIndex, bestScore = paletteIndex, score
		}
	}
	if bestIndex < 0 {
		return color.RGBA{}, false
	}
	return palette[bestIndex], true
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
