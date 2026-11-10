package game

import "testing"

// Shots and hits are counted per shot: a miss, a hit, and a shotgun blast
// whose pellets all hit count as one shot each.
func TestStatsShotsAndHits(t *testing.T) {
	s := stateFrom(t, "##########\n#P.......#\n##########\n")
	s.Spawning = false
	Update(s, Input{Fire: true}, frameDT) // nothing there: a miss
	if s.Stats.Shots != 1 || s.Stats.Hits != 0 {
		t.Fatalf("after a miss: %d shots, %d hits", s.Stats.Shots, s.Stats.Hits)
	}
	if _, ok := (RunStats{}).Accuracy(); ok {
		t.Error("accuracy without a shot should say there is none")
	}
	s.Enemies = []Enemy{newEnemy(KindTank, 4.5, 1.5)}
	s.Player.Guns[Shotgun] = newGun(Shotgun)
	s.Player.Gun = Shotgun
	Update(s, Input{Fire: true}, 1) // past the pistol's cooldown
	if s.Stats.Shots != 2 || s.Stats.Hits != 1 {
		t.Errorf("after a shotgun blast into a tank: %d shots, %d hits; want 2 and 1", s.Stats.Shots, s.Stats.Hits)
	}
	if acc, ok := s.Stats.Accuracy(); !ok || acc != 0.5 {
		t.Errorf("accuracy = %v, %v; want 0.5", acc, ok)
	}
}

// Kills are counted per enemy kind, and add up to Kills.
func TestStatsKillsByKind(t *testing.T) {
	s := stateFrom(t, "##########\n#P.......#\n##########\n")
	s.Spawning = false
	s.Enemies = []Enemy{newEnemy(KindRunner, 3.5, 1.5), newEnemy(KindZombie, 5.5, 1.5)}
	for i := 0; i < 20 && len(s.Enemies) > 0; i++ {
		Update(s, Input{Fire: true}, 0.5)
	}
	k := s.Stats.KillsBy
	if k[KindRunner] != 1 || k[KindZombie] != 1 || s.Kills != 2 {
		t.Errorf("kills by kind %v, total %d; want one runner and one zombie", k, s.Kills)
	}
}

// Damage taken is the HP lost (not what armor took), and the stats say
// what took the last HP: an enemy's kind, an explosion, or acid.
func TestStatsDamageAndKiller(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Player.Armor = 10
	hurtPlayerBy(s, 20, Cause{Kind: CauseEnemy, Enemy: KindTank}) // armor takes 10
	if s.Stats.DamageTaken != 10 || s.Stats.Killer.Kind != CauseUnknown {
		t.Fatalf("after a hit: damage taken %d, killer %+v; want 10 and none yet", s.Stats.DamageTaken, s.Stats.Killer)
	}
	hurtPlayerBy(s, 500, Cause{Kind: CauseEnemy, Enemy: KindTank})
	if s.Stats.DamageTaken != 100 || s.Stats.Killer != (Cause{Kind: CauseEnemy, Enemy: KindTank}) {
		t.Errorf("after dying: damage taken %d, killer %+v; want 100 and the tank", s.Stats.DamageTaken, s.Stats.Killer)
	}

	acid := stateFrom(t, "#####\n#P~.#\n#####\n")
	acid.Spawning = false
	acid.Player.X, acid.Player.HP = 2.5, 1
	for i := 0; i < 10 && !acid.Player.Dead(); i++ {
		Update(acid, Input{}, 0.2)
	}
	if acid.Stats.Killer.Kind != CauseAcid {
		t.Errorf("dying in acid: killer %+v", acid.Stats.Killer)
	}

	boom := stateFrom(t, "#####\n#P..#\n#####\n")
	boom.Player.HP = 1
	addBlast(boom, 2.0, 1.5, 50, 0)
	updateBlasts(boom, frameDT)
	if !boom.Player.Dead() || boom.Stats.Killer.Kind != CauseExplosion {
		t.Errorf("dying in a blast: dead %v, killer %+v", boom.Player.Dead(), boom.Stats.Killer)
	}
}
