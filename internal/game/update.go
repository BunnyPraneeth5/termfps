package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

const (
	// MoveSpeed is in map cells per second.
	MoveSpeed = 3.0
	// TurnSpeed is in radians per second (pi = 180 degrees per second).
	TurnSpeed = math.Pi
	// Radius is the player's collision half-size, in map cells.
	Radius = 0.2
	// BobPerCell is how far the bob phase moves per cell walked
	// (2*pi / 1.5: one full sway every 1.5 cells).
	BobPerCell = 2 * math.Pi / 1.5
)

// Input is what the player is doing this frame. The input package fills it;
// game never sees tcell keys.
type Input struct {
	Forward, Back           bool
	StrafeLeft, StrafeRight bool
	TurnLeft, TurnRight     bool
	Fire                    bool // fire button held (or pressed) this frame
	Reload                  bool // reload asked for this frame
	ToggleMinimap           bool // one-shot: true only on the frame M was pressed
	CycleRenderMode         bool // one-shot: true only on the frame V was pressed
	// TurnSlow and MoveSlow take a fraction (0-1) off the turn and move
	// speed. Terminal key events use them while a key might still be a
	// short tap (P1-14). 0 = full speed, so other input can leave them out.
	TurnSlow, MoveSlow float64
}

// Update advances the game by dt seconds. All speeds are per second, so the
// result doesn't depend on the frame rate.
func Update(s *State, in Input, dt float64) {
	s.Events = s.Events[:0] // keep the backing array, drop last frame's events
	if s.Player.Dead() {
		return // nothing moves after death; the app shows Game over
	}
	s.Time += dt
	s.Player.hurt = max(s.Player.hurt-dt, 0)
	regenerate(&s.Player, dt)

	if in.ToggleMinimap {
		s.ShowMinimap = !s.ShowMinimap
	}
	if in.CycleRenderMode {
		s.Mode = s.Mode.Next()
		s.Events = append(s.Events, EventRenderModeChanged)
	}

	updateWeapon(s, in, dt)
	movePlayer(s, in, dt)
	updateEnemies(s, dt)
	updateSurvival(s, dt)
	updatePickups(s, dt)
}

// Health regeneration (2026-10-21 balancing, D30).
const (
	RegenDelay  = 4.0 // seconds without being hit before healing starts
	RegenPerSec = 5.0 // HP per second while healing
)

// regenerate heals the player slowly once they haven't been hit for
// RegenDelay seconds, up to PlayerMaxHP.
func regenerate(p *Player, dt float64) {
	p.sinceHurt += dt
	if p.sinceHurt < RegenDelay || p.HP >= PlayerMaxHP {
		p.regen = 0
		return
	}
	p.regen += RegenPerSec * dt
	whole := int(p.regen) // move only whole HP into HP; keep the fraction
	p.HP = min(p.HP+whole, PlayerMaxHP)
	p.regen -= float64(whole)
}

// movePlayer turns and moves the player for one frame, sliding along walls.
func movePlayer(s *State, in Input, dt float64) {
	p := &s.Player // pointer: changes below modify s.Player itself, not a copy

	turn := axis(in.TurnRight, in.TurnLeft) // right = growing angle (y points down)
	p.Angle = normalizeAngle(p.Angle + turn*TurnSpeed*(1-in.TurnSlow)*dt)

	forward := axis(in.Forward, in.Back)
	strafe := axis(in.StrafeRight, in.StrafeLeft)
	if forward == 0 && strafe == 0 {
		return
	}
	dirX, dirY := p.Dir()
	rightX, rightY := -dirY, dirX // unit vector to the camera's right (same way as Plane)
	moveX := dirX*forward + rightX*strafe
	moveY := dirY*forward + rightY*strafe
	// Forward + strafe together would be sqrt(2) times faster; normalize.
	length := math.Hypot(moveX, moveY)
	speed := MoveSpeed * (1 - in.MoveSlow)
	dx := moveX / length * speed * dt
	dy := moveY / length * speed * dt
	oldX, oldY := p.X, p.Y
	p.X, p.Y = world.TryMove(s.Map, p.X, p.Y, dx, dy, Radius)
	p.Bob = normalizeAngle(p.Bob + math.Hypot(p.X-oldX, p.Y-oldY)*BobPerCell)
}

// axis turns a pair of opposite keys into +1, -1 or 0 (both or neither).
func axis(positive, negative bool) float64 {
	v := 0.0
	if positive {
		v++
	}
	if negative {
		v--
	}
	return v
}

// normalizeAngle wraps an angle into [0, 2*pi).
func normalizeAngle(a float64) float64 {
	a = math.Mod(a, 2*math.Pi) // Mod keeps the sign, so negatives stay negative
	if a < 0 {
		a += 2 * math.Pi
	}
	return a
}
