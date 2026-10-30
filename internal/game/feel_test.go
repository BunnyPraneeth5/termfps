package game

import (
	"math"
	"testing"
)

const corridor = "##############################\n#P...........................#\n##############################\n"

// Sprinting is SprintMul times faster, uses stamina, and stops when it's
// gone until StaminaResume is back.
func TestSprint(t *testing.T) {
	s := stateFrom(t, corridor)
	run(s, Input{Forward: true, Sprint: true}, 15, 0) // 0.5 s
	sprint := s.Player.X - 1.5
	if want := MoveSpeed * SprintMul * 0.5; math.Abs(sprint-want) > 0.01 {
		t.Errorf("0.5 s sprinting moved %.3f, want %.3f", sprint, want)
	}
	if want := 1 - StaminaDrain*0.5; math.Abs(s.Player.Stamina-want) > 0.01 {
		t.Errorf("stamina %.3f after 0.5 s sprinting, want %.3f", s.Player.Stamina, want)
	}
	for i := 0; i < 5*30 && !s.HasEvent(EventOutOfBreath); i++ {
		Update(s, Input{Forward: true, Sprint: true}, frameDT)
	}
	if s.Player.Stamina != 0 || s.Player.sprinting(Input{Forward: true, Sprint: true}) {
		t.Fatalf("out of breath: stamina %.2f, sprinting %v; want 0 and no sprinting", s.Player.Stamina, s.Player.sprinting(Input{Forward: true, Sprint: true}))
	}
	run(s, Input{}, int(StaminaResume/StaminaRegen*30)-3, 0)
	if s.Player.sprinting(Input{Forward: true, Sprint: true}) {
		t.Error("sprinting again before stamina came back to StaminaResume")
	}
	run(s, Input{}, 6, 0)
	if !s.Player.sprinting(Input{Forward: true, Sprint: true}) {
		t.Errorf("can't sprint with stamina %.2f, want to from %.2f", s.Player.Stamina, StaminaResume)
	}
}

// Holding F while standing still doesn't use stamina.
func TestSprintStandingStill(t *testing.T) {
	s := stateFrom(t, corridor)
	run(s, Input{Sprint: true}, 30, 0)
	if s.Player.Stamina != 1 {
		t.Errorf("stamina %.2f after holding F standing still, want 1", s.Player.Stamina)
	}
}

// A footstep every 0.75 cells walked.
func TestFootsteps(t *testing.T) {
	s := stateFrom(t, corridor)
	steps := run(s, Input{Forward: true}, 65, EventFootstep) // 6.5 cells
	if steps != 8 {
		t.Errorf("%d footsteps in 6.5 cells, want 8", steps)
	}
	if run(s, Input{TurnLeft: true}, 30, EventFootstep) != 0 {
		t.Error("footsteps while turning in place")
	}
}

// A hit from an enemy points the marker at it; it fades out.
func TestHitDirection(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Enemies = []Enemy{newZombie(s.Player.X, s.Player.Y+0.7)} // south of the player
	s.Player.Angle = 0
	run(s, Input{}, 15, 0)
	angle, strength, ok := s.Player.HitDirection()
	if !ok || math.Abs(angle-math.Pi/2) > 0.01 || strength <= 0 {
		t.Fatalf("hit direction %.2f (strength %.2f, ok %v), want pi/2 (south)", angle, strength, ok)
	}
	s.Enemies = nil
	run(s, Input{}, 25, 0) // HitDirTime is 24 frames
	if _, _, ok := s.Player.HitDirection(); ok {
		t.Error("hit marker still showing after HitDirTime")
	}
}

// The crosshair marker shows on a hit, marked as a kill when it killed.
func TestHitMarker(t *testing.T) {
	s := stateFrom(t, "########\n#P...Z.#\n########\n")
	Update(s, Input{Fire: true}, frameDT)
	if on, kill := s.Player.HitMarker(); !on || kill {
		t.Fatalf("after a hit: marker %v, kill %v; want true, false", on, kill)
	}
	run(s, Input{}, 5, 0) // HitMarkTime is 4.5 frames
	if on, _ := s.Player.HitMarker(); on {
		t.Fatal("hit marker still showing after HitMarkTime")
	}
	s.Enemies[0].HP = 1
	for i := 0; i < 15 && len(s.Enemies) > 0; i++ {
		Update(s, Input{Fire: true}, frameDT)
	}
	if on, kill := s.Player.HitMarker(); !on || !kill {
		t.Errorf("after a kill: marker %v, kill %v; want both true", on, kill)
	}
}

// Explosions and tank hits shake the screen; it settles again.
func TestShake(t *testing.T) {
	s := stateFrom(t, corridor)
	run(s, Input{Forward: true}, 10, 0)
	if x, y := s.Player.Shake(); x != 0 || y != 0 {
		t.Fatalf("shaking (%.2f, %.2f) with nothing going on", x, y)
	}
	addBlast(s, s.Player.X+4, s.Player.Y, 50, 0)
	Update(s, Input{}, frameDT)
	if x, y := s.Player.Shake(); x == 0 && y == 0 {
		t.Fatal("no shake from an explosion 4 cells away")
	}
	run(s, Input{}, 11, 0) // shakeTime is 10.5 frames
	if x, y := s.Player.Shake(); x != 0 || y != 0 {
		t.Errorf("still shaking (%.2f, %.2f) after %.2f s", x, y, shakeTime)
	}
	s.Enemies = []Enemy{newEnemy(KindTank, s.Player.X+0.7, s.Player.Y)}
	run(s, Input{}, 30, 0)
	if s.Player.shake == 0 && s.Player.shakePower == 0 {
		t.Error("a tank hit didn't shake the screen")
	}
}
