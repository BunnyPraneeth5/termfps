package game

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Shooting a red barrel blows it up: it hurts enemies near it (kills
// count), leaves debris, and doesn't reach a player far away (M14).
func TestShootRedBarrel(t *testing.T) {
	s := stateFrom(t, "##########\n#P....B..#\n##########\n")
	s.Enemies = []Enemy{newZombie(7.5, 1.5)}
	Update(s, Input{Fire: true}, frameDT)
	if s.Map.At(6, 1) != world.Debris || count(s, EventExplosion) != 1 {
		t.Fatalf("after shooting the barrel: cell %q, %d explosions; want debris and 1", s.Map.At(6, 1), count(s, EventExplosion))
	}
	if len(s.Enemies) != 0 || s.Kills != 1 {
		t.Errorf("zombie next to the barrel: %d enemies left, %d kills; want 0 and 1", len(s.Enemies), s.Kills)
	}
	if s.Player.HP != PlayerMaxHP {
		t.Errorf("player 5 cells away lost HP (%d)", s.Player.HP)
	}
}

// A barrel's blast sets off red barrels in reach, a moment later; a plain
// barrel stays.
func TestBarrelChain(t *testing.T) {
	s := stateFrom(t, "############\n#P...B.B.b.#\n############\n")
	Update(s, Input{Fire: true}, frameDT)
	if s.Map.At(5, 1) != world.Debris || s.Map.At(7, 1) != world.Debris {
		t.Fatalf("cells %q %q, want both red barrels gone at once (no double blast)", s.Map.At(5, 1), s.Map.At(7, 1))
	}
	if count(s, EventExplosion) != 1 {
		t.Errorf("%d explosions in the first frame, want 1 (the chained one waits)", count(s, EventExplosion))
	}
	if booms := run(s, Input{}, 10, EventExplosion); booms != 1 {
		t.Errorf("%d chained explosions, want 1", booms)
	}
	if s.Map.At(9, 1) != world.Barrel {
		t.Error("a plain gray barrel blew up")
	}
}

// Standing in acid burns AcidDamage every AcidInterval; stepping out stops it.
func TestAcid(t *testing.T) {
	s := stateFrom(t, "######\n#P~..#\n######\n")
	s.Player.X = 2.5
	burns := run(s, Input{}, 30, EventAcidBurn) // 1 s
	if burns != 2 || s.Player.HP != PlayerMaxHP-2*AcidDamage {
		t.Errorf("1 s in acid: %d burns, HP %d; want 2 and %d", burns, s.Player.HP, PlayerMaxHP-2*AcidDamage)
	}
	s.Player.X = 3.5
	if burns := run(s, Input{}, 30, EventAcidBurn); burns != 0 {
		t.Errorf("%d burns after stepping out", burns)
	}
}

// Nothing spawns in acid.
func TestNoSpawnInAcid(t *testing.T) {
	m, err := world.Parse([]byte("#####\n#P~.#\n#####\n"))
	if err != nil {
		t.Fatal(err)
	}
	if freeCell(m, 2, 1) || !freeCell(m, 3, 1) {
		t.Error("acid counts as a free cell, or floor doesn't")
	}
}

// A run on a cloned map doesn't change the original: red barrels and the
// lamp light they shadow are back for the next run.
func TestBarrelsBackNextRun(t *testing.T) {
	m, err := world.Parse([]byte("##########\n#Pl..B...#\n##########\n"))
	if err != nil {
		t.Fatal(err)
	}
	reach := len(m.Lamps[0].Reach)
	s := NewState(m.Clone())
	s.Spawning = false
	Update(s, Input{Fire: true}, frameDT)
	if s.Map.At(5, 1) != world.Debris {
		t.Fatal("barrel didn't blow up")
	}
	if m.At(5, 1) != world.ExplosiveBarrel || len(m.Lamps[0].Reach) != reach {
		t.Errorf("original map changed: cell %q, lamp reach %d (was %d)", m.At(5, 1), len(m.Lamps[0].Reach), reach)
	}
}
