// Package term is the game's own terminal layer (S1, D41): a screen
// interface, cell styles and input events that app, render, ui and input
// use instead of tcell. Only tcell.go talks to tcell, so moving to tcell v3
// (S2) or to our own input reader only changes that file. fake.go is an
// in-memory screen for tests.
package term

// Screen is a terminal the game draws on and reads events from. It's an
// interface: any type with these methods counts as a Screen, so the real
// terminal (Tcell) and the test screen (Fake) can be used the same way.
type Screen interface {
	// Size returns the screen size in cells.
	Size() (w, h int)
	// Put sets one cell to the text s (one character) in style st, in the
	// buffer. Cells outside the screen are ignored.
	Put(x, y int, s string, st Style)
	// Clear blanks every cell in the buffer.
	Clear()
	// Show sends the changed cells to the terminal.
	Show()
	// Sync redraws the whole terminal on the next Show (after a resize).
	Sync()
	// Truecolor reports whether the terminal shows 24-bit colors.
	Truecolor() bool
	// EnableMouse turns on mouse button events (clicks only, no motion, D28).
	EnableMouse()
	// EnableFocus turns on focus events, where the terminal supports them.
	EnableFocus()
	// PollEvent waits for the next event. It returns nil once the screen is
	// closed, so the input goroutine knows to stop.
	PollEvent() Event
}

// Style is how a cell looks. The zero value is the terminal's default
// colors with no attributes. Styles are plain values: compare them with ==.
type Style struct {
	Fg, Bg  Color
	Bold    bool
	Dim     bool
	Reverse bool // swap foreground and background
}

// Color is a cell color: the terminal's default, one of a few basic colors
// (which work without truecolor), or a 24-bit RGB color. The zero value is
// the default color.
type Color uint32

const (
	ColorDefault Color = iota
	ColorRed
	ColorGreen
	ColorYellow
)

// isRGB marks a Color that holds red, green and blue in its low 24 bits.
const isRGB Color = 1 << 24

// RGB returns a 24-bit color.
func RGB(r, g, b uint8) Color {
	return isRGB | Color(r)<<16 | Color(g)<<8 | Color(b)
}

// RGB returns the red, green and blue parts, and ok = false if c is not an
// RGB color.
func (c Color) RGB() (r, g, b uint8, ok bool) {
	if c&isRGB == 0 {
		return 0, 0, 0, false
	}
	return uint8(c >> 16), uint8(c >> 8), uint8(c), true
}

// PutStr writes text from cell (x, y) to the right, one character per cell,
// and cuts it off at the screen edges. The game's text has no wide or
// combining characters, so one character always fills one cell.
func PutStr(s Screen, x, y int, text string, st Style) {
	w, _ := s.Size()
	// range over a string steps through it one character (rune) at a time.
	for _, r := range text {
		if x >= w {
			return
		}
		if x >= 0 {
			s.Put(x, y, string(r), st)
		}
		x++
	}
}
