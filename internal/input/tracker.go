// Package input turns terminal key events into game.Input.
//
// tcell only reports key presses and OS key repeats, never key releases.
// The Tracker guesses "still held" from how recently a key's last event came
// in. See ARCHITECTURE.md "Input approach" and DECISIONS.md D12 and D31.
package input

import (
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// Key is a game key, independent of the physical key that produced it.
type Key int

const (
	KeyForward Key = iota // iota counts up from 0 for each constant in this block
	KeyBack
	KeyStrafeLeft
	KeyStrafeRight
	KeyTurnLeft
	KeyTurnRight
	numKeys // not a key: the number of keys above
)

const (
	// FirstWindow covers the OS delay before key repeat starts (~500ms by
	// default on Windows), until the Tracker has measured the real delay.
	FirstWindow = 550 * time.Millisecond
	// RepeatWindow is used once repeats are arriving (~30 per second by
	// default), so movement stops soon after the key is released.
	RepeatWindow = 100 * time.Millisecond

	// Once measured, the first window is the repeat delay plus delayMargin.
	delayMargin = 100 * time.Millisecond
	// A measured delay only counts between these, so a slow double tap
	// isn't taken for a key repeat.
	minRepeatDelay = 150 * time.Millisecond
	maxRepeatDelay = time.Second
	// Repeats come at least this often (a third event this soon after the
	// second confirms the second was a repeat).
	maxRepeatGap = 200 * time.Millisecond

	// While a key might still be a tap (no repeats yet), turning and moving
	// lose this fraction of their speed, so a tap turns and moves a little.
	// When repeats start, the slowdown fades out over slowFade.
	TapTurnSlow = 0.75
	TapMoveSlow = 0.6
	slowFade    = 150 * time.Millisecond
)

// Tracker remembers recent key events. The zero value is ready to use.
// It takes the current time as a parameter, so tests don't need to wait.
type Tracker struct {
	lastEvent [numKeys]time.Time
	repeating [numKeys]bool      // a second event came in while the key was held
	pressAt   [numKeys]time.Time // first event of the key's current press
	secondAt  [numKeys]time.Time // second event of it (zero until there is one)
	repeatAt  [numKeys]time.Time // when repeating became true

	// delay is the OS repeat delay measured from a held key; 0 until then.
	// It is one value for all keys: the OS uses the same delay for each.
	delay time.Duration
}

// Press records a key event (a press or an OS repeat) at time now.
func (t *Tracker) Press(k Key, now time.Time) {
	gap := now.Sub(t.lastEvent[k])
	switch {
	case t.lastEvent[k].IsZero() || gap > maxRepeatDelay:
		t.pressAt[k], t.secondAt[k] = now, time.Time{} // a new press
	case t.secondAt[k].IsZero():
		t.secondAt[k] = now
	case gap < maxRepeatGap && t.lastEvent[k].Equal(t.secondAt[k]):
		// The third event of a press came quickly, so the second was the
		// first repeat: the gap before it is the repeat delay.
		if d := t.secondAt[k].Sub(t.pressAt[k]); d >= minRepeatDelay && d <= maxRepeatDelay {
			t.delay = d
		}
	}
	held := t.Held(k, now)
	if held && !t.repeating[k] {
		t.repeatAt[k] = now
	}
	t.repeating[k] = held
	t.lastEvent[k] = now
}

// firstWindow is how long a key counts as held after its first event.
func (t *Tracker) firstWindow() time.Duration {
	if t.delay == 0 {
		return FirstWindow
	}
	return t.delay + delayMargin
}

// Held reports whether k counts as held at time now.
func (t *Tracker) Held(k Key, now time.Time) bool {
	if t.lastEvent[k].IsZero() {
		return false
	}
	window := t.firstWindow()
	if t.repeating[k] {
		window = RepeatWindow
	}
	return now.Sub(t.lastEvent[k]) < window
}

// tapness is 1 while a held key might still be a tap (no repeats yet),
// fades to 0 over slowFade once repeats arrive, and is 0 if not held.
func (t *Tracker) tapness(k Key, now time.Time) float64 {
	if !t.Held(k, now) {
		return 0
	}
	if !t.repeating[k] {
		return 1
	}
	return max(0, 1-float64(now.Sub(t.repeatAt[k]))/float64(slowFade))
}

// Input returns the held keys at time now as a game.Input, slowed while
// they might be taps.
func (t *Tracker) Input(now time.Time) game.Input {
	tap := func(keys ...Key) float64 { // the most tap-like of the keys
		m := 0.0
		for _, k := range keys {
			m = max(m, t.tapness(k, now))
		}
		return m
	}
	return game.Input{
		Forward:     t.Held(KeyForward, now),
		Back:        t.Held(KeyBack, now),
		StrafeLeft:  t.Held(KeyStrafeLeft, now),
		StrafeRight: t.Held(KeyStrafeRight, now),
		TurnLeft:    t.Held(KeyTurnLeft, now),
		TurnRight:   t.Held(KeyTurnRight, now),
		TurnSlow:    TapTurnSlow * tap(KeyTurnLeft, KeyTurnRight),
		MoveSlow:    TapMoveSlow * tap(KeyForward, KeyBack, KeyStrafeLeft, KeyStrafeRight),
	}
}
