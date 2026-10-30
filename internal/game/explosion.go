package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Explosions (M12: exploders). Balancing (M18) may tune them.
const (
	ExplosionRadius = 2.5 // cells: blast damage falls off to nothing here
	BoomTime        = 0.4 // seconds an explosion shows on screen
)

// blast is an explosion waiting to go off after delay seconds.
type blast struct {
	x, y   float64
	damage int // at its center
	delay  float64
}

// Boom is an explosion on screen, for the renderer.
type Boom struct {
	X, Y float64
	Age  float64 // seconds since it went off, up to BoomTime
}

// addBlast queues an explosion at (x, y) that goes off after delay seconds
// (0 = this frame).
func addBlast(s *State, x, y float64, damage int, delay float64) {
	s.blasts = append(s.blasts, blast{x: x, y: y, damage: damage, delay: delay})
}

// updateBlasts ages the explosions on screen and sets off the queued
// blasts whose time has come. A blast can queue more (an exploder it
// kills); those with no delay go off in the same frame.
func updateBlasts(s *State, dt float64) {
	kept := s.Booms[:0] // filter in place, reusing the array
	for _, b := range s.Booms {
		b.Age += dt
		if b.Age < BoomTime {
			kept = append(kept, b)
		}
	}
	s.Booms = kept

	for i := 0; i < len(s.blasts); {
		s.blasts[i].delay -= dt
		if s.blasts[i].delay > 0 {
			i++
			continue
		}
		b := s.blasts[i]
		s.blasts = append(s.blasts[:i], s.blasts[i+1:]...) // remove it, then set it off
		explode(s, b)
	}
}

// explode sets off b: it hurts the player and every enemy within
// ExplosionRadius that no wall shields, less the farther away they are.
func explode(s *State, b blast) {
	s.Booms = append(s.Booms, Boom{X: b.x, Y: b.y})
	s.Events = append(s.Events, EventExplosion)
	if d := math.Hypot(s.Player.X-b.x, s.Player.Y-b.y); d < ExplosionRadius && blastReaches(s.Map, b.x, b.y, s.Player.X, s.Player.Y) {
		hurtPlayer(s, blastDamage(b.damage, d))
	}
	// Backwards, so removing a killed enemy doesn't skip the next one.
	for i := len(s.Enemies) - 1; i >= 0; i-- {
		e := s.Enemies[i]
		if d := math.Hypot(e.X-b.x, e.Y-b.y); d < ExplosionRadius && blastReaches(s.Map, b.x, b.y, e.X, e.Y) {
			damageEnemy(s, i, blastDamage(b.damage, d))
		}
	}
}

// blastDamage is a blast's damage at d cells from its center.
func blastDamage(center int, d float64) int {
	return int(math.Round(float64(center) * (1 - d/ExplosionRadius)))
}

// blastReaches reports whether no wall (or closed door) stands between
// (ax, ay) and (bx, by). Props don't shield: a blast goes over a crate.
func blastReaches(m *world.Map, ax, ay, bx, by float64) bool {
	dx, dy := bx-ax, by-ay
	if dx == 0 && dy == 0 {
		return true
	}
	return world.CastRay(m, ax, ay, dx, dy).PerpDist >= 1
}
