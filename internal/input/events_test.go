package input

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

func TestKeyFor(t *testing.T) {
	cases := []struct {
		name string
		ev   term.Key
		want Key
	}{
		{"w", term.Key{Code: term.KeyRune, Rune: 'w'}, KeyForward},
		{"W (shift/caps)", term.Key{Code: term.KeyRune, Rune: 'W'}, KeyForward},
		{"s", term.Key{Code: term.KeyRune, Rune: 's'}, KeyBack},
		{"a", term.Key{Code: term.KeyRune, Rune: 'a'}, KeyStrafeLeft},
		{"D", term.Key{Code: term.KeyRune, Rune: 'D'}, KeyStrafeRight},
		{"q", term.Key{Code: term.KeyRune, Rune: 'q'}, KeyTurnLeft},
		{"E", term.Key{Code: term.KeyRune, Rune: 'E'}, KeyTurnRight},
		{"Left", term.Key{Code: term.KeyLeft}, KeyTurnLeft},
		{"Right", term.Key{Code: term.KeyRight}, KeyTurnRight},
		{"Up", term.Key{Code: term.KeyUp}, KeyForward},
		{"Down", term.Key{Code: term.KeyDown}, KeyBack},
	}
	for _, c := range cases {
		got, ok := KeyFor(c.ev)
		if !ok || got != c.want {
			t.Errorf("%s: KeyFor = %v, %v; want %v, true", c.name, got, ok, c.want)
		}
	}
	if _, ok := KeyFor(term.Key{Code: term.KeyRune, Rune: 'x'}); ok {
		t.Error("x: KeyFor ok = true, want false")
	}
}

func TestIsToggleMinimap(t *testing.T) {
	for _, r := range []rune{'m', 'M'} {
		if !IsToggleMinimap(term.Key{Code: term.KeyRune, Rune: r}) {
			t.Errorf("%q is not the minimap toggle", r)
		}
	}
	if IsToggleMinimap(term.Key{Code: term.KeyRune, Rune: 'n'}) {
		t.Error("n counts as the minimap toggle")
	}
}

func TestIsCycleRenderMode(t *testing.T) {
	for _, r := range []rune{'v', 'V'} {
		if !IsCycleRenderMode(term.Key{Code: term.KeyRune, Rune: r}) {
			t.Errorf("%q is not the render mode key", r)
		}
	}
	if IsCycleRenderMode(term.Key{Code: term.KeyRune, Rune: 'm'}) {
		t.Error("m counts as the render mode key")
	}
}

func TestIsFireAndReload(t *testing.T) {
	if !IsFire(term.Key{Code: term.KeyRune, Rune: ' '}) {
		t.Error("Space is not fire")
	}
	for _, r := range []rune{'r', 'R'} {
		if !IsReload(term.Key{Code: term.KeyRune, Rune: r}) {
			t.Errorf("%q is not reload", r)
		}
	}
	if IsFire(term.Key{Code: term.KeyRune, Rune: 'f'}) || IsReload(term.Key{Code: term.KeyRune, Rune: 'e'}) {
		t.Error("wrong keys count as fire or reload")
	}
}

func TestIsQuit(t *testing.T) {
	if !IsQuit(term.Key{Code: term.KeyEscape}) {
		t.Error("Esc is not quit")
	}
	if !IsQuit(term.Key{Code: term.KeyCtrlC, Mod: term.ModCtrl}) {
		t.Error("Ctrl+C is not quit")
	}
	if IsQuit(term.Key{Code: term.KeyRune, Rune: 'q'}) {
		t.Error("q (turn left) counts as quit")
	}
}

func TestMenuKeyFor(t *testing.T) {
	cases := []struct {
		ev   term.Key
		want ui.Key
	}{
		{term.Key{Code: term.KeyUp}, ui.KeyUp},
		{term.Key{Code: term.KeyRune, Rune: 'W'}, ui.KeyUp},
		{term.Key{Code: term.KeyRune, Rune: 's'}, ui.KeyDown},
		{term.Key{Code: term.KeyLeft}, ui.KeyLeft},
		{term.Key{Code: term.KeyRune, Rune: 'd'}, ui.KeyRight},
		{term.Key{Code: term.KeyEnter}, ui.KeyEnter},
		{term.Key{Code: term.KeyEscape}, ui.KeyBack},
	}
	for _, c := range cases {
		if got, ok := MenuKeyFor(c.ev); !ok || got != c.want {
			t.Errorf("%+v: MenuKeyFor = %v, %v; want %v, true", c.ev, got, ok, c.want)
		}
	}
	if _, ok := MenuKeyFor(term.Key{Code: term.KeyRune, Rune: ' '}); ok {
		t.Error("Space counts as a menu key")
	}
}

func TestMouseButtons(t *testing.T) {
	cases := []struct {
		buttons     term.Buttons
		left, right bool
	}{
		{term.ButtonLeft, true, false},
		{term.ButtonRight, false, true},
		{term.ButtonLeft | term.ButtonRight, true, true},
		{0, false, false},                 // a release
		{term.ButtonMiddle, false, false}, // the middle button is unused
	}
	for _, c := range cases {
		l, r := MouseButtons(term.Mouse{X: 5, Y: 5, Buttons: c.buttons})
		if l != c.left || r != c.right {
			t.Errorf("buttons %v: left %v right %v, want %v %v", c.buttons, l, r, c.left, c.right)
		}
	}
}

// Pump passes events on and returns once the screen is closed.
func TestPump(t *testing.T) {
	s := term.NewFake(10, 4)
	s.Post(term.Key{Code: term.KeyRune, Rune: 'w'})
	s.Post(term.Resize{W: 20, H: 8})
	s.Fini()
	out := make(chan term.Event, 4)
	Pump(s, out, false) // returns, since the screen is closed
	if len(out) != 2 {
		t.Fatalf("Pump sent %d events, want 2", len(out))
	}
	if k, ok := (<-out).(term.Key); !ok || k.Rune != 'w' {
		t.Errorf("first event = %+v, want the w key", k)
	}
	if ev := <-out; ev != (term.Resize{W: 20, H: 8}) {
		t.Errorf("second event = %+v", ev)
	}
}
