package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func TestRGBColor(t *testing.T) {
	cases := []struct {
		tint    Tint
		r, g, b uint8
	}{
		{TintNone, 200, 200, 200},
		{TintRed, 200, 50, 50},
		{TintGreen, 50, 200, 50},
		{TintYellow, 200, 200, 50},
		{TintBlue, 50, 100, 200},
	}
	for _, c := range cases {
		r, g, b, _ := rgbColor(200, c.tint).RGB()
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("rgbColor(200, %d) = (%d, %d, %d), want (%d, %d, %d)", c.tint, r, g, b, c.r, c.g, c.b)
		}
	}
}

// A target block 2 cells ahead fills the wall rows with red-tinted pixels;
// the ceiling and floor stay plain gray.
func TestDrawFrameTintsTargets(t *testing.T) {
	m, err := world.Parse([]byte("#######\n" +
		"#.....#\n" +
		"#.....#\n" +
		"#P...T#\n" +
		"#.....#\n" +
		"#.....#\n" +
		"#######\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := game.NewState(m)
	s.Player.X, s.Player.Y = 3.0, 3.5 // facing east, target face at x=5
	f := newFrame(120, 40)
	DrawFrame(s, f)
	x := f.W / 2
	if p := f.Pix[39*f.W+x]; p.Tint != TintRed {
		t.Errorf("center pixel tint = %d, want red", p.Tint)
	}
	if p := f.Pix[0*f.W+x]; p.Tint != TintNone {
		t.Errorf("ceiling pixel tint = %d, want none", p.Tint)
	}
	if p := f.Pix[(f.H-1)*f.W+x]; p.Tint != TintNone {
		t.Errorf("floor pixel tint = %d, want none", p.Tint)
	}
}

// Text presenters color tinted cells with the basic terminal color and
// leave plain cells in the default style.
func TestTextPresentersUseBasicColor(t *testing.T) {
	for _, present := range []func(term.Screen, *Frame){PresentShade, PresentASCII} {
		s := term.NewFake(4, 3)
		f := newFrame(4, 3)
		for i := range f.Pix {
			f.Pix[i] = Pixel{Gray: 200}
		}
		f.Pix[0*f.W+1].Tint = TintRed // top pixel of cell (1, 0)
		present(s, f)
		_, style := s.Get(1, 0)
		if fg := style.Fg; fg != term.ColorRed {
			t.Errorf("tinted cell foreground = %v, want red", fg)
		}
		if _, style := s.Get(2, 0); style != (term.Style{}) {
			t.Errorf("plain cell style = %v, want default", style)
		}
	}
}

func TestHalfBlockUsesTint(t *testing.T) {
	s := term.NewFake(4, 3)
	f := newFrame(4, 3)
	f.Pix[0*f.W+1] = Pixel{Gray: 200, Tint: TintRed}
	PresentHalfBlock(s, f)
	_, style := s.Get(1, 0)
	fg := style.Fg
	if r, g, _, _ := fg.RGB(); r != 200 || g != 50 {
		t.Errorf("tinted top pixel = (%d, %d, _), want (200, 50, _)", r, g)
	}
}
