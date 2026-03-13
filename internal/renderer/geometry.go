package renderer

import "math"

// IntersectLineRect finds the point where a line from (cx, cy) to (tx, ty)
// crosses the boundary of a rectangle defined by top-left (rx, ry) with size (rw, rh).
// It returns the intersection point on the rectangle boundary closest to (cx, cy).
func IntersectLineRect(cx, cy, tx, ty, rx, ry, rw, rh float64) (float64, float64) {
	dx := tx - cx
	dy := ty - cy

	if dx == 0 && dy == 0 {
		return cx, cy
	}

	// Check intersection with each edge of the rectangle.
	// We parameterize the line as P = (cx, cy) + t * (dx, dy), t in [0, 1].
	// For each edge, solve for t and check bounds.
	bestT := math.Inf(1)
	bestX, bestY := cx, cy

	edges := [4][2]float64{
		// {coordinate value, 0=vertical/1=horizontal}
	}
	_ = edges

	// Left edge: x = rx
	if dx != 0 {
		t := (rx - cx) / dx
		if t >= 0 && t <= 1 {
			iy := cy + t*dy
			if iy >= ry && iy <= ry+rh && t < bestT {
				bestT = t
				bestX, bestY = rx, iy
			}
		}
	}

	// Right edge: x = rx + rw
	if dx != 0 {
		t := (rx + rw - cx) / dx
		if t >= 0 && t <= 1 {
			iy := cy + t*dy
			if iy >= ry && iy <= ry+rh && t < bestT {
				bestT = t
				bestX, bestY = rx+rw, iy
			}
		}
	}

	// Top edge: y = ry
	if dy != 0 {
		t := (ry - cy) / dy
		if t >= 0 && t <= 1 {
			ix := cx + t*dx
			if ix >= rx && ix <= rx+rw && t < bestT {
				bestT = t
				bestX, bestY = ix, ry
			}
		}
	}

	// Bottom edge: y = ry + rh
	if dy != 0 {
		t := (ry + rh - cy) / dy
		if t >= 0 && t <= 1 {
			ix := cx + t*dx
			if ix >= rx && ix <= rx+rw && t < bestT {
				bestT = t
				bestX, bestY = ix, ry+rh
			}
		}
	}

	return bestX, bestY
}

// IntersectLineShape dispatches to the correct intersection function based on shape type.
func IntersectLineShape(cx, cy, tx, ty, rx, ry, rw, rh float64, shape string) (float64, float64) {
	switch shape {
	case "circle":
		return IntersectLineEllipse(cx, cy, tx, ty, rx, ry, rw, rh)
	case "diamond":
		return IntersectLineDiamond(cx, cy, tx, ty, rx, ry, rw, rh)
	case "hexagon":
		return IntersectLineHexagon(cx, cy, tx, ty, rx, ry, rw, rh)
	case "parallelogram":
		return IntersectLineParallelogram(cx, cy, tx, ty, rx, ry, rw, rh)
	default: // rectangle, rounded, cylinder, cloud — use rect bounding box
		return IntersectLineRect(cx, cy, tx, ty, rx, ry, rw, rh)
	}
}

// IntersectLineEllipse finds where a line from (cx, cy) toward (tx, ty) crosses
// the boundary of an ellipse centered at (rx+rw/2, ry+rh/2) with radii rw/2, rh/2.
func IntersectLineEllipse(cx, cy, tx, ty, rx, ry, rw, rh float64) (float64, float64) {
	dx := tx - cx
	dy := ty - cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	a := rw / 2
	b := rh / 2
	// Normalize direction
	dist := math.Sqrt(dx*dx + dy*dy)
	ndx := dx / dist
	ndy := dy / dist
	// Parametric: ellipse boundary at angle θ → (a*cosθ, b*sinθ)
	// Scale direction to hit ellipse boundary
	denom := math.Sqrt((ndx*ndx)/(a*a) + (ndy*ndy)/(b*b))
	if denom == 0 {
		return cx, cy
	}
	t := 1.0 / denom
	return cx + ndx*t, cy + ndy*t
}

// IntersectLineDiamond finds intersection with a diamond (rotated rectangle).
func IntersectLineDiamond(cx, cy, tx, ty, rx, ry, rw, rh float64) (float64, float64) {
	midX := rx + rw/2
	midY := ry + rh/2
	// Diamond vertices: top, right, bottom, left
	pts := [4][2]float64{
		{midX, ry},      // top
		{rx + rw, midY}, // right
		{midX, ry + rh}, // bottom
		{rx, midY},      // left
	}
	return intersectLinePolygon(cx, cy, tx, ty, pts[:])
}

// IntersectLineHexagon finds intersection with a flat-top hexagon.
func IntersectLineHexagon(cx, cy, tx, ty, rx, ry, rw, rh float64) (float64, float64) {
	inset := rw * 0.25
	midY := ry + rh/2
	pts := [6][2]float64{
		{rx + inset, ry},           // top-left
		{rx + rw - inset, ry},      // top-right
		{rx + rw, midY},            // right
		{rx + rw - inset, ry + rh}, // bottom-right
		{rx + inset, ry + rh},      // bottom-left
		{rx, midY},                 // left
	}
	return intersectLinePolygon(cx, cy, tx, ty, pts[:])
}

// IntersectLineParallelogram finds intersection with a parallelogram.
func IntersectLineParallelogram(cx, cy, tx, ty, rx, ry, rw, rh float64) (float64, float64) {
	skew := rw * 0.2
	pts := [4][2]float64{
		{rx + skew, ry},           // top-left
		{rx + rw, ry},             // top-right
		{rx + rw - skew, ry + rh}, // bottom-right
		{rx, ry + rh},             // bottom-left
	}
	return intersectLinePolygon(cx, cy, tx, ty, pts[:])
}

// intersectLinePolygon finds the intersection of a ray from (cx,cy) toward (tx,ty)
// with the edges of a convex polygon defined by pts.
func intersectLinePolygon(cx, cy, tx, ty float64, pts [][2]float64) (float64, float64) {
	dx := tx - cx
	dy := ty - cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}

	bestT := math.Inf(1)
	bestX, bestY := cx, cy
	n := len(pts)

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		ex, ey := pts[i][0], pts[i][1]
		fx, fy := pts[j][0], pts[j][1]

		// Edge vector
		edx := fx - ex
		edy := fy - ey

		denom := dx*edy - dy*edx
		if denom == 0 {
			continue // parallel
		}

		t := ((ex-cx)*edy - (ey-cy)*edx) / denom
		u := ((ex-cx)*dy - (ey-cy)*dx) / denom

		if t >= 0 && t <= 1 && u >= 0 && u <= 1 && t < bestT {
			bestT = t
			bestX = cx + t*dx
			bestY = cy + t*dy
		}
	}

	return bestX, bestY
}

// ApplyGap moves a point along the direction from (ox, oy) to (px, py) by gap pixels
// away from (ox, oy). Returns the adjusted point.
func ApplyGap(px, py, ox, oy, gap float64) (float64, float64) {
	dx := px - ox
	dy := py - oy
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist == 0 {
		return px, py
	}
	return px + (dx/dist)*gap, py + (dy/dist)*gap
}
