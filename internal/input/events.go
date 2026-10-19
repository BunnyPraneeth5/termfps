package input

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// KeyFor maps a tcell key event to a game key. ok is false for keys the game
// doesn't use for movement. Letters are compared lower-cased, so Shift and
// Caps Lock don't break them.
func KeyFor(ev *tcell.EventKey) (k Key, ok bool) {
	switch ev.Key() {
	case tcell.KeyUp:
		return KeyForward, true
	case tcell.KeyDown:
		return KeyBack, true
	case tcell.KeyLeft:
		return KeyTurnLeft, true
	case tcell.KeyRight:
		return KeyTurnRight, true
	case tcell.KeyRune:
		switch unicode.ToLower(ev.Rune()) {
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
func IsToggleMinimap(ev *tcell.EventKey) bool {
	return ev.Key() == tcell.KeyRune && unicode.ToLower(ev.Rune()) == 'm'
}

// IsQuit reports whether the key is Esc or Ctrl+C.
func IsQuit(ev *tcell.EventKey) bool {
	return ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC
}

// Pump runs on its own goroutine. It reads screen events and sends them to
// out. It only sends; it never touches game state. It returns when the
// screen is finalized (PollEvent returns nil). panicOnKey is the
// TERMFPS_PANIC=input test switch.
func Pump(s tcell.Screen, out chan<- tcell.Event, panicOnKey bool) {
	for {
		ev := s.PollEvent()
		if ev == nil {
			return
		}
		if _, isKey := ev.(*tcell.EventKey); isKey && panicOnKey {
			panic("forced test panic in input goroutine (TERMFPS_PANIC=input)")
		}
		out <- ev
	}
}
