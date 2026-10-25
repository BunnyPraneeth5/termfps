package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// A top pixel of 255 over a bottom pixel of 0 must give a white foreground
// and black background. Swapping them would flip every cell upside down.
func TestPresentHalfBlockTopIsForeground(t *testing.T) {
	s := term.NewFake(4, 3) // Frame 4 x 4 (2 view rows + HUD row)
	f := newFrame(4, 3)
	for x := 0; x < f.W; x++ {
		f.Pix[0*f.W+x].Gray = 255 // view row 0, top
		f.Pix[1*f.W+x].Gray = 0   // view row 0, bottom
		f.Pix[2*f.W+x].Gray = 10  // view row 1, top
		f.Pix[3*f.W+x].Gray = 200 // view row 1, bottom
	}
	PresentHalfBlock(s, f)

	cases := []struct {
		row    int
		fg, bg uint8
	}{
		{0, 255, 0},
		{1, 10, 200},
	}
	for _, c := range cases {
		str, style := s.Get(2, c.row)
		if str != string(halfBlock) {
			t.Errorf("row %d: cell = %q, want %q", c.row, str, string(halfBlock))
		}
		fg, bg := style.Fg, style.Bg
		if r, _, _, _ := fg.RGB(); r != c.fg {
			t.Errorf("row %d: foreground gray = %d, want %d", c.row, r, c.fg)
		}
		if r, _, _, _ := bg.RGB(); r != c.bg {
			t.Errorf("row %d: background gray = %d, want %d", c.row, r, c.bg)
		}
	}
}

// The HUD row (the last one) is not touched by the presenter.
func TestPresentHalfBlockLeavesHUDRow(t *testing.T) {
	s := term.NewFake(4, 3)
	f := newFrame(4, 3)
	PresentHalfBlock(s, f)
	if str, _ := s.Get(0, 2); str == string(halfBlock) {
		t.Error("presenter drew on the HUD row")
	}
}

func TestRampIndex(t *testing.T) {
	cases := []struct {
		g    uint8
		n    int
		want int
	}{
		{230, 5, 0}, // close wall: brightest block
		{255, 5, 0}, // above the scale: clamped
		{0, 5, 4},   // black: blank
		{8, 5, 4},   // ceiling: blank
		{50, 5, 3},  // far wall: still visible (░)
		{230, 10, 0},
		{0, 10, 9},
		{50, 10, 7}, // far wall in ASCII: ':' not blank
	}
	for _, c := range cases {
		if got := rampIndex(c.g, c.n); got != c.want {
			t.Errorf("rampIndex(%d, %d) = %d, want %d", c.g, c.n, got, c.want)
		}
	}
}

// Each text presenter averages the two pixels of a cell and uses the
// terminal's default style (no 24-bit colors).
func TestTextPresenters(t *testing.T) {
	cases := []struct {
		name    string
		present func(term.Screen, *Frame)
		bright  string // character for a close wall
		dark    string // character for black
	}{
		{"shade", PresentShade, "█", " "},
		{"ascii", PresentASCII, "@", " "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := term.NewFake(4, 3)
			f := newFrame(4, 3)
			for x := 0; x < f.W; x++ {
				f.Pix[0*f.W+x].Gray = 230 // row 0: both pixels bright
				f.Pix[1*f.W+x].Gray = 230
				f.Pix[2*f.W+x].Gray = 0 // row 1: both pixels black
				f.Pix[3*f.W+x].Gray = 0
			}
			f.Pix[0*f.W+3].Gray = 0 // cell (3, 0): bright over black averages to the middle
			c.present(s, f)

			if str, style := s.Get(1, 0); str != c.bright || style != (term.Style{}) {
				t.Errorf("bright cell = %q (style %v), want %q with default style", str, style, c.bright)
			}
			if str, _ := s.Get(1, 1); str != c.dark {
				t.Errorf("dark cell = %q, want %q", str, c.dark)
			}
			if str, _ := s.Get(3, 0); str == c.bright || str == c.dark {
				t.Errorf("mixed cell = %q, want a middle character", str)
			}
			if str, _ := s.Get(0, 2); str == c.bright {
				t.Error("presenter drew on the HUD row")
			}
		})
	}
}

// Benchmarks for the T6/T17 budget at 120x40. Run with:
// go test -bench . ./internal/render
// The presenter benchmarks run on the Fake screen, so they measure our code
// only; term's BenchmarkTcellPut120x40 measures filling tcell's buffer.
func BenchmarkDrawFrame120x40(b *testing.B) {
	s := newState(b, 2.0, 3.5)
	f := newFrame(120, 40)
	for b.Loop() {
		DrawFrame(s, f)
	}
}

func BenchmarkPresentHalfBlock120x40(b *testing.B) {
	scr := term.NewFake(120, 40)
	f := newFrame(120, 40)
	DrawFrame(newState(b, 2.0, 3.5), f)
	for b.Loop() {
		PresentHalfBlock(scr, f)
	}
}

// Mixed: the ASCII ramp character in the pixel color, on the same color
// at a fifth of the brightness; tints carry over.
func TestPresentMixed(t *testing.T) {
	s := term.NewFake(3, 2)
	f := newFrame(3, 2)
	f.Pix[0] = Pixel{Gray: 230}
	f.Pix[f.W] = Pixel{Gray: 230}
	f.Pix[1] = Pixel{Gray: 100, Tint: TintRed}
	f.Pix[f.W+1] = Pixel{Gray: 100}
	PresentMixed(s, f)

	str, style := s.Get(0, 0)
	fg, bg := style.Fg, style.Bg
	if str != "@" || fg != rgbColor(230, TintNone) || bg != rgbColor(46, TintNone) {
		t.Errorf("bright cell = %q fg %v bg %v; want @ in gray 230 on gray 46", str, fg, bg)
	}
	str, style = s.Get(1, 0)
	fg, bg = style.Fg, style.Bg
	if str != asciiRamp[rampIndex(100, len(asciiRamp))] || fg != rgbColor(100, TintRed) || bg != rgbColor(20, TintRed) {
		t.Errorf("red cell = %q fg %v bg %v; want the ramp character, red 100 on red 20", str, fg, bg)
	}
	if str, _ := s.Get(2, 0); str != " " {
		t.Errorf("black cell = %q, want a space", str)
	}
	if str, _ := s.Get(0, 1); str == "@" {
		t.Error("presenter drew on the HUD row")
	}
}

func BenchmarkPresentMixed120x40(b *testing.B) {
	scr := term.NewFake(120, 40)
	f := newFrame(120, 40)
	DrawFrame(newState(b, 2.0, 3.5), f)
	for b.Loop() {
		PresentMixed(scr, f)
	}
}
