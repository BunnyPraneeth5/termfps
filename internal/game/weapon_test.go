package game

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

const frameDT = 1.0 / 30

// run calls Update n times with the same input and counts one event.
func run(s *State, in Input, n int, count Event) int {
	c := 0
	for range n {
		Update(s, in, frameDT)
		for _, e := range s.Events {
			if e == count {
				c++
			}
		}
	}
	return c
}

func TestNewWeapon(t *testing.T) {
	s := newOpenState(t)
	if w := s.Player.Weapon; w.Mag != 25 || w.Reserve != 50 || w.Reloading() {
		t.Errorf("start weapon = %+v, want 25 in the gun, 50 spare, not reloading", w)
	}
}

// Holding fire for one second gives 3 shots (at 0, 1/3 and 2/3 s).
func TestFireRate(t *testing.T) {
	s := newOpenState(t)
	shots := run(s, Input{Fire: true}, 30, EventShotFired)
	if shots != 3 {
		t.Errorf("shots in 1 s = %d, want 3", shots)
	}
	if s.Player.Weapon.Mag != 22 {
		t.Errorf("magazine = %d after 3 shots, want 22", s.Player.Weapon.Mag)
	}
}

func TestReloadTiming(t *testing.T) {
	s := newOpenState(t)
	s.Player.Weapon.Mag = 10
	if run(s, Input{Reload: true}, 1, EventReloadStarted) != 1 {
		t.Fatal("R didn't start a reload")
	}
	// 1.4 s more: still reloading, and firing does nothing.
	if shots := run(s, Input{Fire: true}, 42, EventShotFired); shots != 0 {
		t.Errorf("fired %d times while reloading", shots)
	}
	if !s.Player.Weapon.Reloading() {
		t.Fatal("reload finished before 1.5 s")
	}
	if run(s, Input{}, 3, EventReloaded) != 1 {
		t.Fatal("reload didn't finish by 1.5 s")
	}
	if w := s.Player.Weapon; w.Mag != 25 || w.Reserve != 35 {
		t.Errorf("after reload: %d in the gun, %d spare; want 25 and 35", w.Mag, w.Reserve)
	}
}

// Emptying the magazine starts a reload by itself.
func TestAutoReload(t *testing.T) {
	s := newOpenState(t)
	s.Player.Weapon.Mag = 1
	Update(s, Input{Fire: true}, frameDT)
	if s.Player.Weapon.Mag != 0 || !s.Player.Weapon.Reloading() {
		t.Fatalf("after the last round: %+v, want empty and reloading", s.Player.Weapon)
	}
	run(s, Input{}, 46, 0) // 1.53 s
	if w := s.Player.Weapon; w.Mag != 25 || w.Reserve != 25 {
		t.Errorf("after auto-reload: %d in the gun, %d spare; want 25 and 25", w.Mag, w.Reserve)
	}
}

// With fewer spares than the magazine needs, reload takes what's there.
func TestPartialReload(t *testing.T) {
	s := newOpenState(t)
	s.Player.Weapon.Mag, s.Player.Weapon.Reserve = 0, 10
	run(s, Input{Reload: true}, 50, 0)
	if w := s.Player.Weapon; w.Mag != 10 || w.Reserve != 0 {
		t.Errorf("after reload: %d in the gun, %d spare; want 10 and 0", w.Mag, w.Reserve)
	}
}

func TestReloadNotNeeded(t *testing.T) {
	s := newOpenState(t) // full magazine
	if run(s, Input{Reload: true}, 1, EventReloadStarted) != 0 {
		t.Error("reload started with a full magazine")
	}
	s.Player.Weapon.Mag, s.Player.Weapon.Reserve = 5, 0
	if run(s, Input{Reload: true}, 1, EventReloadStarted) != 0 {
		t.Error("reload started with no spare rounds")
	}
}

// With no rounds anywhere, firing only clicks (once per fire interval).
func TestEmptyClick(t *testing.T) {
	s := newOpenState(t)
	s.Player.Weapon.Mag, s.Player.Weapon.Reserve = 0, 0
	if shots := run(s, Input{Fire: true}, 30, EventShotFired); shots != 0 {
		t.Errorf("fired %d shots with no ammo", shots)
	}
	s2 := newOpenState(t)
	s2.Player.Weapon.Mag, s2.Player.Weapon.Reserve = 0, 0
	if clicks := run(s2, Input{Fire: true}, 30, EventEmptyClick); clicks != 3 {
		t.Errorf("empty clicks in 1 s = %d, want 3", clicks)
	}
}

func shootingRange(t *testing.T, layout string) *State {
	t.Helper()
	m, err := world.Parse([]byte(layout))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := NewState(m) // spawn faces east
	s.Spawning = false
	return s
}

func TestShotDestroysTarget(t *testing.T) {
	s := shootingRange(t, "########\n"+
		"#P...T.#\n"+
		"########\n")
	Update(s, Input{Fire: true}, frameDT)
	if !s.HasEvent(EventTargetDestroyed) {
		t.Fatal("no EventTargetDestroyed")
	}
	if c := s.Map.At(5, 1); c != world.Empty {
		t.Errorf("cell (5, 1) = %q after the shot, want empty", c)
	}
}

// A wall in front protects a target behind it.
func TestWallBlocksShot(t *testing.T) {
	s := shootingRange(t, "########\n"+
		"#P.#.T.#\n"+
		"########\n")
	Update(s, Input{Fire: true}, frameDT)
	if s.HasEvent(EventTargetDestroyed) || s.Map.At(5, 1) != world.Target {
		t.Error("shot went through a wall and destroyed the target")
	}
}

// The muzzle flash lasts FlashTime after a shot.
func TestFlashTiming(t *testing.T) {
	s := newOpenState(t)
	Update(s, Input{Fire: true}, frameDT)
	if !s.Player.Weapon.Flashing() {
		t.Fatal("no flash right after a shot")
	}
	run(s, Input{}, 4, 0) // 0.13 s later (0.1 s plus float rounding is 4 frames)
	if s.Player.Weapon.Flashing() {
		t.Error("flash still on 0.13 s after the shot")
	}
}

// The bob phase moves only while walking.
func TestBobAdvancesWhileWalking(t *testing.T) {
	s := newOpenState(t)
	run(s, Input{TurnLeft: true}, 10, 0)
	if s.Player.Bob != 0 {
		t.Errorf("bob = %v after turning in place, want 0", s.Player.Bob)
	}
	run(s, Input{Forward: true}, 10, 0)
	if s.Player.Bob == 0 {
		t.Error("bob didn't move while walking")
	}
}
