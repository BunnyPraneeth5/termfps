package game

import "math"

// Pickups (D22; medkits and armor M10). Balancing (M18) may tune them.
const (
	PickupAmmo        = 25   // spare rounds per box
	MaxReserve        = 125  // spare rounds cap (5 magazines)
	MedkitHeal        = 40   // HP a medkit gives back, up to PlayerMaxHP
	ArmorPickup       = 50   // armor a vest gives, up to MaxArmor
	PickupEveryMin    = 15.0 // seconds between pickups: random in [min, max] (was 20-40 before M10)
	PickupEveryMax    = 30.0
	MaxPickups        = 4   // pickups on the map at once (was 3 before M10)
	PickupRadius      = 0.6 // walk this close to a pickup's center to take it
	pickupMinDistance = 3.0 // cells: pickups never appear right next to the player

	// What a new pickup is: ammo with this chance, else a medkit with
	// medkitShare, else armor.
	ammoShare   = 0.55
	medkitShare = 0.25
)

// PickupKind tells pickups apart (M10).
type PickupKind int

const (
	AmmoBox PickupKind = iota // the zero value, so Pickup{} is an ammo box
	Medkit
	ArmorVest
)

// Pickup is something lying on the floor that the player takes by walking
// over it.
type Pickup struct {
	X, Y   float64
	Kind   PickupKind
	Amount int // rounds, HP or armor it gives
}

// nextPickupDelay picks a random wait before the next pickup.
func nextPickupDelay(s *State) float64 {
	return PickupEveryMin + s.Rand.Float64()*(PickupEveryMax-PickupEveryMin)
}

// updatePickups lets the player take pickups they stand on, and drops new
// ones on a random timer.
func updatePickups(s *State, dt float64) {
	for i := 0; i < len(s.Pickups); {
		pk := s.Pickups[i]
		near := math.Hypot(pk.X-s.Player.X, pk.Y-s.Player.Y) <= PickupRadius
		if !near || !take(s, pk) {
			i++ // a pickup stays when the player is away or can't use it
			continue
		}
		s.Pickups = append(s.Pickups[:i], s.Pickups[i+1:]...) // remove pickup i
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
		s.Pickups = append(s.Pickups, newPickup(s, x, y))
		s.Events = append(s.Events, EventPickupSpawned)
	}
}

// newPickup picks what the next pickup at (x, y) is.
func newPickup(s *State, x, y float64) Pickup {
	r := s.Rand.Float64()
	switch {
	case r < ammoShare:
		return Pickup{X: x, Y: y, Kind: AmmoBox, Amount: PickupAmmo}
	case r < ammoShare+medkitShare:
		return Pickup{X: x, Y: y, Kind: Medkit, Amount: MedkitHeal}
	}
	return Pickup{X: x, Y: y, Kind: ArmorVest, Amount: ArmorPickup}
}

// take gives pk to the player and reports whether they took it. A pickup
// the player can't use (full ammo, HP or armor) stays for later.
func take(s *State, pk Pickup) bool {
	p := &s.Player
	switch pk.Kind {
	case AmmoBox:
		if p.Weapon.Reserve >= MaxReserve {
			return false
		}
		p.Weapon.Reserve = min(p.Weapon.Reserve+pk.Amount, MaxReserve)
		s.Events = append(s.Events, EventPickupTaken)
	case Medkit:
		if p.HP >= PlayerMaxHP {
			return false
		}
		p.HP = min(p.HP+pk.Amount, PlayerMaxHP)
		s.Events = append(s.Events, EventMedkitTaken)
	case ArmorVest:
		if p.Armor >= MaxArmor {
			return false
		}
		p.Armor = min(p.Armor+pk.Amount, MaxArmor)
		s.Events = append(s.Events, EventArmorTaken)
	}
	return true
}

// pickupSpot picks a random reachable free cell at least pickupMinDistance
// from the player and not on another pickup. ok is false if none of
// spawnTries random cells qualify.
func pickupSpot(s *State) (x, y float64, ok bool) {
	ensureDistanceField(s)
	m := s.Map
	for range spawnTries {
		cx, cy := s.Rand.IntN(m.W), s.Rand.IntN(m.H)
		if !freeCell(m, cx, cy) || s.dist[cy*m.W+cx] < 0 {
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
