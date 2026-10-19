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
