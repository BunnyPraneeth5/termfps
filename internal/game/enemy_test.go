package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func stateFrom(t *testing.T, layout string) *State {
	t.Helper()
	m, err := world.Parse([]byte(layout))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return NewState(m)
}

// count returns how often e is in the current frame's events.
func count(s *State, e Event) int {
	n := 0
	for _, got := range s.Events {
		if got == e {
			n++
		}
	}
	return n
}

func TestZombieFromMap(t *testing.T) {
	s := stateFrom(t, "#######\n#P..Z.#\n#######\n")
	if len(s.Enemies) != 1 || s.Enemies[0].Kind != KindZombie || s.Enemies[0].HP != ZombieHP {
		t.Fatalf("enemies = %+v, want one full-health zombie", s.Enemies)
	}
	if s.Player.HP != PlayerMaxHP {
		t.Errorf("player HP = %d, want %d", s.Player.HP, PlayerMaxHP)
	}
}

// Four hits kill a zombie: it's removed and counts as a kill.
func TestShootingKillsZombie(t *testing.T) {
	s := stateFrom(t, "##########\n#P......Z#\n##########\n")
	hits, kills := 0, 0
	for range 60 { // 2 s of holding fire: 6 shots, the zombie walks closer
		Update(s, Input{Fire: true}, frameDT)
		hits += count(s, EventEnemyHit)
		kills += count(s, EventEnemyKilled)
	}
	if hits != 4 || kills != 1 || len(s.Enemies) != 0 || s.Kills != 1 {
		t.Errorf("hits %d, kills %d, enemies left %d, Kills %d; want 4, 1, 0, 1", hits, kills, len(s.Enemies), s.Kills)
	}
}

func TestShotMissesOffAxisZombie(t *testing.T) {
	s := stateFrom(t, "########\n#P.....#\n#.....Z#\n########\n")
	s.Enemies[0].X, s.Enemies[0].Y = 6.5, 2.5 // a full cell to the side of the shot
	Update(s, Input{Fire: true}, frameDT)
	if count(s, EventEnemyHit) != 0 {
		t.Error("shot hit a zombie a cell away from the line of fire")
	}
}

func TestWallProtectsZombie(t *testing.T) {
	s := stateFrom(t, "#########\n#P..#..Z#\n#########\n")
	Update(s, Input{Fire: true}, frameDT)
	if count(s, EventEnemyHit) != 0 {
		t.Error("shot went through a wall and hit the zombie")
	}
}

// A zombie walks the long way around a wall to reach a standing player.
func TestZombieWalksAroundWalls(t *testing.T) {
	s := stateFrom(t, "#########\n"+
		"#P......#\n"+
		"#######.#\n"+
		"#Z......#\n"+
		"#########\n") // path: right 6, up 2, left 6 = 14 cells at 2/s
	reached := false
	for range 10 * 30 {
		Update(s, Input{}, frameDT)
		if len(s.Enemies) == 1 && math.Hypot(s.Enemies[0].X-s.Player.X, s.Enemies[0].Y-s.Player.Y) <= ZombieMeleeRange {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("zombie at (%.2f, %.2f) didn't reach the player in 10 s", s.Enemies[0].X, s.Enemies[0].Y)
	}
}

// In reach, a zombie hits after a short delay, then once per second; the
// player dies at 0 HP and the game stops.
func TestZombieMeleeAndDeath(t *testing.T) {
	s := stateFrom(t, "######\n#PZ..#\n######\n")
	s.Enemies[0].X = s.Player.X + 0.6 // already in reach
	hurt := 0
	var times []float64
	for range 15 * 30 {
		Update(s, Input{}, frameDT)
		if count(s, EventPlayerHurt) > 0 {
			hurt++
			times = append(times, s.Time)
		}
		if s.Player.Dead() {
			break
		}
	}
	if hurt != 10 || !s.Player.Dead() || s.Player.HP != 0 {
		t.Fatalf("hits %d, HP %d, dead %v; want 10 hits of 10 HP and dead", hurt, s.Player.HP, s.Player.Dead())
	}
	if times[0] > ZombieFirstHitDelay+0.05 {
		t.Errorf("first hit at %.2f s, want about %.1f s", times[0], ZombieFirstHitDelay)
	}
	if gap := times[1] - times[0]; math.Abs(gap-ZombieMeleeInterval) > 0.05 {
		t.Errorf("time between hits = %.2f s, want %.1f", gap, ZombieMeleeInterval)
	}
	// After death nothing changes: time stops, no events.
	before := s.Time
	Update(s, Input{Forward: true, Fire: true}, frameDT)
	if s.Time != before || len(s.Events) != 0 {
		t.Error("game kept running after the player died")
	}
}

func TestHurtFlash(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	hurtPlayer(s, 10)
	if !s.Player.Hurting() || s.Player.HP != 90 {
		t.Fatalf("after a hit: HP %d, hurting %v; want 90 and true", s.Player.HP, s.Player.Hurting())
	}
	run(s, Input{}, 9, 0) // 0.3 s
	if s.Player.Hurting() {
		t.Error("hurt flash still on 0.3 s later")
	}
}

func TestTimeCounts(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	run(s, Input{}, 30, 0)
	if math.Abs(s.Time-1) > 1e-9 {
		t.Errorf("time after 30 frames = %v, want 1 s", s.Time)
	}
}
