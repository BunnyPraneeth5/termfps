package world

import "testing"

// A wall splits the room; the way around goes through the gap at the top.
const maze = "#######\n" +
	"#.....#\n" +
	"#.###.#\n" +
	"#.#P#.#\n" +
	"#.#.#.#\n" +
	"#######\n"

func TestDistanceFieldAroundWalls(t *testing.T) {
	m := mustParse(t, maze)
	d := DistanceField(m, 3, 3, nil) // from P
	at := func(x, y int) int { return d[y*m.W+x] }
	if at(3, 3) != 0 || at(3, 4) != 1 {
		t.Errorf("start %d, below %d; want 0 and 1", at(3, 3), at(3, 4))
	}
	// (1, 1) is cut off from P by walls: P's pocket only opens downward
	// into (3, 4), which is closed too, so the outer ring is unreachable.
	if at(1, 1) != -1 || at(5, 4) != -1 {
		t.Errorf("cut-off cells = %d, %d; want -1", at(1, 1), at(5, 4))
	}
	if at(0, 0) != -1 || at(2, 2) != -1 {
		t.Error("solid cells should be -1")
	}

	// From the top-left corner, (5, 4) is reached around the wall block:
	// 4 steps right along the top, then 3 down.
	d = DistanceField(m, 1, 1, d)
	if got := d[4*m.W+5]; got != 7 {
		t.Errorf("distance (1, 1) -> (5, 4) = %d, want 7 (around the walls)", got)
	}
	if got := d[4*m.W+1]; got != 3 {
		t.Errorf("distance (1, 1) -> (1, 4) = %d, want 3", got)
	}
}

func TestDistanceFieldReusesBuffer(t *testing.T) {
	m := mustParse(t, room)
	buf := make([]int, 0, m.W*m.H)
	d := DistanceField(m, 1, 1, buf)
	if &d[0] != &buf[:1][0] {
		t.Error("DistanceField allocated a new slice despite enough capacity")
	}
	// Open 5x5 room: the far corner (5, 5) is 8 steps from (1, 1).
	if got := d[5*m.W+5]; got != 8 {
		t.Errorf("distance to (5, 5) = %d, want 8", got)
	}
}

func TestLineOfSight(t *testing.T) {
	m := mustParse(t, "#######\n"+
		"#P....#\n"+
		"#..#..#\n"+
		"#.....#\n"+
		"#######\n")
	if !LineOfSight(m, 1.5, 1.5, 5.5, 1.5) {
		t.Error("clear row: no line of sight")
	}
	if LineOfSight(m, 1.5, 2.5, 5.5, 2.5) {
		t.Error("pillar at (3, 2) in the way, but line of sight is true")
	}
	if !LineOfSight(m, 2.5, 2.5, 2.5, 2.5) {
		t.Error("same point: want line of sight")
	}
}

func TestZombieStartsAndClone(t *testing.T) {
	m := mustParse(t, "######\n"+
		"#P.Z.#\n"+
		"#..Z.#\n"+
		"######\n")
	if len(m.EnemyStarts) != 2 || m.EnemyStarts[0] != (EnemyStart{Kind: ZombieStart, X: 3.5, Y: 1.5}) {
		t.Fatalf("EnemyStarts = %+v, want two zombies, first at (3.5, 1.5)", m.EnemyStarts)
	}
	if m.At(3, 1) != Empty {
		t.Errorf("Z cell = %q after parsing, want floor", m.At(3, 1))
	}
	c := m.Clone()
	c.Set(2, 1, Target)
	c.EnemyStarts[0].X = 99
	if m.At(2, 1) != Empty || m.EnemyStarts[0].X != 3.5 {
		t.Error("changing the clone changed the original")
	}
}
