package input

import (
	"unicode"

	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// KeyFor maps a key event to a game key. ok is false for keys the game
// doesn't use for movement. Letters are compared lower-cased, so Shift and
// Caps Lock don't break them.
func KeyFor(ev term.Key) (k Key, ok bool) {
	switch ev.Code {
	case term.KeyUp:
		return KeyForward, true
	case term.KeyDown:
		return KeyBack, true
	case term.KeyLeft:
		return KeyTurnLeft, true
	case term.KeyRight:
		return KeyTurnRight, true
	case term.KeyRune:
		switch unicode.ToLower(ev.Rune) {
		case 'w':
			return KeyForward, true
		case 's':
			return KeyBack, true
		case 'a':
			return KeyStrafeLeft, true
		case 'd':
			return KeyStrafeRight, true
		case 'q':
			return KeyTurnLeft, true
		case 'e':
			return KeyTurnRight, true
		}
	}
	return 0, false
}

// IsToggleMinimap reports whether the key is M (either case).
func IsToggleMinimap(ev term.Key) bool {
	return ev.Code == term.KeyRune && unicode.ToLower(ev.Rune) == 'm'
}

// IsFire reports whether the key is Space (fire).
func IsFire(ev term.Key) bool {
	return ev.Code == term.KeyRune && ev.Rune == ' '
}

// IsReload reports whether the key is R (either case).
func IsReload(ev term.Key) bool {
	return ev.Code == term.KeyRune && unicode.ToLower(ev.Rune) == 'r'
}

// IsCycleRenderMode reports whether the key is V (either case).
func IsCycleRenderMode(ev term.Key) bool {
	return ev.Code == term.KeyRune && unicode.ToLower(ev.Rune) == 'v'
}

// IsQuit reports whether the key is Esc or Ctrl+C.
func IsQuit(ev term.Key) bool {
	return ev.Code == term.KeyEscape || ev.Code == term.KeyCtrlC
}

// Pump runs on its own goroutine. It reads screen events and sends them to
// out. It only sends; it never touches game state. It returns when the
// screen is finalized (PollEvent returns nil). panicOnKey is the
// TERMFPS_PANIC=input test switch.
func Pump(s term.Screen, out chan<- term.Event, panicOnKey bool) {
	for {
		ev := s.PollEvent()
		if ev == nil {
			return
		}
		if _, isKey := ev.(term.Key); isKey && panicOnKey {
			panic("forced test panic in input goroutine (TERMFPS_PANIC=input)")
		}
		out <- ev
	}
}

// MenuKeyFor maps a key event to a menu key: arrows or W/A/S/D,
// Enter, Esc. Space is left out: it's fire, and a held Space after dying
// would pick a menu item by accident.
func MenuKeyFor(ev term.Key) (k ui.Key, ok bool) {
	switch ev.Code {
	case term.KeyUp:
		return ui.KeyUp, true
	case term.KeyDown:
		return ui.KeyDown, true
	case term.KeyLeft:
		return ui.KeyLeft, true
	case term.KeyRight:
		return ui.KeyRight, true
	case term.KeyEnter:
		return ui.KeyEnter, true
	case term.KeyEscape:
		return ui.KeyBack, true
	case term.KeyRune:
		switch unicode.ToLower(ev.Rune) {
		case 'w':
			return ui.KeyUp, true
		case 's':
			return ui.KeyDown, true
		case 'a':
			return ui.KeyLeft, true
		case 'd':
			return ui.KeyRight, true
		}
	}
	return 0, false
}

// MouseButtons reports which of the game's mouse buttons are down in a
// mouse event: left (shoot, pick menu items) and right (reload). The
// terminal sends an event for each press and release; after a release, the
// released button is no longer in the event.
func MouseButtons(ev term.Mouse) (left, right bool) {
	return ev.Buttons&term.ButtonLeft != 0, ev.Buttons&term.ButtonRight != 0
}
