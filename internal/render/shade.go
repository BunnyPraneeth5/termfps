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

	// ceilingGray is a flat near-black ceiling.
	ceilingGray uint8 = 8
	// The floor fades from floorHorizon (far away, at the screen's middle)
	// to floorNear (at the bottom). floorNear stays well below a close wall
	// (230) so the floor doesn't wash the walls out.
	floorHorizon = 12.0
	floorNear    = 80.0

	// With textures on, floor and ceiling are cast (V1): their brightness
	// fades with distance like walls, from these values at distance 0.
	// floorBase gives about the old floorNear (80) at the bottom of the
	// screen; the ceiling stays dim so walls stand out against it.
	floorBase   = 120.0
	ceilingBase = 45.0
)

// backgroundShade returns the ceiling or floor gray for pixel row y of a
// frame h pixels tall. The top half is ceiling, the bottom half is floor.
func backgroundShade(y, h int) uint8 {
	center := float64(h) / 2
	row := float64(y) + 0.5 // middle of the pixel
	if row < center {
		return ceilingGray
	}
	t := (row - center) / center // 0 at the horizon, 1 at the bottom edge
	return toGray(floorHorizon + (floorNear-floorHorizon)*t)
}

// castShade returns the gray of a floor or ceiling point at distance dist,
// for a surface whose brightness at distance 0 is base: the same fog as
// walls.
func castShade(base, dist float64) uint8 {
	return toGray(base / (1 + fogK*math.Max(dist, 0)))
}

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
