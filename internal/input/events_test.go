package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestKeyFor(t *testing.T) {
	cases := []struct {
		name string
		ev   *tcell.EventKey
		want Key
	}{
		{"w", tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModNone), KeyForward},
		{"W (shift/caps)", tcell.NewEventKey(tcell.KeyRune, 'W', tcell.ModNone), KeyForward},
		{"s", tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone), KeyBack},
		{"a", tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone), KeyStrafeLeft},
		{"D", tcell.NewEventKey(tcell.KeyRune, 'D', tcell.ModNone), KeyStrafeRight},
		{"q", tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), KeyTurnLeft},
		{"E", tcell.NewEventKey(tcell.KeyRune, 'E', tcell.ModNone), KeyTurnRight},
		{"Left", tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), KeyTurnLeft},
		{"Right", tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), KeyTurnRight},
		{"Up", tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), KeyForward},
		{"Down", tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), KeyBack},
	}
	for _, c := range cases {
		got, ok := KeyFor(c.ev)
		if !ok || got != c.want {
			t.Errorf("%s: KeyFor = %v, %v; want %v, true", c.name, got, ok, c.want)
		}
	}
	if _, ok := KeyFor(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)); ok {
		t.Error("x: KeyFor ok = true, want false")
	}
}

func TestIsQuit(t *testing.T) {
	if !IsQuit(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)) {
		t.Error("Esc is not quit")
	}
	if !IsQuit(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)) {
		t.Error("Ctrl+C is not quit")
	}
	if IsQuit(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)) {
		t.Error("q (turn left) counts as quit")
	}
}
