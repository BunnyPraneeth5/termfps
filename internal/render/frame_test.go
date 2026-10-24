package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// room has its east wall face at x=4 and open rows y=1..5.
const room = "#####\n" +
	"#...#\n" +
	"#...#\n" +
	"#P..#\n" +
	"#...#\n" +
	"#...#\n" +
	"#####\n"

func newState(t testing.TB, x, y float64) *game.State {
	t.Helper()
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := game.NewState(m)
	s.Player.X, s.Player.Y = x, y // facing east (Angle 0)
	return s
}

func newFrame(cols, rows int) *Frame {
	f := Frame{PixelAspect: DefaultPixelAspect}
	f.Resize(FrameSize(cols, rows))
	return &f
}

// Taller pixels (aspect 2) make walls half as many pixels tall.
// At 120x78, distance 2: 120 / (2*0.66*aspect*2) = 45.45 / aspect pixels.
func TestWallSpanPixelAspect(t *testing.T) {
	f := newFrame(120, 40)
	cases := []struct {
		aspect     float64
		start, end int
	}{
		{1.0, 16, 62}, // 45.45 tall around center 39
		{2.0, 28, 50}, // 22.7 tall
		{0.5, 0, 78},  // 90.9 tall, clamped to the frame
	}
	for _, c := range cases {
		f.PixelAspect = c.aspect
		if s, e := wallSpan(f, 2); s != c.start || e != c.end {
			t.Errorf("aspect %v: span = [%d, %d), want [%d, %d)", c.aspect, s, e, c.start, c.end)
		}
	}
}

func TestDrawFrameZeroAspectPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("DrawFrame with PixelAspect 0 didn't panic")
		}
	}()
	f := newFrame(80, 24)
	f.PixelAspect = 0
	DrawFrame(newState(t, 2.0, 3.5), f)
}

func TestFrameSize(t *testing.T) {
	if w, h := FrameSize(120, 40); w != 120 || h != 78 {
		t.Errorf("FrameSize(120, 40) = %d x %d, want 120 x 78", w, h)
	}
}

func TestResizeReusesMemory(t *testing.T) {
	f := newFrame(120, 40)
	first := &f.Pix[0]
	f.Resize(FrameSize(80, 24))
	if &f.Pix[0] != first {
		t.Error("resizing smaller allocated a new slice, want reuse")
	}
	if len(f.Pix) != 80*46 {
		t.Errorf("len(Pix) = %d, want %d", len(f.Pix), 80*46)
	}
}

// Facing a flat wall 2 cells away at 120x40 (Frame 120x78):
// lineHeight = 120 / (2 * 0.66 * 1.0 * 2) = 45.45, centered on 39,
// so the wall covers pixel rows 16..61 in every column.
func TestDrawFrameFacingWall(t *testing.T) {
	s := newState(t, 2.0, 3.5)
	f := newFrame(120, 40)
	DrawFrame(s, f)
	wallGray := wallShade(2, 0) // all columns hit the east (X-side) wall at 2

	for _, x := range []int{0, f.W / 2, f.W - 1} {
		for y := 0; y < f.H; y++ {
			want := backgroundShade(y, f.H)
			if y >= 16 && y < 62 {
				want = wallGray
			}
			if got := f.Pix[y*f.W+x].Gray; got != want {
				t.Errorf("pixel (%d, %d) = %d, want %d", x, y, got, want)
			}
		}
	}
}

// Touching the wall: PerpDist is near zero, lineHeight is enormous.
// It must not panic, and every pixel is wall.
func TestDrawFrameTouchingWall(t *testing.T) {
	s := newState(t, 3.99995, 3.5)
	f := newFrame(120, 40)
	DrawFrame(s, f)
	wallGray := wallShade(0.00005, 0)
	for i, p := range f.Pix {
		if p.Gray != wallGray {
			t.Fatalf("pixel (%d, %d) = %d, want wall %d", i%f.W, i/f.W, p.Gray, wallGray)
		}
	}
}

func TestDrawFrameEmptyFrame(t *testing.T) {
	s := newState(t, 2.0, 3.5)
	f := newFrame(80, 1) // 1 row = HUD only, Frame height 0
	DrawFrame(s, f)      // must not panic
	if len(f.Pix) != 0 {
		t.Errorf("len(Pix) = %d, want 0", len(f.Pix))
	}
}

func TestDim(t *testing.T) {
	var f Frame
	f.Resize(2, 1)
	f.Pix[0] = Pixel{Gray: 240, Tint: TintRed}
	f.Pix[1] = Pixel{Gray: 2}
	Dim(&f)
	if f.Pix[0] != (Pixel{Gray: 80, Tint: TintRed}) || f.Pix[1].Gray != 0 {
		t.Errorf("Dim: %+v, want gray 80 (tint kept) and 0", f.Pix)
	}
}
