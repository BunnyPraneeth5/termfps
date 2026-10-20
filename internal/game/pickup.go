package game

import "math"

// Ammo pickups (D22). Balancing (P1-16) may tune them.
const (
	PickupAmmo        = 25   // spare rounds per box
	MaxReserve        = 125  // spare rounds cap (5 magazines)
	PickupEveryMin    = 20.0 // seconds between boxes: random in [min, max]
	PickupEveryMax    = 40.0
	MaxPickups        = 3   // boxes on the map at once
	PickupRadius      = 0.6 // walk this close to a box's center to take it
	pickupMinDistance = 3.0 // cells: boxes never appear right next to the player
)

// Pickup is an ammo box lying on the floor.
type Pickup struct {
	X, Y   float64
	Amount int
}

// nextPickupDelay picks a random wait before the next box.
func nextPickupDelay(s *State) float64 {
	return PickupEveryMin + s.Rand.Float64()*(PickupEveryMax-PickupEveryMin)
}

// updatePickups lets the player take boxes they stand on, and drops new
// boxes on a random timer.
func updatePickups(s *State, dt float64) {
	w := &s.Player.Weapon
	for i := 0; i < len(s.Pickups); {
		pk := s.Pickups[i]
		near := math.Hypot(pk.X-s.Player.X, pk.Y-s.Player.Y) <= PickupRadius
		if !near || w.Reserve >= MaxReserve {
			i++ // a box stays when the player is away or already full
			continue
		}
		w.Reserve = min(w.Reserve+pk.Amount, MaxReserve)
		s.Pickups = append(s.Pickups[:i], s.Pickups[i+1:]...) // remove box i
		s.Events = append(s.Events, EventPickupTaken)
	}

	if !s.Spawning {
		return
	}
	s.pickupTimer -= dt
	if s.pickupTimer > 0 {
		return
	}
	s.pickupTimer = nextPickupDelay(s)
	if len(s.Pickups) >= MaxPickups {
		return
	}
	if x, y, ok := pickupSpot(s); ok {
		s.Pickups = append(s.Pickups, Pickup{X: x, Y: y, Amount: PickupAmmo})
		s.Events = append(s.Events, EventPickupSpawned)
	}
}

// pickupSpot picks a random reachable free cell at least pickupMinDistance
// from the player and not on another box. ok is false if none of
// spawnTries random cells qualify.
func pickupSpot(s *State) (x, y float64, ok bool) {
	ensureDistanceField(s)
	m := s.Map
	for range spawnTries {
		cx, cy := s.Rand.IntN(m.W), s.Rand.IntN(m.H)
		if m.IsSolid(cx, cy) || s.dist[cy*m.W+cx] < 0 {
			continue
		}
		x, y = float64(cx)+0.5, float64(cy)+0.5
		if math.Hypot(x-s.Player.X, y-s.Player.Y) < pickupMinDistance {
			continue
		}
		taken := false
		for _, pk := range s.Pickups {
			taken = taken || (pk.X == x && pk.Y == y)
		}
		if !taken {
			return x, y, true
		}
	}
	return 0, 0, false
}
