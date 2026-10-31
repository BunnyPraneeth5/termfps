package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

const longHall = "##############################\n#P...........................#\n##############################\n"

// From tier 2 a shot alerts enemies within HearingSteps; farther ones and
// tier 1 enemies don't hear it.
func TestHearing(t *testing.T) {
	for _, tier := range []int{1, 2} {
		s := stateFrom(t, longHall)
		s.Tier = tier
		s.Player.Angle = math.Pi // shoot west, into the wall
		s.Enemies = []Enemy{newZombie(11.5, 1.5), newZombie(25.5, 1.5)}
		Update(s, Input{Fire: true}, frameDT)
		near, far := s.Enemies[0].alert > 0, s.Enemies[1].alert > 0
		if want := tier >= HearTier; near != want || far {
			t.Errorf("tier %d: near zombie alerted %v (want %v), far one %v (want false)", tier, near, want, far)
		}
	}
}

// An alerted enemy rushes: RushMul times faster.
func TestRush(t *testing.T) {
	moved := func(alert bool) float64 {
		s := stateFrom(t, longHall)
		s.Enemies = []Enemy{newZombie(20.5, 1.5)}
		if alert {
			s.Enemies[0].alert = AlertTime
		}
		run(s, Input{}, 30, 0)
		return 20.5 - s.Enemies[0].X
	}
	if calm, rush := moved(false), moved(true); math.Abs(rush/calm-RushMul) > 0.05 {
		t.Errorf("1 s: calm %.2f cells, rushing %.2f; want %.1f times as far", calm, rush, RushMul)
	}
}

// coverRoom: the shooter stands right of the player, a pillar above it and
// to the right; behind the pillar the player can't see.
const coverRoom = "##########\n" +
	"#......i.#\n" +
	"#P....S..#\n" +
	"#........#\n" +
	"##########\n"

// From tier 3 a shooter that fired goes where the player can't see it.
func TestShooterTakesCover(t *testing.T) {
	for _, tier := range []int{2, 3} {
		s := stateFrom(t, coverRoom)
		s.Tier = tier
		for i := 0; i < 90 && !s.HasEvent(EventEnemyShot); i++ {
			Update(s, Input{}, frameDT)
			s.Player.HP = PlayerMaxHP
		}
		if !s.HasEvent(EventEnemyShot) {
			t.Fatalf("tier %d: the shooter never fired", tier)
		}
		for range 75 { // 2.5 s: enough to walk round the pillar
			Update(s, Input{}, frameDT)
			s.Player.HP = PlayerMaxHP
		}
		e := s.Enemies[0]
		hidden := !world.LineOfSight(s.Map, s.Player.X, s.Player.Y, e.X, e.Y)
		if want := tier >= CoverTier; hidden != want {
			t.Errorf("tier %d: 2.5 s after shooting the shooter is hidden %v at (%.1f, %.1f), want %v", tier, hidden, e.X, e.Y, want)
		}
	}
}

// From tier 4 a zombie with the "behind" role goes around the player and
// reaches them from behind, not straight on.
func TestFlankFromBehind(t *testing.T) {
	room := "#####################\n"
	for range 9 {
		room += "#...................#\n"
	}
	room += "#####################\n"
	m, err := world.Parse([]byte(room[:22*5+10] + "P" + room[22*5+11:]))
	if err != nil {
		t.Fatal(err)
	}
	for _, tier := range []int{3, 4} {
		s := NewState(m.Clone())
		s.Spawning = false
		s.Tier = tier // the player faces east; the zombie comes from the east
		z := newZombie(s.Player.X+8, s.Player.Y)
		z.role = 3
		s.Enemies = []Enemy{z}
		for i := 0; i < 20*30 && !s.Player.Hurting(); i++ {
			Update(s, Input{}, frameDT)
		}
		if !s.Player.Hurting() {
			t.Fatalf("tier %d: the zombie never reached the player", tier)
		}
		behind := s.Enemies[0].X < s.Player.X
		if want := tier >= FlankTier; behind != want {
			t.Errorf("tier %d: zombie hit from x %.1f (player at %.1f): from behind %v, want %v", tier, s.Enemies[0].X, s.Player.X, behind, want)
		}
	}
}

// Spawned zombies take the roles in turn: straight, left, right, behind.
func TestFlankRoles(t *testing.T) {
	s := survivalState(t)
	s.Tier = 1
	var roles []int
	for i := 0; i < 120*30 && len(roles) < 4; i++ {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		if count(s, EventEnemySpawned) > 0 {
			if e := s.Enemies[len(s.Enemies)-1]; e.Kind == KindZombie {
				roles = append(roles, e.role)
			}
			s.Enemies = s.Enemies[:0]
		}
	}
	for i, r := range roles {
		if r != i {
			t.Fatalf("zombie roles %v, want 0 1 2 3", roles)
		}
	}
}

// Anti-camping: after CampTime without a kill every enemy hunts (faster,
// and shooters stop hiding), until the next kill.
func TestHunting(t *testing.T) {
	s := stateFrom(t, longHall)
	s.Tier = 5
	hunts := run(s, Input{}, int(CampTime*30)+2, EventHuntStarted)
	if !s.Hunting() || hunts != 1 {
		t.Fatalf("after %.0f s without a kill: hunting %v, %d start events; want true, 1", CampTime, s.Hunting(), hunts)
	}
	e := newZombie(5.5, 1.5)
	if v := speedFor(s, &e, ZombieSpeed); math.Abs(v-ZombieSpeed*HuntMul) > 1e-9 {
		t.Errorf("hunting zombie speed %.2f, want %.2f", v, ZombieSpeed*HuntMul)
	}
	s.Enemies = []Enemy{e}
	s.Enemies[0].HP = 1
	Update(s, Input{Fire: true}, frameDT)
	if s.Kills != 1 || s.Hunting() {
		t.Errorf("after a kill: kills %d, hunting %v; want 1, false", s.Kills, s.Hunting())
	}
}
