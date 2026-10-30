package game

import (
	"math"
	"testing"
)

// The new kinds come by tier (D49): runners from 2, tanks from 3,
// exploders from 4.
func TestKindsByTier(t *testing.T) {
	s := survivalState(t)
	for tier, want := range map[int][]EnemyKind{
		1: {KindZombie, KindShooter},
		2: {KindZombie, KindShooter, KindRunner},
		3: {KindZombie, KindShooter, KindRunner, KindTank},
		4: {KindZombie, KindShooter, KindRunner, KindTank, KindExploder},
	} {
		got := map[EnemyKind]int{}
		for range 2000 {
			got[spawnKind(s, TierFor(tier))]++
		}
		if len(got) != len(want) {
			t.Errorf("tier %d spawns kinds %v, want exactly %v", tier, got, want)
		}
		for _, k := range want {
			if got[k] == 0 {
				t.Errorf("tier %d never spawned kind %d", tier, k)
			}
		}
	}
}

// A runner closes in faster than a zombie.
func TestRunnerFaster(t *testing.T) {
	s := stateFrom(t, "############\n#P.........#\n############\n")
	s.Enemies = []Enemy{newEnemy(KindRunner, 10.5, 1.5)}
	run(s, Input{}, 30, 0)
	runner := 10.5 - s.Enemies[0].X
	s = stateFrom(t, "############\n#P.........#\n############\n")
	s.Enemies = []Enemy{newZombie(10.5, 1.5)}
	run(s, Input{}, 30, 0)
	if zombie := 10.5 - s.Enemies[0].X; runner < 2*zombie {
		t.Errorf("in 1 s the runner moved %.2f, the zombie %.2f; want the runner about twice as fast", runner, zombie)
	}
}

// A tank hits hard and says so (the screen shakes, M16).
func TestTankHit(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Enemies = []Enemy{newEnemy(KindTank, s.Player.X+0.7, s.Player.Y)}
	hits := run(s, Input{}, 30, EventTankHit)
	lost := PlayerMaxHP - s.Player.HP
	if hits != 1 || lost < TankMeleeDamageMin || lost > TankMeleeDamageMax {
		t.Errorf("1 s next to a tank: %d hits, %d HP lost; want 1 hit of %d-%d", hits, lost, TankMeleeDamageMin, TankMeleeDamageMax)
	}
}

// Shots that would miss a zombie by a little still hit the bigger tank.
func TestTankEasierToHit(t *testing.T) {
	s := stateFrom(t, "########\n#P.....#\n########\n")
	off := (EnemyHitRadius + TankHitRadius) / 2
	s.Enemies = []Enemy{newZombie(5.5, 1.5+off)}
	if shotHitsEnemy(s, 1, 0, 10) >= 0 {
		t.Fatal("shot hit a zombie it passes beside")
	}
	s.Enemies = []Enemy{newEnemy(KindTank, 5.5, 1.5+off)}
	if shotHitsEnemy(s, 1, 0, 10) != 0 {
		t.Error("shot missed a tank at the same place")
	}
}

// An exploder that reaches the player lights its fuse and blows up after
// ExploderFuse seconds. It's gone, and it's not a kill.
func TestExploderBlowsUp(t *testing.T) {
	s := stateFrom(t, "#######\n#P....#\n#######\n")
	s.Enemies = []Enemy{newEnemy(KindExploder, s.Player.X+1.2, s.Player.Y)}
	Update(s, Input{}, frameDT)
	if count(s, EventExploderArmed) != 1 {
		t.Fatal("exploder in reach didn't light its fuse")
	}
	blinks := map[bool]int{}
	booms := 0
	for range int(ExploderFuse*30) + 1 {
		if len(s.Enemies) > 0 {
			blinks[s.Enemies[0].Flashing()]++
		}
		Update(s, Input{}, frameDT)
		booms += count(s, EventExplosion)
	}
	if blinks[true] == 0 || blinks[false] == 0 {
		t.Errorf("lit exploder drawn white %d frames, normal %d; want it to blink", blinks[true], blinks[false])
	}
	if booms != 1 || len(s.Enemies) != 0 || s.Kills != 0 {
		t.Fatalf("after the fuse: %d explosions, %d enemies, %d kills; want 1, 0, 0", booms, len(s.Enemies), s.Kills)
	}
	if s.Player.HP >= PlayerMaxHP-ExploderDamage/2 {
		t.Errorf("player HP %d after an exploder went off next to them, want a big loss", s.Player.HP)
	}
	if len(s.Booms) != 1 {
		t.Errorf("%d explosions on screen, want 1", len(s.Booms))
	}
}

// Shooting an exploder sets it off where it is: it hurts the enemies around
// it (they count as kills) but not a player out of reach.
func TestShotExploderBlowsUp(t *testing.T) {
	s := stateFrom(t, "##########\n#P.......#\n##########\n")
	s.Enemies = []Enemy{newEnemy(KindExploder, 6.5, 1.5), newEnemy(KindRunner, 7.5, 1.6)}
	s.Enemies[0].HP = 1
	Update(s, Input{Fire: true}, frameDT)
	if count(s, EventExplosion) != 1 || len(s.Enemies) != 0 || s.Kills != 2 {
		t.Errorf("after shooting the exploder: %d explosions, %d enemies, %d kills; want 1, 0, 2", count(s, EventExplosion), len(s.Enemies), s.Kills)
	}
	if s.Player.HP != PlayerMaxHP {
		t.Errorf("player 5 cells away lost HP (%d)", s.Player.HP)
	}
}

// Walls shield from blasts; damage falls off with distance.
func TestBlastFalloffAndWalls(t *testing.T) {
	s := stateFrom(t, "#######\n#P.#..#\n#######\n")
	addBlast(s, 4.5, 1.5, 50, 0) // behind the wall, 3 cells away... but the radius is 2.5
	addBlast(s, 2.5, 1.5, 50, 0) // 1 cell away, open
	Update(s, Input{}, frameDT)
	want := PlayerMaxHP - blastDamage(50, 1)
	if s.Player.HP != want {
		t.Errorf("HP %d, want %d (only the open blast, at 1 cell)", s.Player.HP, want)
	}
	if blastDamage(50, 0) != 50 || blastDamage(50, ExplosionRadius) != 0 || math.Abs(float64(blastDamage(50, ExplosionRadius/2))-25) > 1 {
		t.Error("blast damage doesn't fall off linearly from 50 to 0")
	}
	if blastReaches(s.Map, 2.5, 1.5, 4.5, 1.5) {
		t.Error("blast reaches through a wall")
	}
}
