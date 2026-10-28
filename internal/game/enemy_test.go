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
	s := NewState(m)
	s.Spawning = false // these tests are about the map's own enemies
	return s
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

// Three hits kill a zombie (75 HP, 25 per hit): it's removed and counts as
// a kill.
func TestShootingKillsZombie(t *testing.T) {
	s := stateFrom(t, "##########\n#P......Z#\n##########\n")
	hits, kills := 0, 0
	for range 60 { // 2 s of holding fire: 6 shots, the zombie walks closer
		Update(s, Input{Fire: true}, frameDT)
		hits += count(s, EventEnemyHit)
		kills += count(s, EventEnemyKilled)
	}
	if hits != 3 || kills != 1 || len(s.Enemies) != 0 || s.Kills != 1 {
		t.Errorf("hits %d, kills %d, enemies left %d, Kills %d; want 3, 1, 0, 1", hits, kills, len(s.Enemies), s.Kills)
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

// In reach, a zombie hits after a short delay, then once per second for a
// random 7-10 HP; the player dies at 0 HP and the game stops.
func TestZombieMeleeAndDeath(t *testing.T) {
	s := stateFrom(t, "######\n#PZ..#\n######\n")
	s.Enemies[0].X = s.Player.X + 0.6 // already in reach
	hurt := 0
	var times []float64
	for range 20 * 30 {
		hpBefore := s.Player.HP
		Update(s, Input{}, frameDT)
		if count(s, EventPlayerHurt) > 0 {
			hurt++
			times = append(times, s.Time)
			if d := hpBefore - s.Player.HP; !s.Player.Dead() && (d < ZombieMeleeDamageMin || d > ZombieMeleeDamageMax) {
				t.Errorf("a hit took %d HP, want %d-%d", d, ZombieMeleeDamageMin, ZombieMeleeDamageMax)
			}
		}
		if s.Player.Dead() {
			break
		}
	}
	// 100 HP at 7-10 per hit: 10 to 15 hits.
	if hurt < 10 || hurt > 15 || !s.Player.Dead() || s.Player.HP != 0 {
		t.Fatalf("hits %d, HP %d, dead %v; want 10-15 hits and dead", hurt, s.Player.HP, s.Player.Dead())
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

// Healing starts 4 s after the last hit, at 5 HP per second, up to 100.
func TestRegeneration(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	hurtPlayer(s, 40) // 60 HP left
	run(s, Input{}, int((RegenDelay-0.1)*30), 0)
	if s.Player.HP != 60 {
		t.Fatalf("HP %d just before %.0f s, want still 60", s.Player.HP, RegenDelay)
	}
	run(s, Input{}, 2*30+6, 0) // a bit over 2 s of healing
	if s.Player.HP < 69 || s.Player.HP > 71 {
		t.Errorf("HP %d after ~2 s of healing, want about 70", s.Player.HP)
	}
	run(s, Input{}, 20*30, 0)
	if s.Player.HP != PlayerMaxHP {
		t.Errorf("HP %d after a long rest, want capped at %d", s.Player.HP, PlayerMaxHP)
	}
}

// A new hit restarts the wait.
func TestRegenerationResetsOnHit(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	hurtPlayer(s, 40)
	run(s, Input{}, 3*30, 0)
	hurtPlayer(s, 10) // 50 HP, wait starts over
	run(s, Input{}, 3*30, 0)
	if s.Player.HP != 50 {
		t.Errorf("HP %d 3 s after the second hit, want 50 (no healing yet)", s.Player.HP)
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

// A crate is cover: shots stop at it (M2). Debris isn't.
func TestCrateProtectsZombie(t *testing.T) {
	s := stateFrom(t, "#########\n#P..c..Z#\n#########\n")
	Update(s, Input{Fire: true}, frameDT)
	if count(s, EventEnemyHit) != 0 {
		t.Error("shot went through a crate and hit the zombie")
	}
	s = stateFrom(t, "#########\n#P..,..Z#\n#########\n")
	Update(s, Input{Fire: true}, frameDT)
	if count(s, EventEnemyHit) != 1 {
		t.Error("shot didn't pass over debris to the zombie")
	}
}

// The player can't walk through a crate (M2).
func TestPlayerBlockedByCrate(t *testing.T) {
	s := stateFrom(t, "#######\n#P.c..#\n#######\n")
	for range 60 {
		Update(s, Input{Forward: true}, frameDT)
	}
	if s.Player.X > 3-Radius+1e-9 {
		t.Errorf("player at x %v, want stopped before the crate at x 3", s.Player.X)
	}
}
