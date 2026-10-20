package main

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// The real map is embedded into the binary, so a broken arena.txt must fail
// here, not when someone runs the game.
func TestEmbeddedArenaParses(t *testing.T) {
	m, err := world.Parse(arenaFile)
	if err != nil {
		t.Fatalf("maps/arena.txt: %v", err)
	}
	if m.W != 16 || m.H != 16 {
		t.Errorf("arena is %dx%d, want 16x16 (PRD)", m.W, m.H)
	}
}

// Every open cell must be reachable from the spawn, so the whole map can be
// walked (PRD success criterion).
func TestEmbeddedArenaAllReachable(t *testing.T) {
	m, err := world.Parse(arenaFile)
	if err != nil {
		t.Fatalf("maps/arena.txt: %v", err)
	}
	seen := make([]bool, len(m.Cells))
	start := int(m.SpawnY)*m.W + int(m.SpawnX)
	queue := []int{start}
	seen[start] = true
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%m.W, i/m.W
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			j := ny*m.W + nx
			if !m.Cells[j].Solid() && !seen[j] {
				seen[j] = true
				queue = append(queue, j)
			}
		}
	}
	for i, c := range m.Cells {
		if !c.Solid() && !seen[i] {
			t.Errorf("open cell (%d, %d) can't be reached from the spawn", i%m.W, i/m.W)
		}
	}
}
