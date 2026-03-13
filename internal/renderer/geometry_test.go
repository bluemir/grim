package renderer

import (
	"math"
	"testing"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestIntersectLineRect(t *testing.T) {
	tests := []struct {
		name   string
		cx, cy float64 // line start (inside rect)
		tx, ty float64 // line end (outside rect)
		rx, ry float64 // rect top-left
		rw, rh float64 // rect size
		wantX  float64
		wantY  float64
	}{
		{
			name: "right edge",
			cx:   50, cy: 50, tx: 200, ty: 50,
			rx: 0, ry: 0, rw: 100, rh: 100,
			wantX: 100, wantY: 50,
		},
		{
			name: "left edge",
			cx:   50, cy: 50, tx: -100, ty: 50,
			rx: 0, ry: 0, rw: 100, rh: 100,
			wantX: 0, wantY: 50,
		},
		{
			name: "top edge",
			cx:   50, cy: 50, tx: 50, ty: -100,
			rx: 0, ry: 0, rw: 100, rh: 100,
			wantX: 50, wantY: 0,
		},
		{
			name: "bottom edge",
			cx:   50, cy: 50, tx: 50, ty: 200,
			rx: 0, ry: 0, rw: 100, rh: 100,
			wantX: 50, wantY: 100,
		},
		{
			name: "diagonal top-right corner",
			cx:   50, cy: 50, tx: 200, ty: -100,
			rx: 0, ry: 0, rw: 100, rh: 100,
			// Line goes to top-right. dx=150, dy=-150.
			// Right edge: t = 50/150 = 1/3, iy = 50 - 50 = 0. On boundary.
			// Top edge: t = 50/150 = 1/3, ix = 50 + 50 = 100. On boundary.
			// Both at same t, corner point (100, 0).
			wantX: 100, wantY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := IntersectLineRect(tt.cx, tt.cy, tt.tx, tt.ty, tt.rx, tt.ry, tt.rw, tt.rh)
			if !approxEqual(gotX, tt.wantX, 0.01) || !approxEqual(gotY, tt.wantY, 0.01) {
				t.Errorf("IntersectLineRect() = (%v, %v), want (%v, %v)", gotX, gotY, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestApplyGap(t *testing.T) {
	// Moving right from origin
	x, y := ApplyGap(100, 50, 50, 50, 4)
	if !approxEqual(x, 104, 0.01) || !approxEqual(y, 50, 0.01) {
		t.Errorf("ApplyGap right = (%v, %v), want (104, 50)", x, y)
	}

	// Moving up
	x, y = ApplyGap(50, 0, 50, 50, 4)
	if !approxEqual(x, 50, 0.01) || !approxEqual(y, -4, 0.01) {
		t.Errorf("ApplyGap up = (%v, %v), want (50, -4)", x, y)
	}

	// Zero distance
	x, y = ApplyGap(50, 50, 50, 50, 4)
	if x != 50 || y != 50 {
		t.Errorf("ApplyGap zero = (%v, %v), want (50, 50)", x, y)
	}
}
