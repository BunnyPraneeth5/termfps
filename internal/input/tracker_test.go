package input

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestNeverPressedNotHeld(t *testing.T) {
	var tr Tracker
	if tr.Held(KeyForward, t0) {
		t.Error("key never pressed counts as held")
	}
}

// After the first press, the key stays held through Windows' repeat delay.
func TestFirstPressCoversRepeatDelay(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	if !tr.Held(KeyForward, t0.Add(ms(300))) {
		t.Error("not held 300ms after the first press, want held (repeat delay)")
	}
	if tr.Held(KeyForward, t0.Add(FirstWindow)) {
		t.Error("still held after FirstWindow with no repeats, want released")
	}
}

// Once repeats arrive, release is detected quickly.
func TestRepeatsUseShortWindow(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	last := t0.Add(ms(500)) // repeat delay
	for i := 0; i < 10; i++ {
		tr.Press(KeyForward, last)
		last = last.Add(ms(33)) // ~30 repeats per second
	}
	last = last.Add(-ms(33)) // time of the final repeat
	if !tr.Held(KeyForward, last.Add(ms(50))) {
		t.Error("not held 50ms after a repeat, want held")
	}
	if tr.Held(KeyForward, last.Add(ms(150))) {
		t.Error("held 150ms after the last repeat, want released")
	}
}

