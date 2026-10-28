package game

import "github.com/BunnyPraneeth5/termfps/internal/world"

// seenRays is how many rays markSeen casts across the view each frame.
const seenRays = 48

// markSeen marks the cells the player sees this frame (M8): the cells
// around them, and along rays across the field of view every cell up to
// and including the first wall or closed door. Props don't hide what's
// behind them (you see past them, as the 3D view does).
func markSeen(s *State) {
	m, p := s.Map, s.Player
	mark := func(x, y int) {
		if x >= 0 && y >= 0 && x < m.W && y < m.H {
			s.Seen[y*m.W+x] = true
		}
	}
	px, py := int(p.X), int(p.Y)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			mark(px+dx, py+dy)
		}
	}
	dirX, dirY := p.Dir()
	planeX, planeY := p.Plane()
	for i := range seenRays {
		cameraX := 2*float64(i)/float64(seenRays-1) - 1 // -1 (left edge) to 1 (right edge)
		world.VisitCells(m, p.X, p.Y, dirX+planeX*cameraX, dirY+planeY*cameraX, func(x, y int) bool {
			mark(x, y)
			if m.At(x, y) == world.Door {
				return !m.Blocked(x, y) // see through an open door, not a closed one
			}
			return !m.IsSolid(x, y)
		})
	}
}
