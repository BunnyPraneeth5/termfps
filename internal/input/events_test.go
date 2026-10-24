package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/ui"
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

func TestIsToggleMinimap(t *testing.T) {
	for _, r := range []rune{'m', 'M'} {
		if !IsToggleMinimap(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)) {
			t.Errorf("%q is not the minimap toggle", r)
		}
	}
	if IsToggleMinimap(tcell.NewEventKey(tcell.KeyRune, 'n', tcell.ModNone)) {
		t.Error("n counts as the minimap toggle")
	}
}

func TestIsCycleRenderMode(t *testing.T) {
	for _, r := range []rune{'v', 'V'} {
		if !IsCycleRenderMode(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)) {
			t.Errorf("%q is not the render mode key", r)
		}
	}
	if IsCycleRenderMode(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone)) {
		t.Error("m counts as the render mode key")
	}
}

func TestIsFireAndReload(t *testing.T) {
	if !IsFire(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)) {
		t.Error("Space is not fire")
	}
	for _, r := range []rune{'r', 'R'} {
		if !IsReload(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)) {
			t.Errorf("%q is not reload", r)
		}
	}
	if IsFire(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone)) || IsReload(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone)) {
		t.Error("wrong keys count as fire or reload")
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

func TestMenuKeyFor(t *testing.T) {
	cases := []struct {
		ev   *tcell.EventKey
		want ui.Key
	}{
		{tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), ui.KeyUp},
		{tcell.NewEventKey(tcell.KeyRune, 'W', tcell.ModNone), ui.KeyUp},
		{tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone), ui.KeyDown},
		{tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), ui.KeyLeft},
		{tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone), ui.KeyRight},
		{tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), ui.KeyEnter},
		{tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), ui.KeyBack},
	}
	for _, c := range cases {
		if got, ok := MenuKeyFor(c.ev); !ok || got != c.want {
			t.Errorf("%s: MenuKeyFor = %v, %v; want %v, true", c.ev.Name(), got, ok, c.want)
		}
	}
	if _, ok := MenuKeyFor(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)); ok {
		t.Error("Space counts as a menu key")
	}
}
