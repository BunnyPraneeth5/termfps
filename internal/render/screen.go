package render

import (
	"fmt"

	"github.com/BunnyPraneeth5/termfps/internal/term"
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

// DrawTooSmall clears the screen and asks the user to resize. Lines are
// centered and clipped to the screen, so any size (even 1x1) is safe.
func DrawTooSmall(s term.Screen) {
	s.Clear()
	w, h := s.Size()
	lines := []string{
		"Terminal too small",
		fmt.Sprintf("%dx%d, need %dx%d", w, h, MinCols, MinRows),
		"Resize, or Ctrl+C to quit",
	}
	top := max((h-len(lines))/2, 0)
	for i, line := range lines {
		term.PutStr(s, max((w-len(line))/2, 0), top+i, line, term.Style{}) // PutStr clips at the edges
	}
}
