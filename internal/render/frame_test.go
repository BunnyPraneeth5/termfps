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
	var f Frame
	f.Resize(FrameSize(cols, rows))
	return &f
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

	for _, x := range []int{0, f.W / 2, f.W - 1} {
		for y := 0; y < f.H; y++ {
			want := ceilingGray
			if y >= 16 && y < 62 {
				want = wallGray
			} else if y >= 62 {
				want = floorGray
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
