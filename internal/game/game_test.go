package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

const eps = 1e-9

func near(a, b float64) bool { return math.Abs(a-b) < eps }

func TestDirAndPlane(t *testing.T) {
	cases := []struct {
		name           string
		angle          float64
		dirX, dirY     float64
		planeX, planeY float64
	}{
		// Facing east, the camera's right is south (+y, because y points down).
		{"east", 0, 1, 0, 0, PlaneLen},
		// Facing south, the camera's right is west (-x).
		{"south", math.Pi / 2, 0, 1, -PlaneLen, 0},
		{"west", math.Pi, -1, 0, 0, -PlaneLen},
		{"north", 3 * math.Pi / 2, 0, -1, PlaneLen, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Player{Angle: c.angle}
			dx, dy := p.Dir()
			px, py := p.Plane()
			if !near(dx, c.dirX) || !near(dy, c.dirY) {
				t.Errorf("Dir = (%v, %v), want (%v, %v)", dx, dy, c.dirX, c.dirY)
			}
			if !near(px, c.planeX) || !near(py, c.planeY) {
				t.Errorf("Plane = (%v, %v), want (%v, %v)", px, py, c.planeX, c.planeY)
			}
			if !near(dx*px+dy*py, 0) {
				t.Error("Plane is not perpendicular to Dir")
			}
		})
	}
}

func TestNewStateSpawn(t *testing.T) {
	m, err := world.Parse([]byte("####\n#.P#\n####\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := NewState(m)
	if s.Player.X != 2.5 || s.Player.Y != 1.5 || s.Player.Angle != 0 {
		t.Errorf("player = %+v, want X 2.5, Y 1.5, Angle 0", s.Player)
	}
}
