package game

import "math"

// Game feel (M16): sprint with stamina, where hits come from, hit markers,
// screen shake. Balancing (M18) may tune the numbers.
const (
	SprintMul     = 1.6  // speed while sprinting
	StaminaDrain  = 0.25 // stamina used per second of sprinting (4 s from full)
	StaminaRegen  = 0.2  // stamina back per second when not sprinting (5 s to full)
	StaminaResume = 0.3  // after running out, sprinting waits until stamina is back to this

	HitDirTime  = 0.8  // seconds the "hit from there" marker shows
	HitMarkTime = 0.15 // seconds the crosshair hit marker shows

	shakeTime    = 0.35 // seconds a full shake lasts
	shakeReach   = 8.0  // cells: explosions farther away than this don't shake the screen
	tankShake    = 0.7  // shake strength (0-1) of a tank hit
	maxBlastFelt = 1.0  // shake strength of an explosion right next to you
)

// Sprinting reports whether the player sprints this frame: F held while
// moving, with stamina left (and not waiting for it to come back).
func (p Player) sprinting(in Input) bool {
	moving := axis(in.Forward, in.Back) != 0 || axis(in.StrafeRight, in.StrafeLeft) != 0
	return in.Sprint && moving && !p.tired && p.Stamina > 0
}

// updateStamina drains stamina while sprinting and refills it otherwise.
func updateStamina(s *State, sprinting bool, dt float64) {
	p := &s.Player
	if sprinting {
		p.Stamina = max(p.Stamina-StaminaDrain*dt, 0)
		if p.Stamina == 0 {
			p.tired = true
			s.Events = append(s.Events, EventOutOfBreath)
		}
		return
	}
	p.Stamina = min(p.Stamina+StaminaRegen*dt, 1)
	if p.tired && p.Stamina >= StaminaResume {
		p.tired = false
	}
}

// HitDirection returns the world angle from the player toward whatever
// hit them last, and how strongly to show it (1 right after the hit,
// fading to 0); ok is false when there's nothing to show.
func (p Player) HitDirection() (angle, strength float64, ok bool) {
	if p.hitDir <= 0 {
		return 0, 0, false
	}
	return p.hitFrom, p.hitDir / HitDirTime, true
}

// HitMarker reports whether the crosshair hit marker shows (the player's
// shot just hit an enemy), and whether that hit was a kill.
func (p Player) HitMarker() (on, kill bool) {
	return p.hitMark > 0, p.hitKill
}

// Shake returns how far the view is pushed this frame, each from -1 to 1
// (the renderer scales it to pixels). 0, 0 when not shaking.
func (p Player) Shake() (x, y float64) {
	return p.shakeX, p.shakeY
}

// shake starts a screen shake of strength 0-1; a stronger one replaces a
// weaker one still going.
func shake(s *State, strength float64) {
	p := &s.Player
	p.shakePower = max(p.shakePower*p.shake/shakeTime, min(strength, 1))
	p.shake = shakeTime
}

// updateFeel counts down the markers and the shake for one frame. Update
// calls it first, so ones started this frame show in full.
func updateFeel(s *State, dt float64) {
	p := &s.Player
	p.hitDir = max(p.hitDir-dt, 0)
	p.hitMark = max(p.hitMark-dt, 0)
	p.shake = max(p.shake-dt, 0)
}

// shakeView picks this frame's shake push. Update calls it last, so a
// shake started this frame already shows.
func shakeView(s *State) {
	p := &s.Player
	if p.shake == 0 {
		p.shakeX, p.shakeY = 0, 0
		return
	}
	// A random push each frame, weaker as the shake runs out. From the
	// injected Rand, so tests repeat.
	k := p.shakePower * p.shake / shakeTime
	p.shakeX = (s.Rand.Float64()*2 - 1) * k
	p.shakeY = (s.Rand.Float64()*2 - 1) * k
}

// hurtPlayerFrom is hurtPlayer for a hit coming from (x, y): the HUD shows
// a marker pointing there (M16).
func hurtPlayerFrom(s *State, damage int, x, y float64) {
	if s.Player.Dead() || damage <= 0 {
		return
	}
	hurtPlayer(s, damage)
	p := &s.Player
	p.hitFrom = math.Atan2(y-p.Y, x-p.X)
	p.hitDir = HitDirTime
}

// markHit shows the crosshair hit marker for a shot that hit an enemy.
func markHit(s *State, kill bool) {
	p := &s.Player
	if p.hitMark == 0 || kill {
		p.hitKill = kill // a kill in the same moment wins over a plain hit
	}
	p.hitMark = HitMarkTime
}

// blastShake shakes the screen for an explosion d cells away.
func blastShake(s *State, d float64) {
	if d < shakeReach {
		shake(s, maxBlastFelt*(1-d/shakeReach))
	}
}
