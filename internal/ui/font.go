package ui

import (
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// The block font (U1): big letters for the logo and the end screens, each
// FontRows tall and FontCols wide, made of full blocks. Letters are
// LetterGap columns apart and words WordGap, so they never touch (the
// crowded-text rule in TASKS.md "UI redesign").
const (
	FontRows  = 5
	FontCols  = 5
	LetterGap = 2
	WordGap   = 6
)

// bigLetters holds the letters the game's big words use (TERM FPS, YOU
// DIED, YOU WON); '#' is a filled cell.
var bigLetters = map[rune][FontRows]string{
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'E': {"#####", "#    ", "#### ", "#    ", "#####"},
	'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'F': {"#####", "#    ", "#### ", "#    ", "#    "},
	'P': {"#### ", "#   #", "#### ", "#    ", "#    "},
	'S': {" ####", "#    ", " ### ", "    #", "#### "},
	'Y': {"#   #", " # # ", "  #  ", "  #  ", "  #  "},
	'O': {" ### ", "#   #", "#   #", "#   #", " ### "},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'D': {"#### ", "#   #", "#   #", "#   #", "#### "},
	'I': {"#####", "  #  ", "  #  ", "  #  ", "#####"},
	'W': {"#   #", "#   #", "# # #", "## ##", "#   #"},
	'N': {"#   #", "##  #", "# # #", "#  ##", "#   #"},
}

// HasBigLetter reports whether the block font has r (a space counts).
func HasBigLetter(r rune) bool {
	_, ok := bigLetters[r]
	return ok || r == ' '
}

// BigWidth is how many columns text takes in the block font.
func BigWidth(text string) int {
	w, prevLetter := 0, false
	for _, r := range text {
		if r == ' ' {
			w += WordGap
			prevLetter = false
			continue
		}
		if prevLetter {
			w += LetterGap
		}
		w += FontCols
		prevLetter = true
	}
	return w
}

// BigText draws text in the block font with its top-left corner at (x, y),
// rows 0 to rows-1 only (so it can appear row by row). Only the filled
// cells are written: what's behind the letters stays. style gives each
// filled cell's style from its row and its column within the whole text.
func BigText(s term.Screen, x, y int, text string, rows int, style func(row, col int) term.Style) {
	col, prevLetter := 0, false
	for _, r := range text {
		if r == ' ' {
			col += WordGap
			prevLetter = false
			continue
		}
		if prevLetter {
			col += LetterGap
		}
		prevLetter = true
		g := bigLetters[r] // an unknown letter gives empty rows: nothing drawn
		for row := 0; row < min(rows, FontRows); row++ {
			for i, c := range g[row] {
				if c == '#' {
					s.Put(x+col+i, y+row, "█", style(row, col+i))
				}
			}
		}
		col += FontCols
	}
}

// The small logo: "TERM FPS" two rows tall, for the top of the Play and
// Settings screens.
var smallLogo = [2]string{
	"▀█▀ █▀▀ █▀█ █▀▄▀█   █▀▀ █▀█ █▀",
	" █  ██▄ █▀▄ █ ▀ █   █▀  █▀▀ ▄█",
}

// SmallLogoWidth is the small logo's width in columns.
var SmallLogoWidth = RuneLen(smallLogo[0])

// SmallLogo draws the small logo at (x, y) in amber on bg.
func SmallLogo(s term.Screen, x, y int, bg term.Color) {
	term.PutStr(s, x, y, smallLogo[0], term.Style{Fg: LogoRows[0], Bg: bg})
	term.PutStr(s, x, y+1, smallLogo[1], term.Style{Fg: LogoRows[3], Bg: bg})
}

// RuneLen is how many characters (screen cells) text takes.
func RuneLen(text string) int {
	return len([]rune(text))
}

// pad returns n spaces (none for n <= 0).
func pad(n int) string {
	return strings.Repeat(" ", max(n, 0))
}
