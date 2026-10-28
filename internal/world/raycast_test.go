package world

import (
	"math"
	"testing"
)

// room is 7x7 with an open 5x5 inside: wall faces at x=1, x=6, y=1, y=6.
const room = "#######\n" +
	"#P....#\n" +
	"#.....#\n" +
	"#.....#\n" +
	"#.....#\n" +
	"#.....#\n" +
	"#######\n"

func mustParse(t *testing.T, s string) *Map {
	t.Helper() // report failures at the caller's line, not here
	m, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return m
}

const eps = 1e-9

func TestCastRayStraight(t *testing.T) {
	m := mustParse(t, room)
	// Off-center on purpose, so a sign mix-up between directions shows up.
	const px, py = 2.25, 1.5
	cases := []struct {
		name         string
		dx, dy       float64
		wantDist     float64
		wantSide     int
		wantX, wantY int
	}{
		{"east", 1, 0, 3.75, 0, 6, 1},
		{"west", -1, 0, 1.25, 0, 0, 1},
		{"north", 0, -1, 0.5, 1, 2, 0},
		{"south", 0, 1, 4.5, 1, 2, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := CastRay(m, px, py, c.dx, c.dy)
			if math.Abs(h.PerpDist-c.wantDist) > eps {
				t.Errorf("PerpDist = %v, want %v", h.PerpDist, c.wantDist)
			}
			if h.Side != c.wantSide {
				t.Errorf("Side = %d, want %d", h.Side, c.wantSide)
			}
			if h.MapX != c.wantX || h.MapY != c.wantY {
				t.Errorf("hit cell = (%d, %d), want (%d, %d)", h.MapX, h.MapY, c.wantX, c.wantY)
			}
			if h.Cell != Wall {
				t.Errorf("Cell = %q, want '#'", h.Cell)
			}
		})
	}
}

// One direction component exactly 0, starting exactly on a grid line.
func TestCastRayAxisAlignedOnGridLine(t *testing.T) {
	m := mustParse(t, room)
	h := CastRay(m, 3.0, 2.5, 0, 1)
	if math.Abs(h.PerpDist-3.5) > eps || h.Side != 1 || h.MapY != 6 {
		t.Errorf("got %+v, want PerpDist 3.5, Side 1, MapY 6", h)
	}
	if math.IsNaN(h.PerpDist) {
		t.Error("PerpDist is NaN")
	}
}

// Perpendicular distance is the same for every column looking at a flat wall:
// rays of dir + plane*cameraX all report the distance along dir (no fisheye).
func TestCastRayNoFisheye(t *testing.T) {
	m := mustParse(t, room)
	const dirX, dirY = 1.0, 0.0
	const planeX, planeY = 0.0, 0.66
	// 2 cells from the east wall face (x=6), the edge rays (cameraX ±1) spread
	// ±1.32 cells sideways, staying inside y 2.18-4.82, so all hit that wall.
	for _, cameraX := range []float64{-1, -0.5, 0, 0.5, 1} {
		h := CastRay(m, 4.0, 3.5, dirX+planeX*cameraX, dirY+planeY*cameraX)
		if math.Abs(h.PerpDist-2.0) > eps || h.Side != 0 || h.MapX != 6 {
			t.Errorf("cameraX %v: got %+v, want PerpDist 2 on the east wall", cameraX, h)
		}
	}
}

func TestCastRayDiagonalIntoCorner(t *testing.T) {
	m := mustParse(t, room)
	h := CastRay(m, 1.5, 1.5, -1, -1)
	if math.Abs(h.PerpDist-0.5) > eps {
		t.Errorf("PerpDist = %v, want 0.5", h.PerpDist)
	}
	if h.Cell != Wall || h.MapX > 1 || h.MapY > 1 || (h.MapX == 1 && h.MapY == 1) {
		t.Errorf("hit %+v, want a wall cell next to the corner", h)
	}
}

func TestCastRayDiagonalHitsPillar(t *testing.T) {
	m := mustParse(t, "#######\n"+
		"#P....#\n"+
		"#.....#\n"+
		"#..#..#\n"+
		"#.....#\n"+
		"#.....#\n"+
		"#######\n")
	// From (1.5, 1.5) toward the pillar at (3, 3) along the diagonal.
	h := CastRay(m, 1.5, 1.5, 1, 1)
	if h.MapX != 3 || h.MapY != 3 {
		t.Fatalf("hit cell = (%d, %d), want the pillar (3, 3)", h.MapX, h.MapY)
	}
	if math.Abs(h.PerpDist-1.5) > eps {
		t.Errorf("PerpDist = %v, want 1.5", h.PerpDist)
	}
}

