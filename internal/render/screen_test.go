package render

import (
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

func TestTooSmall(t *testing.T) {
	cases := []struct {
		cols, rows int
		want       bool
	}{
		{80, 24, false},
		{120, 40, false},
		{79, 24, true},
		{80, 23, true},
		{1, 1, true},
		{0, 0, true},
	}
	for _, c := range cases {
		if got := TooSmall(c.cols, c.rows); got != c.want {
			t.Errorf("TooSmall(%d, %d) = %v, want %v", c.cols, c.rows, got, c.want)
		}
	}
}

func TestDrawTooSmallShowsSize(t *testing.T) {
	s := term.NewFake(40, 10)
	DrawTooSmall(s)
	found := false
	for row := 0; row < 10; row++ {
		if strings.Contains(s.Row(row), "40x10, need 80x24") {
			found = true
		}
	}
	if !found {
		t.Error("message with the current and needed size not found")
	}
}

// Tiny screens must not panic; the message is just clipped.
func TestDrawTooSmallTinyScreens(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 2}, {12, 3}} {
		s := term.NewFake(size[0], size[1])
		DrawTooSmall(s)
	}
}
