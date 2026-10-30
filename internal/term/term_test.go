package term

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
)

// stubScreen stands in for a tcell screen in the adapter tests (tcell v3
// has no simulation screen). Cells go into tcell's own CellBuffer, the
// buffer a real tcell screen draws from.
type stubScreen struct {
	cells  tcell.CellBuffer
	colors int
	events chan tcell.Event
}

func newStub(w, h int) *stubScreen {
	s := &stubScreen{colors: 256, events: make(chan tcell.Event, 8)}
	s.cells.Resize(w, h)
	return s
}

func (s *stubScreen) Put(x, y int, str string, st tcell.Style) (string, int) {
	return s.cells.Put(x, y, str, st)
}
func (s *stubScreen) Size() (int, int)                { return s.cells.Size() }
func (s *stubScreen) Clear()                          { s.cells.Fill(' ', tcell.StyleDefault) }
func (s *stubScreen) Show()                           {}
func (s *stubScreen) Sync()                           {}
func (s *stubScreen) Colors() int                     { return s.colors }
func (s *stubScreen) EnableMouse(...tcell.MouseFlags) {}
func (s *stubScreen) EnableFocus()                    {}
func (s *stubScreen) EventQ() chan tcell.Event        { return s.events }
func (s *stubScreen) Fini()                           { close(s.events) }

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
		{Style{Fg: ColorBlue}, tcell.StyleDefault.Foreground(tcell.ColorBlue)},
		{Style{Bold: true}, tcell.StyleDefault.Bold(true)},
		{Style{Reverse: true, Dim: true}, tcell.StyleDefault.Reverse(true).Dim(true)},
	}
	for _, c := range cases {
		if got := tcellStyle(c.st); got != c.want {
			t.Errorf("tcellStyle(%+v) = %v, want %v", c.st, got, c.want)
		}
	}
}

// Put, Clear, Size and Truecolor go through to the tcell screen.
func TestTcellScreen(t *testing.T) {
	stub := newStub(10, 4)
	scr := &Tcell{s: stub}
	if w, h := scr.Size(); w != 10 || h != 4 {
		t.Fatalf("Size = %dx%d, want 10x4", w, h)
	}
	if scr.Truecolor() {
		t.Error("Truecolor with 256 colors")
	}
	stub.colors = 1 << 24
	if !scr.Truecolor() {
		t.Error("no Truecolor with 1<<24 colors")
	}
	scr.Put(2, 1, "▀", Style{Fg: RGB(9, 9, 9)})
	str, style, _ := stub.cells.Get(2, 1)
	if str != "▀" || style != tcell.StyleDefault.Foreground(tcell.NewRGBColor(9, 9, 9)) {
		t.Errorf("cell (2, 1) = %q %v", str, style)
	}
	scr.Clear()
	if str, _, _ := stub.cells.Get(2, 1); str != " " {
		t.Errorf("after Clear, cell (2, 1) = %q", str)
	}
}

