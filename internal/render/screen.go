package render

import (
	"fmt"

	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// Smallest terminal the game draws in (ARCHITECTURE "Performance budget").
const (
	MinCols = 80
	MinRows = 24
)

// TooSmall reports whether a cols x rows terminal is below the minimum.
func TooSmall(cols, rows int) bool {
	return cols < MinCols || rows < MinRows
}

// The too-small screen's picture: the needed size as a dim box this big,
// with the window's size inside it to scale.
const (
	sizePicW = 20
	sizePicH = 6
)

// DrawTooSmall fills the screen and asks the player to make the window
// bigger (U3): the two sizes in numbers and as a small picture. Lines are
// centered and clipped to the screen, so any size (even 1x1) is safe; the
// picture is left out when there's no room for it.
func DrawTooSmall(s term.Screen) {
	w, h := s.Size()
	ui.Fill(s, 0, 0, w, h, ui.StBg)
	sy := ui.Sym()
	top := []string{
		"WINDOW TOO SMALL",
		fmt.Sprintf("now %d %s %d, needs %d %s %d", w, sy.Times, h, MinCols, sy.Times, MinRows),
	}
	bottom := []string{
		"Make the window bigger or the font smaller.",
		"The game waits here and carries on by itself.",
	}
	rows := len(top) + len(bottom)
	picture := h >= rows+sizePicH+2 && w >= sizePicW
	if picture {
		rows += sizePicH + 2
	}
	y := max((h-rows)/2, 0)
	center := func(line string, st term.Style) {
		ui.Center(s, 0, w, y, line, st) // PutStr clips at the edges
		y++
	}
	center(top[0], ui.Bold(ui.On(ui.ColAmber, ui.ColBg)))
	center(top[1], ui.StBg)
	if picture {
		y++
		x := (w - sizePicW) / 2
		ui.Box(s, x, y, sizePicW, sizePicH, ui.BoxSingle, ui.On(ui.ColBorder, ui.ColBg))
		// The window, to scale, in the needed size's top-left corner.
		pw := min(max(sizePicW*w/MinCols, 2), sizePicW)
		ph := min(max(sizePicH*h/MinRows, 2), sizePicH)
		ui.Box(s, x, y, pw, ph, ui.BoxSingle, ui.On(ui.ColAmber, ui.ColBg))
		y += sizePicH + 1
	}
	for _, l := range bottom {
		center(l, ui.On(ui.ColDim, ui.ColBg))
	}
}
