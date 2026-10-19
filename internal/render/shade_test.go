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

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
