package app

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

// Coarse readings (0 or 0.5ms, like Windows' clock) average to the true
// value: here 1 frame in 7 reads 0.5ms, so the mean is about 0.071ms.
func TestTimeAveragerSmoothsCoarseClock(t *testing.T) {
	var a timeAverager
	for i := range 28 {
		d := time.Duration(0)
		if i%7 == 0 {
			d = 500 * time.Microsecond
		}
		a.add(frameTimes{draw: d, present: time.Millisecond, show: 2 * time.Millisecond})
	}
	if a.avg != (frameTimes{}) {
		t.Fatalf("avg = %+v before roll, want zero", a.avg)
	}
	a.roll()
	if got := a.avg.draw; got < 70*time.Microsecond || got > 72*time.Microsecond {
		t.Errorf("avg draw = %v, want about 71µs", got)
	}
	if a.avg.present != time.Millisecond || a.avg.show != 2*time.Millisecond {
		t.Errorf("avg = %+v, want present 1ms, show 2ms", a.avg)
	}
	a.roll() // empty window keeps the last average
	if a.avg.present != time.Millisecond {
		t.Errorf("empty window changed avg to %+v", a.avg)
	}
}

func TestFPSCounterReportsWindowClose(t *testing.T) {
	var fc fpsCounter
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	closed := 0
	for range 90 { // 3 seconds at 30 FPS
		if fc.tick(now) {
			closed++
		}
		now = now.Add(time.Second / 30)
	}
	if closed < 2 || closed > 3 {
		t.Errorf("windows closed = %d in 3 seconds, want 2-3", closed)
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
