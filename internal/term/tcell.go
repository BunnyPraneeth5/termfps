package term

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// Tcell is the real terminal, through tcell v2. This is the only file in
// the game that imports tcell.
type Tcell struct {
	s tcell.Screen
}

// Open creates and initializes the terminal screen. Call Fini to restore
// the terminal.
func Open() (*Tcell, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("create screen: %w", err)
	}
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("init screen: %w", err)
	}
	return &Tcell{s: s}, nil
}

// Fini restores the terminal. PollEvent returns nil after it.
func (t *Tcell) Fini() { t.s.Fini() }

func (t *Tcell) Size() (w, h int) { return t.s.Size() }

func (t *Tcell) Put(x, y int, s string, st Style) {
	t.s.Put(x, y, s, tcellStyle(st))
}

func (t *Tcell) Clear() { t.s.Clear() }
func (t *Tcell) Show()  { t.s.Show() }
func (t *Tcell) Sync()  { t.s.Sync() }

// Truecolor: tcell reports 1<<24 colors when it has truecolor, and fewer
// with TCELL_TRUECOLOR=disable or NO_COLOR.
func (t *Tcell) Truecolor() bool { return t.s.Colors() >= 1<<24 }

func (t *Tcell) EnableMouse() { t.s.EnableMouse(tcell.MouseButtonEvents) }
func (t *Tcell) EnableFocus() { t.s.EnableFocus() }

// PollEvent waits for the next tcell event the game uses and converts it.
// Other tcell events (paste, interrupts, errors) were never used by the
// game, so they're skipped.
func (t *Tcell) PollEvent() Event {
	for {
		ev := t.s.PollEvent()
		if ev == nil {
			return nil // the screen was finalized
		}
		if e, ok := fromTcell(ev); ok {
			return e
		}
	}
}

// fromTcell converts a tcell event; ok is false for event types the game
// doesn't use.
func fromTcell(ev tcell.Event) (e Event, ok bool) {
	// A type switch picks a case by the concrete type behind the
	// tcell.Event interface; ev has that type inside each case.
	switch ev := ev.(type) {
	case *tcell.EventKey:
		// tcell v2 has no key releases or repeat flag: every key is a Press.
		k := Key{Code: KeyUnknown, Mod: fromTcellMod(ev.Modifiers()), Action: Press, When: ev.When()}
		switch ev.Key() {
		case tcell.KeyRune:
			k.Code, k.Rune = KeyRune, ev.Rune()
		case tcell.KeyUp:
			k.Code = KeyUp
		case tcell.KeyDown:
			k.Code = KeyDown
		case tcell.KeyLeft:
			k.Code = KeyLeft
		case tcell.KeyRight:
			k.Code = KeyRight
		case tcell.KeyEnter:
			k.Code = KeyEnter
		case tcell.KeyEscape:
			k.Code = KeyEscape
		case tcell.KeyCtrlC:
			k.Code = KeyCtrlC
		}
		return k, true
	case *tcell.EventMouse:
		x, y := ev.Position()
		b := ev.Buttons()
		var bs Buttons
		if b&tcell.Button1 != 0 {
			bs |= ButtonLeft
		}
		if b&tcell.Button2 != 0 { // Button2 is the right button
			bs |= ButtonRight
		}
		if b&tcell.Button3 != 0 {
			bs |= ButtonMiddle
		}
		return Mouse{X: x, Y: y, Buttons: bs, When: ev.When()}, true
	case *tcell.EventResize:
		w, h := ev.Size()
		return Resize{W: w, H: h}, true
	case *tcell.EventFocus:
		return Focus{Focused: ev.Focused}, true
	}
	return nil, false
}

// fromTcellMod converts modifier bits. tcell's Meta counts as Alt.
func fromTcellMod(m tcell.ModMask) Mod {
	var out Mod
	if m&tcell.ModShift != 0 {
		out |= ModShift
	}
	if m&tcell.ModCtrl != 0 {
		out |= ModCtrl
	}
	if m&(tcell.ModAlt|tcell.ModMeta) != 0 {
		out |= ModAlt
	}
	return out
}

// tcellStyle converts a Style. The zero Style becomes tcell.StyleDefault.
func tcellStyle(st Style) tcell.Style {
	s := tcell.StyleDefault.Foreground(tcellColor(st.Fg)).Background(tcellColor(st.Bg))
	if st.Bold {
		s = s.Bold(true)
	}
	if st.Dim {
		s = s.Dim(true)
	}
	if st.Reverse {
		s = s.Reverse(true)
	}
	return s
}

func tcellColor(c Color) tcell.Color {
	if r, g, b, ok := c.RGB(); ok {
		return tcell.NewRGBColor(int32(r), int32(g), int32(b))
	}
	switch c {
	case ColorRed:
		return tcell.ColorRed
	case ColorGreen:
		return tcell.ColorGreen
	case ColorYellow:
		return tcell.ColorYellow
	}
	return tcell.ColorDefault
}