func TestFromTcell(t *testing.T) {
	key := func(k tcell.Key, s string, m tcell.ModMask) *tcell.EventKey { return tcell.NewEventKey(k, s, m) }
	// keyEx is a key event as advanced keys report it.
	keyEx := func(k tcell.Key, s string, m tcell.ModMask, pressed bool, repeat int) *tcell.EventKey {
		return tcell.NewEventKeyEx(k, s, m, pressed, 0, repeat)
	}
	cases := []struct {
		name string
		ev   tcell.Event
		want Event
	}{
		{"rune", key(tcell.KeyRune, "w", tcell.ModNone), Key{Code: KeyRune, Rune: 'w'}},
		{"shift+arrow", key(tcell.KeyUp, "", tcell.ModShift), Key{Code: KeyUp, Mod: ModShift}},
		{"up", key(tcell.KeyUp, "", tcell.ModNone), Key{Code: KeyUp}},
		{"down", key(tcell.KeyDown, "", tcell.ModNone), Key{Code: KeyDown}},
		{"left", key(tcell.KeyLeft, "", tcell.ModNone), Key{Code: KeyLeft}},
		{"right", key(tcell.KeyRight, "", tcell.ModNone), Key{Code: KeyRight}},
		{"enter", key(tcell.KeyEnter, "", tcell.ModNone), Key{Code: KeyEnter}},
		{"esc", key(tcell.KeyEscape, "", tcell.ModNone), Key{Code: KeyEscape}},
		{"ctrl+c legacy", key(tcell.KeyCtrlC, "", tcell.ModCtrl), Key{Code: KeyCtrlC, Mod: ModCtrl}},
		{"meta is alt", key(tcell.KeyRune, "x", tcell.ModMeta), Key{Code: KeyRune, Rune: 'x', Mod: ModAlt}},
		{"unused key", key(tcell.KeyTab, "", tcell.ModNone), Key{Code: KeyUnknown}},
		{"composed character", key(tcell.KeyRune, "é", tcell.ModNone), Key{Code: KeyUnknown}},
		// Advanced keys: releases, kitty repeats, Ctrl+C as "c" with Ctrl.
		{"press", keyEx(tcell.KeyRune, "w", tcell.ModNone, true, 1), Key{Code: KeyRune, Rune: 'w'}},
		{"release", keyEx(tcell.KeyRune, "w", tcell.ModNone, false, 1), Key{Code: KeyRune, Rune: 'w', Action: Release}},
		{"repeat", keyEx(tcell.KeyLeft, "", tcell.ModNone, true, 2), Key{Code: KeyLeft, Action: Repeat}},
		{"arrow release", keyEx(tcell.KeyLeft, "", tcell.ModNone, false, 1), Key{Code: KeyLeft, Action: Release}},
		{"ctrl+c advanced", keyEx(tcell.KeyRune, "c", tcell.ModCtrl, true, 1), Key{Code: KeyCtrlC, Mod: ModCtrl}},
		{"ctrl+w is not ctrl+c", keyEx(tcell.KeyRune, "w", tcell.ModCtrl, true, 1), Key{Code: KeyRune, Rune: 'w', Mod: ModCtrl}},
		{"left click", tcell.NewEventMouse(3, 4, tcell.Button1, tcell.ModNone), Mouse{X: 3, Y: 4, Buttons: ButtonLeft}},
		{"right and middle", tcell.NewEventMouse(0, 0, tcell.Button2|tcell.Button3, tcell.ModNone), Mouse{Buttons: ButtonRight | ButtonMiddle}},
		{"wheel up", tcell.NewEventMouse(1, 1, tcell.WheelUp, tcell.ModNone), Mouse{X: 1, Y: 1, Buttons: ButtonWheelUp}},
		{"wheel down", tcell.NewEventMouse(1, 1, tcell.WheelDown, tcell.ModNone), Mouse{X: 1, Y: 1, Buttons: ButtonWheelDown}},
		{"resize", tcell.NewEventResize(100, 30), Resize{W: 100, H: 30}},
		{"focus", tcell.NewEventFocus(true), Focus{Focused: true}},
	}
	for _, c := range cases {
		got, ok := fromTcell(c.ev)
		if !ok {
			t.Errorf("%s: not converted", c.name)
			continue
		}
		// The times are the event's.
		switch g := got.(type) {
		case Key:
			if g.When.IsZero() {
				t.Errorf("%s: no When", c.name)
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

// PollEvent skips events the game doesn't use and returns nil once tcell
// closes its event channel (in Fini).
func TestTcellPollEvent(t *testing.T) {
	stub := newStub(10, 4)
	scr := &Tcell{s: stub}
	stub.events <- tcell.NewEventInterrupt(nil)
	stub.events <- tcell.NewEventKey(tcell.KeyRune, "w", tcell.ModNone)
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

// BenchmarkTcellPut120x40 fills tcell's cell buffer through the adapter
// with a different RGB style per cell, like the half-block presenter. With
// the render benchmarks (on the Fake) it splits the present time into our
// part and tcell's part.
func BenchmarkTcellPut120x40(b *testing.B) {
	scr := &Tcell{s: newStub(120, 40)}
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
