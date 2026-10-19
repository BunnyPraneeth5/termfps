package main

import (
	"math"
	"testing"
	"time"
)

func TestClampDT(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{-1, 0},
		{0, 0},
		{0.033, 0.033},
		{0.1, 0.1},
		{5, 0.1},
	}
	for _, c := range cases {
		if got := clampDT(c.in); got != c.want {
			t.Errorf("clampDT(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFPSCounter(t *testing.T) {
	cases := []struct {
		name     string
		interval time.Duration
		want     float64
	}{
		{"30fps", time.Second / 30, 30},
		{"15fps", time.Second / 15, 15},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fc fpsCounter
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			if fc.value() != 0 {
				t.Fatalf("value before any window = %v, want 0", fc.value())
			}
			// Simulate 3 seconds of frames.
			frames := int(3 * time.Second / c.interval)
			for range frames {
				fc.tick(now)
				now = now.Add(c.interval)
			}
			if got := fc.value(); math.Abs(got-c.want) > 1 {
				t.Errorf("fps = %.2f, want about %v", got, c.want)
			}
		})
	}
}
