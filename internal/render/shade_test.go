package render

import (
	"math"
	"testing"
)

func TestWallShadeFadesWithDistance(t *testing.T) {
	prev := wallShade(0, 0)
	for _, d := range []float64{0.5, 1, 2, 4, 8, 16} {
		g := wallShade(d, 0)
		if g >= prev {
			t.Errorf("wallShade(%v) = %d, not darker than closer wall (%d)", d, g, prev)
		}
		prev = g
	}
	if g := wallShade(16, 1); g == 0 {
		t.Error("far Y-side wall is pure black, want still visible")
	}
}

func TestWallShadeSideIsDarker(t *testing.T) {
	for _, d := range []float64{0.5, 2, 6} {
		x, y := wallShade(d, 0), wallShade(d, 1)
		if y >= x {
			t.Errorf("dist %v: Y-side %d not darker than X-side %d", d, y, x)
		}
		if want := toGray(float64(wallShade(d, 0)) * sideFactor); absDiff(y, want) > 1 {
			t.Errorf("dist %v: Y-side %d, want about %d (0.7x)", d, y, want)
		}
	}
}

func TestToGrayClamps(t *testing.T) {
	cases := []struct {
		in   float64
		want uint8
	}{
		{-5, 0}, {0, 0}, {127.4, 127}, {127.6, 128}, {255, 255}, {300, 255}, {math.Inf(1), 255},
	}
	for _, c := range cases {
		if got := toGray(c.in); got != c.want {
			t.Errorf("toGray(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBackgroundShade(t *testing.T) {
	const h = 78
	for y := 0; y < h/2; y++ {
		if g := backgroundShade(y, h); g != ceilingGray {
			t.Errorf("row %d (top half) = %d, want ceiling %d", y, g, ceilingGray)
		}
	}
	// Floor gets brighter toward the bottom, never darker.
	prev := backgroundShade(h/2, h)
	for y := h/2 + 1; y < h; y++ {
		g := backgroundShade(y, h)
		if g < prev {
			t.Errorf("floor row %d = %d, darker than the row above (%d)", y, g, prev)
		}
		prev = g
	}
	top, bottom := backgroundShade(h/2, h), backgroundShade(h-1, h)
	if top > 15 || bottom < 75 || bottom > 80 {
		t.Errorf("floor: horizon %d (want ~12), bottom %d (want ~80)", top, bottom)
	}
	// The floor never outshines a wall up close.
	if bottom >= wallShade(1, 1) {
		t.Errorf("floor bottom %d is as bright as a close side wall %d", bottom, wallShade(1, 1))
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
