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
