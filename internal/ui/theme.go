package ui

import "github.com/BunnyPraneeth5/termfps/internal/term"

// The palette of the "Arcade terminal" look (U1, D54). Every menu, the HUD
// and the guide take their colors from here, so the look changes in one
// place.
var (
	ColBg     = term.RGB(0x0b, 0x0d, 0x0e) // screen background
	ColPanel  = term.RGB(0x11, 0x14, 0x16) // panels, the HUD
	ColCard   = term.RGB(0x15, 0x18, 0x1a) // cards inside panels
	ColBorder = term.RGB(0x3a, 0x3f, 0x42) // borders, empty bar cells
	ColLine   = term.RGB(0x2a, 0x2f, 0x33) // lines inside panels
	ColText   = term.RGB(0xc9, 0xcc, 0xce) // normal text
	ColBright = term.RGB(0xf2, 0xf2, 0xf2) // bright text
	ColDim    = term.RGB(0x8a, 0x91, 0x96) // help lines, labels
	ColAmber  = term.RGB(0xff, 0xb0, 0x00) // the accent: selection, titles
	ColOrange = term.RGB(0xff, 0x8a, 0x3d)
	ColRed    = term.RGB(0xff, 0x5c, 0x5c) // health
	ColBlue   = term.RGB(0x6a, 0xa9, 0xff) // armor
	ColYellow = term.RGB(0xff, 0xe0, 0x66) // stamina, tagline, warnings
	ColGreen  = term.RGB(0x5e, 0xe2, 0x7a) // good news
	ColChipBg = term.RGB(0x2a, 0x2f, 0x33) // key caps
	ColChipFg = term.RGB(0xe8, 0xe8, 0xe8)
	ColToast  = term.RGB(0x1d, 0x21, 0x24) // the notice at the top of the game
)

// LogoRows are the logo's row colors, amber at the top to orange at the
// bottom.
var LogoRows = [FontRows]term.Color{
	term.RGB(0xff, 0xc5, 0x33), term.RGB(0xff, 0xb0, 0x00), term.RGB(0xff, 0x9a, 0x1f),
	term.RGB(0xff, 0x8a, 0x3d), term.RGB(0xe8, 0x70, 0x2e),
}

// Styles used all over. On is a helper for "this color on that ground".
var (
	StBg     = term.Style{Fg: ColText, Bg: ColBg}
	StPanel  = term.Style{Fg: ColText, Bg: ColPanel}
	StBorder = term.Style{Fg: ColBorder, Bg: ColPanel}
	StTitle  = term.Style{Fg: ColAmber, Bg: ColPanel, Bold: true}
	StDim    = term.Style{Fg: ColDim, Bg: ColPanel}
	StBright = term.Style{Fg: ColBright, Bg: ColPanel, Bold: true}
	// StSelected is the selection bar: dark text on amber.
	StSelected = term.Style{Fg: ColBg, Bg: ColAmber, Bold: true}
	StChip     = term.Style{Fg: ColChipFg, Bg: ColChipBg}
)

// On returns a style with foreground fg on background bg.
func On(fg, bg term.Color) term.Style {
	return term.Style{Fg: fg, Bg: bg}
}

// Bold returns st in bold.
func Bold(st term.Style) term.Style {
	st.Bold = true
	return st
}
