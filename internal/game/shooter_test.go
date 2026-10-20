package game

import (
	"math"
	"testing"
)

// shooterState puts a shooter (from the map) and the player in a long room.
func shooterState(t *testing.T, layout string) *State {
	t.Helper()
	s := stateFrom(t, layout)
	if len(s.Enemies) != 1 || s.Enemies[0].Kind != KindShooter {
		t.Fatalf("enemies = %+v, want one shooter", s.Enemies)
	}
	return s
}

func TestShooterFromMap(t *testing.T) {
	s := shooterState(t, "#######\n#P..S.#\n#######\n")
	if s.Enemies[0].HP != ShooterHP {
		t.Errorf("shooter HP = %d, want %d", s.Enemies[0].HP, ShooterHP)
	}
}

// With a wall in the way it never fires, even when close.
func TestShooterNeedsLineOfSight(t *testing.T) {
	s := shooterState(t, "#########\n"+
		"#P.#....#\n"+
		"#..#.S..#\n"+
		"#########\n")
	s.Enemies[0].X, s.Enemies[0].Y = 5.5, 1.5 // just past the wall, in line with the player
	shots := 0
	for range 3 * 30 {
		Update(s, Input{}, frameDT)
		shots += count(s, EventEnemyShot)
		// Keep it on its side of the wall: this test is about sight, not walking.
		s.Enemies[0].X, s.Enemies[0].Y = 5.5, 1.5
	}
	if shots != 0 {
		t.Errorf("shooter fired %d times through a wall", shots)
	}
}

// In view and range: the first shot comes after the aiming delay, then one
// every ShooterInterval.
func TestShooterFireTiming(t *testing.T) {
	s := shooterState(t, "##########\n#P...S...#\n##########\n") // 4 cells apart
	var times []float64
	for range 5 * 30 {
		Update(s, Input{}, frameDT)
		if count(s, EventEnemyShot) > 0 {
			times = append(times, s.Time)
		}
		s.Player.HP = PlayerMaxHP // keep the player alive to keep counting
	}
	if len(times) < 3 {
		t.Fatalf("shots in 5 s = %d, want at least 3", len(times))
	}
	// The delay starts counting on the frame after spotting, plus float
	// rounding: up to 2-3 frames (0.1 s) late is fine.
	if times[0] < ShooterFirstShotDelay || times[0] > ShooterFirstShotDelay+0.1 {
		t.Errorf("first shot at %.2f s, want about %.1f s", times[0], ShooterFirstShotDelay)
	}
	if gap := times[1] - times[0]; math.Abs(gap-ShooterInterval) > 0.05 {
		t.Errorf("time between shots = %.2f s, want %.1f", gap, ShooterInterval)
	}
}

func TestShooterHitChance(t *testing.T) {
	if c := ShooterHitChance(0); c != ShooterAccNear {
		t.Errorf("chance at 0 cells = %v, want %v", c, ShooterAccNear)
	}
	if near, far := ShooterHitChance(2), ShooterHitChance(5); far >= near {
		t.Errorf("chance at 5 cells (%v) not lower than at 2 cells (%v)", far, near)
	}
	if c := ShooterHitChance(50); c != ShooterAccMin {
		t.Errorf("chance at 50 cells = %v, want the floor %v", c, ShooterAccMin)
	}
}

// Over many shots, a near shooter hits clearly more often than a far one
// (the fixed random seed makes this repeatable).
func TestShooterHitsMoreWhenClose(t *testing.T) {
	rate := func(layout string) float64 {
		s := shooterState(t, layout)
		shots, hits := 0, 0
		for range 60 * 30 {
			Update(s, Input{}, frameDT)
			shots += count(s, EventEnemyShot)
			hits += count(s, EventPlayerHurt)
			s.Player.HP = PlayerMaxHP
		}
		return float64(hits) / float64(shots)
	}
	near := rate("############\n#P..S......#\n############\n") // 3 cells: ~57%
	far := rate("############\n#P.....S...#\n############\n")  // 6 cells: ~25%
	if near < far+0.15 {
		t.Errorf("hit rate near %.2f, far %.2f; want near clearly higher", near, far)
	}
}

// Too close: the shooter backs away.
func TestShooterRetreats(t *testing.T) {
	s := shooterState(t, "###########\n#P.S......#\n###########\n")
	x0 := s.Enemies[0].X // 2 cells from the player
	run(s, Input{}, 15, 0)
	if s.Enemies[0].X <= x0 {
		t.Errorf("shooter x %.2f -> %.2f, want it to back away (grow)", x0, s.Enemies[0].X)
	}
}

// Out of sight, it walks toward the player.
func TestShooterApproachesWhenHidden(t *testing.T) {
	s := shooterState(t, "##########\n"+
		"#P.......#\n"+
		"######.###\n"+
		"#......S.#\n"+
		"##########\n")
	run(s, Input{}, 3*30, 0)
	e := s.Enemies[0]
	if e.Y > 2.6 {
		t.Errorf("shooter at (%.2f, %.2f) after 3 s, want it up through the gap", e.X, e.Y)
	}
}

func TestTwoShotsKillShooter(t *testing.T) {
	s := shooterState(t, "##########\n#P...S...#\n##########\n")
	run(s, Input{Fire: true}, 15, 0) // 0.5 s: 2 shots
	if len(s.Enemies) != 0 || s.Kills != 1 {
		t.Errorf("enemies left %d, kills %d; want 0 and 1", len(s.Enemies), s.Kills)
	}
}
