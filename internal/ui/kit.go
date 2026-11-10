package ui

import (
	"strconv"
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// The look kit (U1): the pieces every screen is built from. Each one only
// fills the screen's buffer; the caller sends it to the terminal with Show.

// BoxKind picks a box's border lines.
type BoxKind int

const (
	BoxSingle BoxKind = iota
	BoxDouble
	BoxRound
)

// Fill sets every cell of the w x h block at (x, y) to a space in style st.
func Fill(s term.Screen, x, y, w, h int, st term.Style) {
	if w <= 0 {
		return
	}
	row := pad(w)
	for j := 0; j < h; j++ {
		term.PutStr(s, x, y+j, row, st)
	}
}

// Put writes text at (x, y) and returns the column after it.
func Put(s term.Screen, x, y int, text string, st term.Style) int {
	term.PutStr(s, x, y, text, st)
	return x + RuneLen(text)
}

// Center writes text centered in the w columns from x.
func Center(s term.Screen, x, w, y int, text string, st term.Style) {
	term.PutStr(s, x+max((w-RuneLen(text))/2, 0), y, text, st)
}

// PutRight writes text so it ends just before column xEnd.
func PutRight(s term.Screen, xEnd, y int, text string, st term.Style) {
	term.PutStr(s, xEnd-RuneLen(text), y, text, st)
}

// Box draws a border w wide and h tall with its top-left corner at (x, y).
// The inside is left as it is.
func Box(s term.Screen, x, y, w, h int, kind BoxKind, st term.Style) {
	if w < 2 || h < 2 {
		return
	}
	b := sym.Boxes[kind]
	line := strings.Repeat(b[4], w-2)
	term.PutStr(s, x, y, b[0]+line+b[1], st)
	term.PutStr(s, x, y+h-1, b[2]+line+b[3], st)
	for j := 1; j < h-1; j++ {
		s.Put(x, y+j, b[5], st)
		s.Put(x+w-1, y+j, b[5], st)
	}
}

// Panel fills the block with fill's background and draws a border round it.
func Panel(s term.Screen, x, y, w, h int, kind BoxKind, border, fill term.Style) {
	Fill(s, x, y, w, h, fill)
	Box(s, x, y, w, h, kind, border)
}

// BoxTitle writes a title into a box's top border, one line in from its
// left corner: "┌─ TITLE ─┐".
func BoxTitle(s term.Screen, x, y int, title string, st term.Style) {
	term.PutStr(s, x+2, y, " "+title+" ", st)
}

// Bar draws a bar of cells cells at (x, y), the first frac (0-1) of them
// on, and returns the column after it. The simple symbol set puts it in
// brackets.
func Bar(s term.Screen, x, y, cells int, frac float64, on, off term.Style) int {
	return bar(s, x, y, cells, frac, sym.BarOn, sym.BarOff, on, off)
}

// LevelBar is Bar with the settings look (volume): ▮▮▮▮▯▯.
func LevelBar(s term.Screen, x, y, cells int, frac float64, on, off term.Style) int {
	return bar(s, x, y, cells, frac, sym.LevelOn, sym.LevelOff, on, off)
}

func bar(s term.Screen, x, y, cells int, frac float64, onCh, offCh string, on, off term.Style) int {
	n := BarCells(cells, frac)
	x = Put(s, x, y, sym.BarOpen, off)
	x = Put(s, x, y, strings.Repeat(onCh, n), on)
	x = Put(s, x, y, strings.Repeat(offCh, cells-n), off)
	return Put(s, x, y, sym.BarClose, off)
}

// BarCells is how many of cells are on for frac (0-1): rounded, but never
// empty while something is left and never full before it is.
func BarCells(cells int, frac float64) int {
	frac = min(max(frac, 0), 1)
	n := int(frac*float64(cells) + 0.5)
	switch {
	case frac > 0 && n == 0:
		n = 1
	case frac < 1 && n == cells:
		n = cells - 1
	}
	return min(max(n, 0), cells)
}

// BarWidth is how many columns a bar of cells cells takes.
func BarWidth(cells int) int {
	return cells + RuneLen(sym.BarOpen) + RuneLen(sym.BarClose)
}

// Hint is one entry of a key-cap footer: the keys and what they do.
type Hint struct {
	Keys, Text string
}

// hintsWidth is how wide Hints draws hs.
func hintsWidth(hs []Hint) int {
	w := 0
	for i, h := range hs {
		if i > 0 {
			w += 3
		}
		w += RuneLen(h.Keys) + 2 + 1 + RuneLen(h.Text)
	}
	return w
}

// Hints draws a key-cap footer centered in the w columns from x: each key
// as a chip, then what it does in dim text on bg.
func Hints(s term.Screen, x, w, y int, hs []Hint, bg term.Color) {
	cx := x + max((w-hintsWidth(hs))/2, 0)
	for i, h := range hs {
		if i > 0 {
			cx = Put(s, cx, y, "   ", On(ColDim, bg))
		}
		cx = Put(s, cx, y, " "+h.Keys+" ", StChip)
		cx = Put(s, cx, y, " "+h.Text, On(ColDim, bg))
	}
}

// Tabs draws a row of tab names from (x, y), the selected one as an amber
// bar, and returns where each tab starts and ends (for clicks).
func Tabs(s term.Screen, x, y int, names []string, selected int, bg term.Color) (spans [][2]int) {
	for i, n := range names {
		st := On(ColDim, bg)
		if i == selected {
			st = StSelected
		}
		end := Put(s, x, y, " "+n+" ", st)
		spans = append(spans, [2]int{x, end})
		x = end + 2
	}
	return spans
}

// Badge draws text as a small filled label at (x, y) and returns the
// column after it.
func Badge(s term.Screen, x, y int, text string, fg, bg term.Color) int {
	return Put(s, x, y, " "+text+" ", term.Style{Fg: fg, Bg: bg, Bold: true})
}

// Toast draws a one-line notice centered on row y of the screen: amber on
// the toast ground.
func Toast(s term.Screen, y int, text string) {
	w, _ := s.Size()
	Center(s, 0, w, y, "  "+text+"  ", term.Style{Fg: ColAmber, Bg: ColToast, Bold: true})
}

// Switch is an on/off switch's text: "[■ on ]" or "[  off]".
func Switch(on bool) string {
	if on {
		return "[" + sym.SwitchOn + " on ]"
	}
	return "[" + pad(RuneLen(sym.SwitchOn)) + " off]"
}

// Dots is page dots for page i (from 0) of n: "● ○ ○ 1/3".
func Dots(i, n int) string {
	var b strings.Builder
	if n <= 12 { // more than that doesn't fit a header; the numbers alone do
		for k := 0; k < n; k++ {
			if k == i {
				b.WriteString(sym.DotOn + " ")
			} else {
				b.WriteString(sym.DotOff + " ")
			}
		}
	}
	return b.String() + strconv.Itoa(i+1) + "/" + strconv.Itoa(n)
}
