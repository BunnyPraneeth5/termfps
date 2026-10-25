package term

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// newSim wraps tcell's in-memory screen in the adapter, so the tcell side
// of the adapter is tested without a terminal.
func newSim(t testing.TB, w, h int) (*Tcell, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(w, h)
	t.Cleanup(s.Fini)
	return &Tcell{s: s}, s
}

func TestRGB(t *testing.T) {
	r, g, b, ok := RGB(10, 20, 30).RGB()
	if !ok || r != 10 || g != 20 || b != 30 {
		t.Errorf("RGB(10, 20, 30).RGB() = %d %d %d %v", r, g, b, ok)
	}
	// Black RGB is not the default color, and basic colors aren't RGB.
	if RGB(0, 0, 0) == ColorDefault {
		t.Error("RGB(0, 0, 0) == ColorDefault")
	}
	if _, _, _, ok := ColorRed.RGB(); ok {
		t.Error("ColorRed counts as RGB")
	}
}

func TestTcellStyle(t *testing.T) {
	if got := tcellStyle(Style{}); got != tcell.StyleDefault {
		t.Errorf("zero Style = %v, want tcell.StyleDefault", got)
	}
	cases := []struct {
		st   Style
		want tcell.Style
	}{
		{Style{Fg: RGB(1, 2, 3), Bg: RGB(4, 5, 6)},
			tcell.StyleDefault.Foreground(tcell.NewRGBColor(1, 2, 3)).Background(tcell.NewRGBColor(4, 5, 6))},
		{Style{Fg: ColorRed}, tcell.StyleDefault.Foreground(tcell.ColorRed)},
		{Style{Fg: ColorGreen}, tcell.StyleDefault.Foreground(tcell.ColorGreen)},
		{Style{Fg: ColorYellow}, tcell.StyleDefault.Foreground(tcell.ColorYellow)},
		{Style{Bold: true}, tcell.StyleDefault.Bold(true)},
		{Style{Reverse: true, Dim: true}, tcell.StyleDefault.Reverse(true).Dim(true)},
	}
	for _, c := range cases {
		if got := tcellStyle(c.st); got != c.want {
			t.Errorf("tcellStyle(%+v) = %v, want %v", c.st, got, c.want)
		}
	}
}

// Put, Clear and Size go through to tcell's buffer.
func TestTcellScreen(t *testing.T) {
	scr, sim := newSim(t, 10, 4)
	if w, h := scr.Size(); w != 10 || h != 4 {
		t.Fatalf("Size = %dx%d, want 10x4", w, h)
	}
	// The simulation screen has 256 colors: no truecolor.
	if scr.Truecolor() {
		t.Error("Truecolor on the simulation screen")
	}
	scr.Put(2, 1, "▀", Style{Fg: RGB(9, 9, 9)})
	str, style, _ := sim.Get(2, 1)
	if str != "▀" || style != tcell.StyleDefault.Foreground(tcell.NewRGBColor(9, 9, 9)) {
		t.Errorf("cell (2, 1) = %q %v", str, style)
	}
	scr.Clear()
	if str, _, _ := sim.Get(2, 1); str != " " {
		t.Errorf("after Clear, cell (2, 1) = %q", str)
	}
}

