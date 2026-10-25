package term

import "time"

// Event is something from the terminal: Key, Mouse, Resize or Focus. The
// unexported method means only this package's types can be Events, so a
// type switch over these four covers every case.
type Event interface {
	event()
}

// Key is a key event.
type Key struct {
	Code   KeyCode
	Rune   rune // the character, when Code is KeyRune
	Mod    Mod
	Action Action
	When   time.Time
}

// KeyCode says which key it was. Letters, digits and Space are KeyRune
// (the character is in Key.Rune). Keys the game doesn't use are KeyUnknown.
type KeyCode uint8

const (
	KeyUnknown KeyCode = iota
	KeyRune
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEscape
	KeyCtrlC
)

// Mod is the modifier keys held with a key or click, as bits.
type Mod uint8

const (
	ModShift Mod = 1 << iota
	ModCtrl
	ModAlt
)

// Action says whether a key went down, repeats while held, or went up.
// tcell v2 can't tell a repeat from a press and never reports a release,
// so with it every key is Press. S2 (tcell v3) adds Repeat and Release.
type Action uint8

const (
	Press Action = iota
	Repeat
	Release
)

// Mouse is a mouse button event: Buttons holds the buttons down after it,
// so a release is an event without that button.
type Mouse struct {
	X, Y    int
	Buttons Buttons
	When    time.Time
}

// Buttons is a set of mouse buttons, as bits.
type Buttons uint8

const (
	ButtonLeft Buttons = 1 << iota
	ButtonRight
	ButtonMiddle
)

// Resize means the terminal changed size; W and H are the new size in cells.
type Resize struct {
	W, H int
}

// Focus means the terminal window got (Focused = true) or lost focus.
type Focus struct {
	Focused bool
}

func (Key) event()    {}
func (Mouse) event()  {}
func (Resize) event() {}
func (Focus) event()  {}
