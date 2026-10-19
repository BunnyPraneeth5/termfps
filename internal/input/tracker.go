// Package input turns terminal key events into game.Input.
//
// tcell only reports key presses and OS key repeats, never key releases.
// The Tracker guesses "still held" from how recently a key's last event came
// in. See ARCHITECTURE.md "Input approach" and DECISIONS.md D12.
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
	// FirstWindow covers Windows' delay before key repeat starts (~500ms by
	// default). A quick tap moves for up to this long.
	FirstWindow = 550 * time.Millisecond
	// RepeatWindow is used once repeats are arriving (~30 per second by
	// default), so movement stops soon after the key is released.
	RepeatWindow = 100 * time.Millisecond
)

// Tracker remembers recent key events. The zero value is ready to use.
// It takes the current time as a parameter, so tests don't need to wait.
type Tracker struct {
	lastEvent [numKeys]time.Time
	repeating [numKeys]bool // a second event came in while the key was held
}

// Press records a key event (a press or an OS repeat) at time now.
func (t *Tracker) Press(k Key, now time.Time) {
	t.repeating[k] = t.Held(k, now)
	t.lastEvent[k] = now
}

// Held reports whether k counts as held at time now.
func (t *Tracker) Held(k Key, now time.Time) bool {
	if t.lastEvent[k].IsZero() {
		return false
	}
	window := FirstWindow
	if t.repeating[k] {
		window = RepeatWindow
	}
	return now.Sub(t.lastEvent[k]) < window
}

// Input returns the held keys at time now as a game.Input.
func (t *Tracker) Input(now time.Time) game.Input {
	return game.Input{
		Forward:     t.Held(KeyForward, now),
		Back:        t.Held(KeyBack, now),
		StrafeLeft:  t.Held(KeyStrafeLeft, now),
		StrafeRight: t.Held(KeyStrafeRight, now),
		TurnLeft:    t.Held(KeyTurnLeft, now),
		TurnRight:   t.Held(KeyTurnRight, now),
	}
}
