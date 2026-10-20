package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// open is a big empty room so movement tests don't hit walls.
const open = "####################\n" +
	"#..................#\n" +
	"#..................#\n" +
	"#..................#\n" +
	"#.........P........#\n" +
	"#..................#\n" +
	"#..................#\n" +
	"#..................#\n" +
	"####################\n"

func newOpenState(t *testing.T) *State {
	t.Helper()
	m, err := world.Parse([]byte(open))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := NewState(m)
	s.Spawning = false // movement tests: no surprise enemies
	return s
}

// angleDiff is the smallest difference between two angles, in radians.
func angleDiff(a, b float64) float64 {
	return math.Abs(math.Remainder(a-b, 2*math.Pi))
}

func TestTurn360Returns(t *testing.T) {
	s := newOpenState(t)
	start := s.Player.Angle
	oneDegree := (math.Pi / 180) / TurnSpeed // dt that turns exactly 1 degree
	for range 360 {
		Update(s, Input{TurnRight: true}, oneDegree)
	}
	if d := angleDiff(s.Player.Angle, start); d > 1e-9 {
		t.Errorf("after 360 x 1 degree, angle is off by %v rad", d)
	}
	if s.Player.Angle < 0 || s.Player.Angle >= 2*math.Pi {
		t.Errorf("angle %v not in [0, 2pi)", s.Player.Angle)
	}
}

func TestTurnDirection(t *testing.T) {
	s := newOpenState(t)
	Update(s, Input{TurnRight: true}, 0.5) // +90 degrees
	if d := angleDiff(s.Player.Angle, math.Pi/2); d > 1e-9 {
		t.Errorf("TurnRight 0.5s: angle = %v, want pi/2 (facing south)", s.Player.Angle)
	}
	Update(s, Input{TurnLeft: true}, 1.0) // -180 degrees
	if d := angleDiff(s.Player.Angle, 3*math.Pi/2); d > 1e-9 {
		t.Errorf("then TurnLeft 1s: angle = %v, want 3pi/2 (facing north)", s.Player.Angle)
	}
}

// The same 2 seconds of walking covers the same distance at 15 and 30 FPS.
// (6 cells east from x=10.5 stays clear of the east wall at x=19.)
func TestMoveFrameRateIndependent(t *testing.T) {
	distance := func(fps int) float64 {
		s := newOpenState(t)
		startX := s.Player.X
		for range 2 * fps {
			Update(s, Input{Forward: true}, 1/float64(fps))
		}
		return s.Player.X - startX
	}
	d15, d30 := distance(15), distance(30)
	if math.Abs(d15-d30) > 1e-9 || math.Abs(d30-2*MoveSpeed) > 1e-9 {
		t.Errorf("distance at 15 FPS = %v, at 30 FPS = %v, want both %v", d15, d30, 2*MoveSpeed)
	}
}

// Facing east, strafe right goes south (+y) and strafe left goes north.
func TestStrafeDirection(t *testing.T) {
	s := newOpenState(t)
	y0 := s.Player.Y
	Update(s, Input{StrafeRight: true}, 0.1)
	if s.Player.Y <= y0 {
		t.Errorf("StrafeRight facing east: y went %v -> %v, want it to grow (south)", y0, s.Player.Y)
	}
	y1 := s.Player.Y
	Update(s, Input{StrafeLeft: true}, 0.2)
	if s.Player.Y >= y1 {
		t.Errorf("StrafeLeft facing east: y went %v -> %v, want it to shrink (north)", y1, s.Player.Y)
	}
}

// Forward + strafe at once is not faster than forward alone.
func TestDiagonalNotFaster(t *testing.T) {
	s := newOpenState(t)
	x0, y0 := s.Player.X, s.Player.Y
	Update(s, Input{Forward: true, StrafeRight: true}, 0.1)
	got := math.Hypot(s.Player.X-x0, s.Player.Y-y0)
	if want := MoveSpeed * 0.1; math.Abs(got-want) > 1e-9 {
		t.Errorf("diagonal step = %v, want %v", got, want)
	}
}

// Walking forward for 10 seconds into the east wall stops at the wall.
func TestUpdateStopsAtWall(t *testing.T) {
	s := newOpenState(t) // facing east, east wall face at x=19
	for range 300 {
		Update(s, Input{Forward: true}, 1.0/30)
	}
	if s.Player.X+Radius > 19 {
		t.Errorf("player box edge at %v, inside the east wall (face at 19)", s.Player.X+Radius)
	}
	if s.Player.X < 18.5 {
		t.Errorf("player stopped at x=%v, want near the wall", s.Player.X)
	}
}

func TestToggleMinimap(t *testing.T) {
	s := newOpenState(t)
	if !s.ShowMinimap {
		t.Fatal("minimap off at start, want on")
	}
	Update(s, Input{ToggleMinimap: true}, 1.0/30)
	if s.ShowMinimap {
		t.Error("minimap still on after one toggle")
	}
	Update(s, Input{}, 1.0/30) // no toggle this frame: stays off
	if s.ShowMinimap {
		t.Error("minimap turned back on without a toggle")
	}
	Update(s, Input{ToggleMinimap: true}, 1.0/30)
	if !s.ShowMinimap {
		t.Error("minimap still off after the second toggle")
	}
}

func TestOppositeKeysCancel(t *testing.T) {
	s := newOpenState(t)
	before := s.Player
	Update(s, Input{Forward: true, Back: true, TurnLeft: true, TurnRight: true}, 0.1)
	if s.Player != before {
		t.Errorf("player moved with opposite keys held: %+v -> %+v", before, s.Player)
	}
}
