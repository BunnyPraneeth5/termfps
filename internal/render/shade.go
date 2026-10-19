package render

import "math"

// Shading constants, tuned by eye (ARCHITECTURE "Shading").
const (
	// wallBase is a wall's brightness right in front of the camera.
	wallBase = 230.0
	// fogK sets how fast walls fade: brightness = wallBase / (1 + fogK*dist).
	// 0.25 halves brightness at 4 cells and leaves ~1/5 across the map.
	fogK = 0.25
	// sideFactor darkens Y-side (north/south) faces so corners read clearly.
	sideFactor = 0.7
)

// wallShade returns the gray for a wall at perpendicular distance dist.
// side is world.Hit.Side: 0 = X-side face, 1 = Y-side face.
func wallShade(dist float64, side int) uint8 {
	g := wallBase / (1 + fogK*math.Max(dist, 0))
	if side == 1 {
		g *= sideFactor
	}
	return toGray(g)
}

// toGray rounds and clamps a brightness to 0-255 before converting. Go
// doesn't panic on an out-of-range float -> uint8 conversion; it gives a
// garbage value, so the clamp matters.
func toGray(v float64) uint8 {
	return uint8(clamp(math.Round(v), 0, 255))
}
