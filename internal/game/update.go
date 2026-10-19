package game

import "math"

const (
	// MoveSpeed is in map cells per second.
	MoveSpeed = 3.0
	// TurnSpeed is in radians per second (pi = 180 degrees per second).
	TurnSpeed = math.Pi
)

// Input is what the player is doing this frame. The input package fills it;
// game never sees tcell keys.
type Input struct {
	Forward, Back           bool
	StrafeLeft, StrafeRight bool
	TurnLeft, TurnRight     bool
}

// Update advances the game by dt seconds. All speeds are per second, so the
// result doesn't depend on the frame rate.
func Update(s *State, in Input, dt float64) {
	p := &s.Player // pointer: changes below modify s.Player itself, not a copy

	turn := axis(in.TurnRight, in.TurnLeft) // right = growing angle (y points down)
	p.Angle = normalizeAngle(p.Angle + turn*TurnSpeed*dt)

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
	p.X += moveX / length * MoveSpeed * dt
	p.Y += moveY / length * MoveSpeed * dt
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
