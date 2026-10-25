package app

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

// tick records one frame at time now and reports whether a window just
// closed (a new FPS value is ready). The method has a pointer receiver
// (*fpsCounter) so it changes the counter itself, not a copy.
func (c *fpsCounter) tick(now time.Time) bool {
	if c.windowStart.IsZero() {
		c.windowStart = now
		return false
	}
	c.frames++
	if elapsed := now.Sub(c.windowStart); elapsed >= time.Second {
		c.fps = float64(c.frames) / elapsed.Seconds()
		c.frames = 0
		c.windowStart = now
		return true
	}
	return false
}

// value returns the FPS of the last full window, or 0 before the first one.
func (c *fpsCounter) value() float64 {
	return c.fps
}

// frameTimes are the three parts of a frame measured separately (see
// ARCHITECTURE "Performance budget").
type frameTimes struct {
	draw    time.Duration // DrawFrame: rays into the pixel buffer
	present time.Duration // presenter, minimap, HUD: into the screen's cell buffer
	show    time.Duration // Show: changed cells go to the terminal
}

// timeAverager averages frame timings over the FPS counter's windows.
// On Windows, Go's clock moves in steps of about 0.5ms, so one frame's
// 0.07ms DrawFrame usually reads as 0; the average over ~30 frames is
// accurate. The zero value is ready to use.
type timeAverager struct {
	sum frameTimes
	n   int
	avg frameTimes // average of the last full window
}

// add records one frame's timings.
func (a *timeAverager) add(t frameTimes) {
	a.sum.draw += t.draw
	a.sum.present += t.present
	a.sum.show += t.show
	a.n++
}

// roll closes the window: the average becomes visible and sums restart.
func (a *timeAverager) roll() {
	if a.n > 0 {
		n := time.Duration(a.n)
		a.avg = frameTimes{draw: a.sum.draw / n, present: a.sum.present / n, show: a.sum.show / n}
	}
	a.sum, a.n = frameTimes{}, 0
}
