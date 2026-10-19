package main

import "time"

// maxDT caps one frame's time step. After a stall (for example dragging the
// window), a huge dt would move the player through a wall in one step.
const maxDT = 0.1

// clampDT limits a frame's elapsed seconds to [0, maxDT].
func clampDT(seconds float64) float64 {
	if seconds < 0 {
		return 0
	}
	if seconds > maxDT {
		return maxDT
	}
	return seconds
}

// fpsCounter measures frames per second over windows of at least 1 second.
// The zero value is ready to use.
type fpsCounter struct {
	windowStart time.Time
	frames      int
	fps         float64
}

// tick records one frame at time now. The method has a pointer receiver
// (*fpsCounter) so it changes the counter itself, not a copy.
func (c *fpsCounter) tick(now time.Time) {
	if c.windowStart.IsZero() {
		c.windowStart = now
		return
	}
	c.frames++
	if elapsed := now.Sub(c.windowStart); elapsed >= time.Second {
		c.fps = float64(c.frames) / elapsed.Seconds()
		c.frames = 0
		c.windowStart = now
	}
}

// value returns the FPS of the last full window, or 0 before the first one.
func (c *fpsCounter) value() float64 {
	return c.fps
}
