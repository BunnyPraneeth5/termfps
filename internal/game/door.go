package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Door numbers (M3).
const (
	DoorReach = 1.5 // a door opens when the player or an enemy is this close to its center
	DoorHold  = 2.5 // seconds it stays open after the last one leaves its reach
	DoorSpeed = 2.0 // how much of the way it slides per second (half a second to open)
)

// updateDoors opens doors when the player or an enemy is near, and closes
// them again once nobody has been near for DoorHold seconds. A door never
// closes on someone standing in its cell.
func updateDoors(s *State, dt float64) {
	for i := range s.Map.Doors {
		d := &s.Map.Doors[i] // a pointer, so the changes stay in the map
		cx, cy := float64(d.X)+0.5, float64(d.Y)+0.5
		near, inside := false, false
		check := func(x, y, radius float64) {
			if math.Hypot(x-cx, y-cy) <= DoorReach {
				near = true
			}
			if math.Abs(x-cx) < 0.5+radius && math.Abs(y-cy) < 0.5+radius {
				inside = true
			}
		}
		check(s.Player.X, s.Player.Y, Radius)
		for _, e := range s.Enemies {
			check(e.X, e.Y, EnemyRadius)
		}
		if near || inside {
			d.Hold = DoorHold
		} else {
			d.Hold = max(d.Hold-dt, 0)
		}
		if d.Hold > 0 {
			d.Open = min(d.Open+DoorSpeed*dt, 1)
		} else {
			d.Open = max(d.Open-DoorSpeed*dt, 0)
		}
	}
}

// freeCell reports whether cell (x, y) is somewhere an enemy or an ammo box
// may appear: open floor (or a prop you can walk over), not a door.
func freeCell(m *world.Map, x, y int) bool {
	return !m.Blocked(x, y) && m.At(x, y) != world.Door
}
