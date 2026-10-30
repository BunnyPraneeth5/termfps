package game

import "github.com/BunnyPraneeth5/termfps/internal/world"

// Acid floor (M14). Balancing (M18) may tune it.
const (
	AcidDamage   = 5    // HP per burn
	AcidInterval = 0.5  // seconds between burns while you stand in it
	acidFirst    = 0.15 // seconds from stepping in to the first burn
)

// updateAcid burns the player every AcidInterval seconds while they stand
// on an acid cell. Enemies don't mind it.
func updateAcid(s *State, dt float64) {
	if s.Map.At(int(s.Player.X), int(s.Player.Y)) != world.Acid {
		s.acidTimer = acidFirst
		return
	}
	s.acidTimer -= dt
	if s.acidTimer > 0 {
		return
	}
	s.acidTimer += AcidInterval
	s.Events = append(s.Events, EventAcidBurn)
	hurtPlayer(s, AcidDamage)
}
