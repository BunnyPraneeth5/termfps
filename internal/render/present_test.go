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

// With a color threshold, a cell reuses the style of the cell to its left
// when every channel is close enough, compared with the style actually
// used there, so a gradient can't drift more than the threshold. Each row
// starts fresh, and a tint never merges into gray.
func TestColorThresholdHalfBlock(t *testing.T) {
	grays := []uint8{100, 103, 106, 110, 140, 140}
	f := newFrame(len(grays), 3) // 2 view rows
	for row := 0; row < 4; row++ {
		for x, g := range grays {
			f.Pix[row*f.W+x] = Pixel{Gray: g}
		}
	}
	f.Pix[0*f.W+5] = Pixel{Gray: 141, Tint: TintRed} // top pixel of cell (5, 0)
	fgGray := func(s *term.Fake, x, y int) uint8 {
		_, st := s.Get(x, y)
		r, _, _, _ := st.Fg.RGB()
		return r
	}

	s := term.NewFake(len(grays), 3)
	PresentHalfBlock(s, f) // threshold 0: every cell its own color
	for x, g := range grays {
		if got := fgGray(s, x, 1); got != g {
			t.Errorf("threshold off: cell %d = %d, want %d", x, got, g)
		}
	}

	f.ColorThreshold = 4
	PresentHalfBlock(s, f)
	// 103 is within 4 of 100: reused. 106 is 6 from the 100 actually used:
	// new. 110 is within 4 of 106: reused. 140: new.
	for x, want := range []uint8{100, 100, 106, 106, 140, 140} {
		if got := fgGray(s, x, 1); got != want {
			t.Errorf("threshold 4: row 1 cell %d = %d, want %d", x, got, want)
		}
	}
	if _, st := s.Get(5, 0); st.Fg != rgbColor(141, TintRed) {
		t.Errorf("red cell next to gray 140 = %+v, want its own red", st.Fg)
	}
	if got := fgGray(s, 0, 1); got != 100 {
		t.Errorf("row 1 starts with %d, want its own 100", got)
	}
}

// Mixed uses the threshold too, and its background brightness comes from
// the Frame (0 = black, 0.5 = half).
func TestPresentMixedBackgroundAndThreshold(t *testing.T) {
	f := newFrame(2, 2)
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: 200}
	}
	f.Pix[1] = Pixel{Gray: 202}
	f.Pix[f.W+1] = Pixel{Gray: 202}
	s := term.NewFake(2, 2)

	f.MixedBackground = 0.5
	PresentMixed(s, f)
	if _, st := s.Get(0, 0); st.Bg != rgbColor(100, TintNone) {
		t.Errorf("background at 50%% = %+v, want gray 100", st.Bg)
	}
	f.MixedBackground = 0
	PresentMixed(s, f)
	if _, st := s.Get(0, 0); st.Bg != rgbColor(0, TintNone) {
		t.Errorf("background at 0%% = %+v, want black", st.Bg)
	}

	f.ColorThreshold = 4
	PresentMixed(s, f)
	_, left := s.Get(0, 0)
	if _, right := s.Get(1, 0); right != left {
		t.Errorf("gray 202 next to 200 with threshold 4: style %+v, want the left cell's %+v", right, left)
	}
}

func TestCloseStyle(t *testing.T) {
	a := term.Style{Fg: term.RGB(10, 20, 30), Bg: term.RGB(0, 0, 0)}
	cases := []struct {
		name      string
		b         term.Style
		threshold uint8
		want      bool
	}{
		{"same, threshold off", a, 0, false},
		{"same", a, 1, true},
		{"one channel at the threshold", term.Style{Fg: term.RGB(10, 25, 30), Bg: term.RGB(0, 0, 0)}, 5, true},
		{"one channel past it", term.Style{Fg: term.RGB(10, 26, 30), Bg: term.RGB(0, 0, 0)}, 5, false},
		{"background past it", term.Style{Fg: term.RGB(10, 20, 30), Bg: term.RGB(0, 0, 9)}, 8, false},
		{"default colors never close", term.Style{}, 255, false},
	}
	for _, c := range cases {
		if got := closeStyle(a, c.b, c.threshold); got != c.want {
			t.Errorf("%s: closeStyle = %v, want %v", c.name, got, c.want)
		}
	}
}

// The ASCII ramp (ASCII and mixed modes) has 16 distinct one-cell
// characters from "@" to blank, and the grays a frame has (0 to wallBase)
// reach every step (V5).
func TestASCIIRamp(t *testing.T) {
	if len(asciiRamp) != 16 || asciiRamp[0] != "@" || asciiRamp[15] != " " {
		t.Fatalf("ramp %q: want 16 steps from @ to blank", asciiRamp)
	}
	seen := map[string]bool{}
	for _, c := range asciiRamp {
		if len(c) != 1 || c[0] < ' ' || c[0] > '~' || seen[c] {
			t.Errorf("ramp step %q: want a distinct printable ASCII character", c)
		}
		seen[c] = true
	}
	used := map[int]bool{}
	for g := 0; g <= wallBase; g++ {
		used[rampIndex(uint8(g), len(asciiRamp))] = true
	}
	if len(used) != len(asciiRamp) {
		t.Errorf("grays 0-%d use %d of %d ramp steps", int(wallBase), len(used), len(asciiRamp))
	}
}

// At scale 2 each Frame cell fills a 2x2 block of screen cells, clipped
// to the view; the HUD row stays untouched (V6).
func TestPresentScaled(t *testing.T) {
	f := Frame{PixelAspect: DefaultPixelAspect, MixedBackground: DefaultMixedBackground, Scale: 2}
	f.Resize(ScaledFrameSize(5, 4, 2)) // 3 x 2 cells for a 5-column, 3-view-row screen
	if f.W != 3 || f.H != 4 {
		t.Fatalf("frame %dx%d, want 3x4", f.W, f.H)
	}
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: uint8(40 * (i%f.W + 1))} // each Frame column its own gray
	}
	s := term.NewFake(5, 4)
	PresentHalfBlock(s, &f)
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			_, st := s.Get(x, y)
			r, _, _, _ := st.Fg.RGB()
			if want := uint8(40 * (x/2 + 1)); r != want {
				t.Errorf("screen cell (%d, %d) = gray %d, want Frame column %d's %d", x, y, r, x/2, want)
			}
		}
	}
	if str, _ := s.Get(0, 3); str != " " {
		t.Errorf("HUD row = %q, want untouched", str)
	}
}

func TestAutoScaleAndFrameSize(t *testing.T) {
	for _, c := range []struct{ cols, want int }{{80, 1}, {160, 1}, {161, 2}, {320, 2}, {321, 3}} {
		if got := AutoScale(c.cols); got != c.want {
			t.Errorf("AutoScale(%d) = %d, want %d", c.cols, got, c.want)
		}
	}
	if w, h := ScaledFrameSize(120, 40, 1); w != 120 || h != 78 {
		t.Errorf("scale 1: %dx%d, want FrameSize's 120x78", w, h)
	}
	if w, h := ScaledFrameSize(321, 81, 2); w != 161 || h != 80 {
		t.Errorf("scale 2: %dx%d, want 161x80 (rounded up)", w, h)
	}
}