func TestCastRayZeroDirection(t *testing.T) {
	m := mustParse(t, room)
	h := CastRay(m, 2.5, 2.5, 0, 0)
	if !math.IsInf(h.PerpDist, 1) {
		t.Errorf("PerpDist = %v, want +Inf", h.PerpDist)
	}
}

func TestOutsideIsSolid(t *testing.T) {
	m := mustParse(t, room)
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {7, 3}, {3, 7}} {
		if !m.IsSolid(p[0], p[1]) {
			t.Errorf("IsSolid(%d, %d) = false, want true outside the map", p[0], p[1])
		}
	}
	if m.IsSolid(1, 1) {
		t.Error("IsSolid(1, 1) = true on the spawn cell, want false")
	}
}

// A red target block stops rays like a wall and reports itself.
func TestCastRayHitsTarget(t *testing.T) {
	m := mustParse(t, "#######\n"+
		"#P..T.#\n"+
		"#.....#\n"+
		"#######\n")
	h := CastRay(m, 1.5, 1.5, 1, 0) // east, toward the T at x=4
	if h.Cell != Target || h.MapX != 4 || math.Abs(h.PerpDist-2.5) > eps {
		t.Errorf("hit %+v, want the target at (4, 1), PerpDist 2.5", h)
	}
}

// WallX is the hit point's position along the face: the y fraction on an
// east/west face, the x fraction on a north/south face.
func TestCastRayWallX(t *testing.T) {
	m := mustParse(t, room) // faces at x=1, x=6, y=1, y=6
	if h := CastRay(m, 2.5, 3.25, 1, 0); math.Abs(h.WallX-0.25) > eps {
		t.Errorf("east ray from y=3.25: WallX = %v, want 0.25", h.WallX)
	}
	if h := CastRay(m, 2.75, 3.5, 0, 1); math.Abs(h.WallX-0.75) > eps {
		t.Errorf("south ray from x=2.75: WallX = %v, want 0.75", h.WallX)
	}
	// A slanted ray: whatever face it hits, WallX stays within [0, 1).
	if h := CastRay(m, 2.5, 2.4, 1, 0.5); h.WallX < 0 || h.WallX >= 1 {
		t.Errorf("WallX = %v, want within [0, 1)", h.WallX)
	}
}

// Rays (walls) see past props; ShotDistance finds the first one that blocks
// shots (M2).
func TestShotDistance(t *testing.T) {
	m, err := Parse([]byte("##########\n#P.,.c...#\n##########\n"))
	if err != nil {
		t.Fatal(err)
	}
	if hit := CastRay(m, 1.5, 1.5, 1, 0); hit.MapX != 9 {
		t.Errorf("ray stopped at x %d, want the wall at 9", hit.MapX)
	}
	if d := ShotDistance(m, 1.5, 1.5, 1, 0, 100); math.Abs(d-3.5) > 1e-9 {
		t.Errorf("shot distance = %v, want 3.5 (the crate's cell starts at x 5)", d)
	}
	if d := ShotDistance(m, 1.5, 1.5, 1, 0, 2); !math.IsInf(d, 1) {
		t.Errorf("shot distance with max 2 = %v, want +Inf", d)
	}
	if !LineOfSight(m, 1.5, 1.5, 4.5, 1.5) || LineOfSight(m, 1.5, 1.5, 7.5, 1.5) {
		t.Error("line of sight: want clear over debris, blocked by the crate")
	}
}

// A ray stops at a door's panel in the middle of its cell; as the door
// slides open, rays through the gap go on (M3).
func TestCastRayDoor(t *testing.T) {
	m, err := Parse([]byte("#####\n#P..#\n##D##\n#...#\n#####\n"))
	if err != nil {
		t.Fatal(err)
	}
	hit := CastRay(m, 2.5, 1.5, 0, 1) // straight south into the closed door
	if hit.Cell != Door || math.Abs(hit.PerpDist-1) > 1e-9 || hit.Side != 1 || math.Abs(hit.WallX-0.5) > 1e-9 {
		t.Errorf("closed door: %+v, want the door at 1.0, side 1, WallX 0.5", hit)
	}
	m.Doors[0].Open = 0.5 // the panel now covers 0.5-1 of the cell's width
	if hit := CastRay(m, 2.7, 1.5, 0, 1); hit.Cell != Door || math.Abs(hit.WallX-0.2) > 1e-9 {
		t.Errorf("half open, x 2.7: %+v, want the panel with WallX 0.2", hit)
	}
	if hit := CastRay(m, 2.2, 1.5, 0, 1); hit.Cell == Door || hit.MapY != 4 {
		t.Errorf("half open, x 2.2: %+v, want through the gap to the wall at y 4", hit)
	}
	m.Doors[0].Open = 1
	if LineOfSight(m, 2.5, 1.5, 2.5, 3.5) != true {
		t.Error("open door blocks line of sight")
	}
}
