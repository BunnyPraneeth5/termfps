package render

// crosshairArm is how many pixels each arm of the crosshair reaches out
// from the center pixel.
const crosshairArm = 2

// DrawCrosshair draws a small "+" at the center of the Frame, where shots
// go. Each pixel is the opposite of what's under it (white on dark, black
// on bright), so it stays visible on any wall or floor. Call it after
// DrawFrame; it is part of the Frame, so every render mode shows it.
func DrawCrosshair(f *Frame) {
	if f.W <= 0 || f.H <= 0 {
		return
	}
	cx, cy := f.W/2, f.H/2
	invert(f, cx, cy)
	for d := 1; d <= crosshairArm; d++ {
		invert(f, cx-d, cy)
		invert(f, cx+d, cy)
		invert(f, cx, cy-d)
		invert(f, cx, cy+d)
	}
}

// invert sets the pixel at (x, y) to white or black, whichever stands out
// against its current gray. Pixels outside the Frame are skipped.
func invert(f *Frame, x, y int) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	p := &f.Pix[y*f.W+x]
	if p.Gray < 128 {
		*p = Pixel{Gray: 255}
	} else {
		*p = Pixel{Gray: 0}
	}
}
