package world

import "math"

// TryMove moves a player (a square box of half-size radius around (x, y)) by
// (dx, dy) and returns the new position. X and Y are tried separately: if
// one axis is blocked, the other still moves, so the player slides along
// walls instead of sticking to them.
//
// Each move must be smaller than 1 - 2*radius, or the box could skip over a
// wall. The game's dt clamp keeps steps far below that.
func TryMove(m *Map, x, y, dx, dy, radius float64) (newX, newY float64) {
	if !boxHitsWall(m, x+dx, y, radius) {
		x += dx
	}
	if !boxHitsWall(m, x, y+dy, radius) {
		y += dy
	}
	return x, y
}

// boxHitsWall reports whether the box around (x, y) overlaps any wall cell.
// Touching a wall face exactly doesn't count, so a player flush against a
// wall can still slide along it.
func boxHitsWall(m *Map, x, y, radius float64) bool {
	// Cells the box covers. Ceil(...)-1 for the far edge: an edge at exactly
	// 4.0 ends in cell 3, it doesn't enter cell 4.
	minX, maxX := int(math.Floor(x-radius)), int(math.Ceil(x+radius))-1
	minY, maxY := int(math.Floor(y-radius)), int(math.Ceil(y+radius))-1
	for cy := minY; cy <= maxY; cy++ {
		for cx := minX; cx <= maxX; cx++ {
			if m.IsSolid(cx, cy) {
				return true
			}
		}
	}
	return false
}