func TestFromTcell(t *testing.T) {
	cases := []struct {
		name string
		ev   tcell.Event
		want Event
	}{
		// tcell drops Shift from upper-case letters ('W' alone says it).
		{"rune", tcell.NewEventKey(tcell.KeyRune, 'W', tcell.ModShift), Key{Code: KeyRune, Rune: 'W'}},
		{"shift+arrow", tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift), Key{Code: KeyUp, Mod: ModShift}},
		{"up", tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), Key{Code: KeyUp}},
		{"down", tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), Key{Code: KeyDown}},
		{"left", tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), Key{Code: KeyLeft}},
		{"right", tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), Key{Code: KeyRight}},
		{"enter", tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), Key{Code: KeyEnter}},
		{"esc", tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), Key{Code: KeyEscape}},
		{"ctrl+c", tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl), Key{Code: KeyCtrlC, Mod: ModCtrl}},
		{"meta is alt", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModMeta), Key{Code: KeyRune, Rune: 'x', Mod: ModAlt}},
		{"unused key", tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), Key{Code: KeyUnknown}},
		{"left click", tcell.NewEventMouse(3, 4, tcell.Button1, tcell.ModNone), Mouse{X: 3, Y: 4, Buttons: ButtonLeft}},
		{"right and middle", tcell.NewEventMouse(0, 0, tcell.Button2|tcell.Button3, tcell.ModNone), Mouse{Buttons: ButtonRight | ButtonMiddle}},
		{"wheel is no button", tcell.NewEventMouse(1, 1, tcell.WheelUp, tcell.ModNone), Mouse{X: 1, Y: 1}},
		{"resize", tcell.NewEventResize(100, 30), Resize{W: 100, H: 30}},
		{"focus", tcell.NewEventFocus(true), Focus{Focused: true}},
	}
	for _, c := range cases {
		got, ok := fromTcell(c.ev)
		if !ok {
			t.Errorf("%s: not converted", c.name)
			continue
		}
		// Every tcell v2 key is a Press; the times are the event's.
		switch g := got.(type) {
		case Key:
			if g.Action != Press || g.When.IsZero() {
				t.Errorf("%s: Action %d When %v", c.name, g.Action, g.When)
			}
			g.When = time.Time{}
			got = g
		case Mouse:
			if g.When.IsZero() {
				t.Errorf("%s: no When", c.name)
			}
			g.When = time.Time{}
			got = g
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
	if _, ok := fromTcell(tcell.NewEventInterrupt(nil)); ok {
		t.Error("interrupt event converted")
	}
}

// PollEvent skips events the game doesn't use and returns nil after Fini.
func TestTcellPollEvent(t *testing.T) {
	scr, sim := newSim(t, 10, 4)
	if err := sim.PostEvent(tcell.NewEventInterrupt(nil)); err != nil {
		t.Fatal(err)
	}
	if err := sim.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if k, ok := scr.PollEvent().(Key); !ok || k.Rune != 'w' {
		t.Errorf("PollEvent = %+v, want the w key", k)
	}
	scr.Fini()
	if ev := scr.PollEvent(); ev != nil {
		t.Errorf("PollEvent after Fini = %+v, want nil", ev)
	}
}

func TestFake(t *testing.T) {
	f := NewFake(6, 2)
	if f.Row(0) != "      " {
		t.Errorf("new row = %q, want blanks", f.Row(0))
	}
	st := Style{Bold: true}
	f.Put(1, 1, "x", st)
	f.Put(6, 0, "y", st) // outside: ignored
	if str, style := f.Get(1, 1); str != "x" || style != st {
		t.Errorf("Get(1, 1) = %q %+v", str, style)
	}
	if f.Text() != "      \n x    " {
		t.Errorf("Text = %q", f.Text())
	}
	f.SetSize(3, 3)
	if w, h := f.Size(); w != 3 || h != 3 || f.Row(2) != "   " {
		t.Errorf("after SetSize: %dx%d %q", w, h, f.Row(2))
	}

	f.Post(Resize{W: 1, H: 2})
	if ev := f.PollEvent(); ev != (Resize{W: 1, H: 2}) {
		t.Errorf("PollEvent = %+v", ev)
	}
	f.Fini()
	f.Fini() // twice is fine
	if ev := f.PollEvent(); ev != nil {
		t.Errorf("PollEvent after Fini = %+v, want nil", ev)
	}
}

// PutStr writes one character per cell and clips at both edges.
func TestPutStr(t *testing.T) {
	f := NewFake(5, 1)
	PutStr(f, -1, 0, "ab°cdefg", Style{})
	if f.Row(0) != "b°cde" {
		t.Errorf("row = %q, want %q", f.Row(0), "b°cde")
	}
}

// BenchmarkTcellPut120x40 fills tcell's buffer through the adapter with a
// different RGB style per cell, like the half-block presenter. With the
// render benchmarks (on the Fake) it splits the present time into our part
// and tcell's part.
func BenchmarkTcellPut120x40(b *testing.B) {
	scr, _ := newSim(b, 120, 40)
	b.ReportAllocs()
	for b.Loop() {
		for y := 0; y < 40; y++ {
			for x := 0; x < 120; x++ {
				v := uint8(x + y)
				scr.Put(x, y, "▀", Style{Fg: RGB(v, v, v), Bg: RGB(v/2, v/2, v/2)})
			}
		}
	}
}
