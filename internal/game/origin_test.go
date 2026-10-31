package game

import "testing"

// An enemy makes its "found you" sound once, when it first sees the player
// within NoticeRange; one behind a wall doesn't (M17).
func TestNotice(t *testing.T) {
	s := stateFrom(t, "############\n#P.....Z...#\n######.#####\n#..........#\n############\n")
	s.Enemies = append(s.Enemies, newEnemy(KindRunner, 9.5, 3.5)) // below the wall
	Update(s, Input{}, frameDT)
	o, ok := s.OriginOf(EventEnemyNoticed)
	if count(s, EventEnemyNoticed) != 1 || !ok || o.Kind != KindZombie || o.X < 7 {
		t.Fatalf("first frame: %d notices (%+v); want 1, the zombie's", count(s, EventEnemyNoticed), o)
	}
	if run(s, Input{}, 10, EventEnemyNoticed) != 0 {
		t.Error("the zombie noticed the player again")
	}
}

// A kill's origin is the killed enemy's place and kind, even when it wasn't
// last in the list (M17).
func TestKillOrigin(t *testing.T) {
	s := stateFrom(t, "##########\n#P...Z..Z#\n##########\n")
	s.Enemies[0].HP = 1
	Update(s, Input{Fire: true}, frameDT)
	o, ok := s.OriginOf(EventEnemyKilled)
	if !ok || o.X != 5.5 || o.Kind != KindZombie {
		t.Errorf("kill origin %+v (%v), want the zombie at x 5.5", o, ok)
	}
}

// A melee hit says which kind hit, from where.
func TestAttackOrigin(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Enemies = []Enemy{newEnemy(KindRunner, s.Player.X+0.6, s.Player.Y)}
	run(s, Input{}, 15, 0)
	for i := 0; i < 30 && !s.HasEvent(EventEnemyAttack); i++ {
		Update(s, Input{}, frameDT)
	}
	if o, ok := s.OriginOf(EventEnemyAttack); !ok || o.Kind != KindRunner {
		t.Errorf("attack origin %+v (%v), want the runner", o, ok)
	}
}

// Each gun's reload has its own event (M17).
func TestReloadEvents(t *testing.T) {
	s := newOpenState(t)
	giveAll(s)
	s.Player.Gun = Shotgun
	s.Player.Weapon().Mag = 2
	Update(s, Input{Reload: true}, frameDT)
	if !s.HasEvent(EventReloadStarted) || !s.HasEvent(EventShotgunReload) || s.HasEvent(EventRifleReload) {
		t.Errorf("shotgun reload events %v, want the reload and the shotgun's", s.Events)
	}
}

// Origins are cleared every frame, like Events.
func TestOriginsCleared(t *testing.T) {
	s := stateFrom(t, "########\n#P...Z.#\n########\n")
	Update(s, Input{Fire: true}, frameDT)
	if len(s.Origins) == 0 {
		t.Fatal("no origins after hitting a zombie")
	}
	Update(s, Input{}, frameDT)
	if _, ok := s.OriginOf(EventEnemyHit); ok {
		t.Error("last frame's hit is still in Origins")
	}
}
