package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Two rooms joined by a gap: plenty of cells hidden from the player.
const twoRooms = "####################\n" +
	"#P.................#\n" +
	"#..................#\n" +
	"#########.##########\n" +
	"#..................#\n" +
	"#..................#\n" +
	"####################\n"

func survivalState(t *testing.T) *State {
	t.Helper()
	m, err := world.Parse([]byte(twoRooms))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return NewState(m) // spawner on
}

func TestTierForCurve(t *testing.T) {
	one := TierFor(1)
	if one.HPMul != 1 || one.DamageMul != 1 || one.SpeedMul != 1 {
		t.Errorf("tier 1 = %+v, want no scaling", one)
	}
	prev := one
	for tier := 2; tier <= 30; tier++ {
		st := TierFor(tier)
		if st.HPMul < prev.HPMul || st.DamageMul < prev.DamageMul || st.SpawnEvery > prev.SpawnEvery || st.MaxAlive < prev.MaxAlive {
			t.Errorf("tier %d got easier than tier %d: %+v vs %+v", tier, tier-1, st, prev)
		}
		prev = st
	}
	if prev.SpeedMul != 1.5 || prev.SpawnEvery != 2 || prev.MaxAlive != 14 || prev.ShooterShare != 0.5 {
		t.Errorf("tier 30 = %+v, want the caps: speed 1.5, spawn every 2 s, 14 alive, half shooters", prev)
	}
}

func TestTierUpByKills(t *testing.T) {
	s := survivalState(t)
	s.Kills = TierKills
	Update(s, Input{}, frameDT)
	if s.Tier != 2 || count(s, EventTierUp) != 1 {
		t.Fatalf("tier %d, tier-up events %d after %d kills; want 2 and 1", s.Tier, count(s, EventTierUp), TierKills)
	}
	s.Kills += TierKills - 1
	Update(s, Input{}, frameDT)
	if s.Tier != 2 {
		t.Errorf("tier %d after %d more kills, want still 2", s.Tier, TierKills-1)
	}
}

func TestTierUpByTime(t *testing.T) {
	s := survivalState(t)
	s.Spawning = false
	s.Time = TierTime - frameDT/2
	Update(s, Input{}, frameDT)
	if s.Tier != 2 {
		t.Errorf("tier %d after 4 minutes, want 2", s.Tier)
	}
}

// For a minute with an invulnerable, still player: every spawn is far
// enough and out of sight, and never more than MaxAlive are alive.
func TestSpawnRules(t *testing.T) {
	s := survivalState(t)
	spawned := 0
	for range 60 * 30 {
		before := len(s.Enemies)
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		if count(s, EventEnemySpawned) == 0 {
			continue
		}
		spawned++
		e := s.Enemies[len(s.Enemies)-1]
		if d := math.Hypot(e.X-s.Player.X, e.Y-s.Player.Y); d < SpawnMinDistance {
			t.Errorf("spawned %.1f cells away, want at least %.0f", d, SpawnMinDistance)
		}
		if world.LineOfSight(s.Map, s.Player.X, s.Player.Y, e.X, e.Y) {
			t.Errorf("spawned at (%.1f, %.1f), in the player's sight", e.X, e.Y)
		}
		if before+1 > TierFor(s.Tier).MaxAlive {
			t.Errorf("spawned with %d already alive, max %d", before, TierFor(s.Tier).MaxAlive)
		}
	}
	if spawned == 0 {
		t.Fatal("nothing spawned in a minute")
	}
	if len(s.Enemies) > TierFor(s.Tier).MaxAlive {
		t.Errorf("%d enemies alive, max %d", len(s.Enemies), TierFor(s.Tier).MaxAlive)
	}
}

func TestFirstSpawnDelay(t *testing.T) {
	s := survivalState(t)
	for i := 0; i < int(FirstSpawnDelay*30)-2; i++ {
		Update(s, Input{}, frameDT)
		if count(s, EventEnemySpawned) > 0 {
			t.Fatalf("spawned at %.2f s, before the %.0f s delay", s.Time, FirstSpawnDelay)
		}
	}
}

// Enemies spawned at a higher tier get more HP.
func TestSpawnScaledByTier(t *testing.T) {
	s := survivalState(t)
	s.Tier = 3 // HP x1.5
	for i := 0; i < 10*30 && len(s.Enemies) == 0; i++ {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
	}
	if len(s.Enemies) == 0 {
		t.Fatal("nothing spawned")
	}
	e := s.Enemies[0]
	want := ZombieHP * 3 / 2
	if e.Kind == KindShooter {
		want = ShooterHP * 3 / 2
	}
	if e.HP != want || e.damageMul != TierFor(3).DamageMul {
		t.Errorf("tier 3 spawn: HP %d, damage x%.2f; want %d and x%.2f", e.HP, e.damageMul, want, TierFor(3).DamageMul)
	}
}

// Over many spawns at a high tier, about half are shooters.
func TestShooterShare(t *testing.T) {
	s := survivalState(t)
	s.Tier = 10 // ShooterShare 0.5 (capped)
	shooters, total := 0, 0
	for range 3 * 60 * 30 {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		if count(s, EventEnemySpawned) > 0 {
			total++
			if s.Enemies[len(s.Enemies)-1].Kind == KindShooter {
				shooters++
			}
			s.Enemies = s.Enemies[:0] // clear so spawning continues
		}
	}
	if total < 30 {
		t.Fatalf("only %d spawns", total)
	}
	if share := float64(shooters) / float64(total); share < 0.3 || share > 0.7 {
		t.Errorf("shooter share %.2f over %d spawns, want about 0.5", share, total)
	}
}
