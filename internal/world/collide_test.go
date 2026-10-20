package world

import (
	"math/rand"
	"testing"
)

const r = 0.2 // player radius used by the game

// Walking straight into a wall stops the box at the wall face, not inside it.
func TestTryMoveBlockedHeadOn(t *testing.T) {
	m := mustParse(t, room) // east wall face at x=6
	x, y := 5.5, 3.5
	for range 20 {
		x, y = TryMove(m, x, y, 0.1, 0, r)
	}
	if x+r > 6 {
		t.Errorf("box right edge %v is inside the east wall (face at 6)", x+r)
	}
	if x+r < 5.85 {
		t.Errorf("stopped at x=%v, too far from the wall", x)
	}
	if y != 3.5 {
		t.Errorf("y changed to %v walking straight east", y)
	}
}

// Walking diagonally into a wall slides along it.
func TestTryMoveSlidesAlongWall(t *testing.T) {
	m := mustParse(t, room)
	x, y := 5.7, 3.5 // box already touching the east wall (5.7+0.2 < 6)
	for range 10 {
		x, y = TryMove(m, x, y, 0.1, 0.1, r) // south-east
	}
	if x+r > 6 {
		t.Errorf("box entered the east wall: x=%v", x)
	}
	if y <= 4.4 {
		t.Errorf("y = %v, want it to keep sliding south (about 4.5)", y)
	}
}

// Moving diagonally toward an outside corner of a pillar must not cut into it.
func TestTryMoveOutsideCorner(t *testing.T) {
	m := mustParse(t, "#######\n"+
		"#P....#\n"+
		"#.....#\n"+
		"#..#..#\n"+
		"#.....#\n"+
		"#.....#\n"+
		"#######\n")
	// Pillar occupies x 3-4, y 3-4. Start up-left of its corner.
	x, y := 2.5, 2.5
	for range 20 {
		x, y = TryMove(m, x, y, 0.05, 0.05, r)
		if boxHitsWall(m, x, y, r) {
			t.Fatalf("box overlaps a wall at (%v, %v)", x, y)
		}
	}
}

// Thousands of random moves around the arena never end inside a wall.
func TestTryMoveRandomNeverInsideWall(t *testing.T) {
	m := mustParse(t, "########\n"+
		"#P.#...#\n"+
		"#..#.#.#\n"+
		"#....#.#\n"+
		"##.###.#\n"+
		"#......#\n"+
		"########\n")
	rng := rand.New(rand.NewSource(1)) // fixed seed: same moves every run
	x, y := m.SpawnX, m.SpawnY
	for i := range 20000 {
		dx := (rng.Float64()*2 - 1) * 0.3 // up to 3 cells/s at 10 FPS (dt clamp)
		dy := (rng.Float64()*2 - 1) * 0.3
		x, y = TryMove(m, x, y, dx, dy, r)
		if boxHitsWall(m, x, y, r) {
			t.Fatalf("step %d: box overlaps a wall at (%v, %v)", i, x, y)
		}
	}
}

func TestBoxTouchingIsNotHit(t *testing.T) {
	m := mustParse(t, room)
	if boxHitsWall(m, 5.8, 3.5, r) { // right edge exactly at the face x=6
		t.Error("box touching the wall face counts as a hit, want free")
	}
	if !boxHitsWall(m, 5.81, 3.5, r) {
		t.Error("box 0.01 into the wall doesn't count as a hit")
	}
}

// Target blocks are solid for movement too.
func TestTryMoveBlockedByTarget(t *testing.T) {
	m := mustParse(t, "#######\n"+
		"#P..T.#\n"+
		"#.....#\n"+
		"#######\n")
	x, y := 2.5, 1.5
	for range 20 {
		x, y = TryMove(m, x, y, 0.1, 0, r)
	}
	if x+r > 4 {
		t.Errorf("box edge %v went into the target (face at x=4)", x+r)
	}
}