// A new press long after release starts with the long window again.
func TestNewPressAfterReleaseUsesFirstWindow(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	tr.Press(KeyForward, t0.Add(ms(500)))
	again := t0.Add(5 * time.Second)
	tr.Press(KeyForward, again)
	if !tr.Held(KeyForward, again.Add(ms(300))) {
		t.Error("new press: not held after 300ms, want the long first window")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	tr.Press(KeyTurnLeft, t0)
	in := tr.Input(t0.Add(ms(10)))
	if !in.Forward || !in.TurnLeft || in.Back || in.TurnRight || in.StrafeLeft || in.StrafeRight {
		t.Errorf("Input = %+v, want only Forward and TurnLeft", in)
	}
}

// holdKey sends a press at start, the first repeat after delay, then
// repeats every 40ms (25 per second, a Linux default) until end.
func holdKey(tr *Tracker, k Key, start time.Time, delay time.Duration, end time.Time) time.Time {
	tr.Press(k, start)
	last := start
	for at := start.Add(delay); at.Before(end); at = at.Add(ms(40)) {
		tr.Press(k, at)
		last = at
	}
	return last
}

// A held key teaches the Tracker the OS repeat delay; later first presses
// then stay held through that delay, so a 660ms Linux delay doesn't stop
// movement just before the repeats start.
func TestLearnsRepeatDelay(t *testing.T) {
	var tr Tracker
	holdKey(&tr, KeyForward, t0, ms(660), t0.Add(ms(1500)))
	if tr.delay != ms(660) {
		t.Fatalf("learned delay = %v, want 660ms", tr.delay)
	}
	again := t0.Add(5 * time.Second)
	tr.Press(KeyTurnLeft, again) // a different key: the delay is the same for all
	if !tr.Held(KeyTurnLeft, again.Add(ms(700))) {
		t.Error("not held 700ms after a new press, want held (learned 660ms + margin)")
	}
	if tr.Held(KeyTurnLeft, again.Add(ms(800))) {
		t.Error("held 800ms after a press with no repeats, want released")
	}
}

// Two taps 400ms apart are not a key repeat: nothing is learned.
func TestDoubleTapLearnsNothing(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	tr.Press(KeyForward, t0.Add(ms(400)))
	tr.Press(KeyForward, t0.Add(ms(900))) // a third tap, not a quick repeat
	if tr.delay != 0 {
		t.Errorf("learned delay = %v from taps, want none", tr.delay)
	}
}

// While a key might be a tap it's slowed; once repeats arrive the slowdown
// fades to nothing. Async-style full speed is 0.
func TestTapSlowsThenFades(t *testing.T) {
	var tr Tracker
	tr.Press(KeyTurnRight, t0)
	in := tr.Input(t0.Add(ms(100)))
	if !in.TurnRight || in.TurnSlow != TapTurnSlow || in.MoveSlow != 0 {
		t.Errorf("during the first window: %+v, want TurnSlow %v, MoveSlow 0", in, TapTurnSlow)
	}
	last := holdKey(&tr, KeyTurnRight, t0, ms(500), t0.Add(ms(900)))
	if in := tr.Input(last.Add(ms(10))); in.TurnSlow != 0 {
		t.Errorf("400ms into repeats: TurnSlow = %v, want 0 (faded)", in.TurnSlow)
	}
}

// A tap turns about 25 degrees, not ~99 (S0ra's report, D31).
func TestTapTurnIsSmall(t *testing.T) {
	var tr Tracker
	tr.Press(KeyTurnRight, t0)
	angle := 0.0
	for now := t0; tr.Held(KeyTurnRight, now); now = now.Add(time.Second / 60) {
		in := tr.Input(now)
		angle += 180 * (1 - in.TurnSlow) / 60 // TurnSpeed is 180 degrees per second
	}
	if angle < 15 || angle > 40 {
		t.Errorf("one tap turned %.0f degrees, want about 25", angle)
	}
}

// After the first real release, keys are held exactly from press to
// release: a long hold with no repeats stays held, at full speed, and a
// tap stops at once.
func TestReleasesMakeHeldExact(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	if tr.Releases() {
		t.Fatal("Releases before any release")
	}
	tr.Release(KeyForward)
	if !tr.Releases() || tr.Held(KeyForward, t0.Add(ms(1))) {
		t.Fatal("released key still held, or Releases false")
	}

	tr.Press(KeyTurnLeft, t0.Add(time.Second))
	later := t0.Add(10 * time.Second) // no repeats in between
	if in := tr.Input(later); !in.TurnLeft || in.TurnSlow != 0 {
		t.Errorf("held turn 9 s later: %+v, want TurnLeft at full speed", in)
	}
	tr.Release(KeyTurnLeft)
	if tr.Held(KeyTurnLeft, later) {
		t.Error("turn still held after its release")
	}
}

// Before any release, presses alone keep the guessing (a terminal without
// releases); a key pressed before the first release stays down until its
// own release.
func TestFirstReleaseKeepsOtherKeysDown(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	tr.Press(KeyTurnRight, t0.Add(ms(10)))
	tr.Release(KeyTurnRight)
	if !tr.Held(KeyForward, t0.Add(5*time.Second)) {
		t.Error("W pressed before the first release isn't held, want held until released")
	}
}

// Losing focus lets go of every key.
func TestReleaseAll(t *testing.T) {
	var tr Tracker
	tr.Press(KeyForward, t0)
	tr.Release(KeyForward)
	tr.Press(KeyBack, t0.Add(ms(100)))
	tr.Press(KeyStrafeLeft, t0.Add(ms(100)))
	tr.ReleaseAll()
	if in := tr.Input(t0.Add(ms(200))); in.Back || in.StrafeLeft {
		t.Errorf("after ReleaseAll: %+v, want nothing held", in)
	}
}

// Reset (a new run) forgets keys but keeps the learned repeat delay and
// release support.
func TestResetKeepsLearned(t *testing.T) {
	var tr Tracker
	holdKey(&tr, KeyForward, t0, ms(300), t0.Add(ms(400)))
	if tr.delay != ms(300) {
		t.Fatalf("delay = %v, want 300ms learned", tr.delay)
	}
	tr.Press(KeyBack, t0.Add(time.Second))
	tr.Release(KeyForward)
	tr.Reset()
	if tr.delay != ms(300) || !tr.Releases() {
		t.Errorf("after Reset: delay %v, releases %v; want 300ms, true", tr.delay, tr.Releases())
	}
	if tr.Held(KeyBack, t0.Add(time.Second)) {
		t.Error("key still held after Reset")
	}
}
