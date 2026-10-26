package render

import (
	"math"
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
	f := Frame{PixelAspect: DefaultPixelAspect, MixedBackground: DefaultMixedBackground}
	f.Resize(FrameSize(cols, rows))
	return &f
}

// Taller pixels (aspect 2) make walls half as many pixels tall.
// At 120x78, distance 2: 120 / (2*0.66*aspect*2) = 45.45 / aspect pixels.
func TestWallLinePixelAspect(t *testing.T) {
	f := newFrame(120, 40)
	cases := []struct {
		aspect, height float64
	}{
		{1.0, 45.45}, // around center 39
		{2.0, 22.73},
		{0.5, 90.91}, // taller than the frame: starts above it
	}
	for _, c := range cases {
		f.PixelAspect = c.aspect
		top, height := wallLine(f, 2)
		if math.Abs(height-c.height) > 0.01 || math.Abs(top-(39-c.height/2)) > 0.01 {
			t.Errorf("aspect %v: top %.2f height %.2f, want height %.2f around row 39", c.aspect, top, height, c.height)
		}
	}
}

func TestCoverage(t *testing.T) {
	cases := []struct {
		y           int
		top, bottom float64
		want        float64
	}{
		{5, 2, 10, 1},       // inside
		{1, 2, 10, 0},       // above
		{10, 2, 10, 0},      // below (bottom is exclusive)
		{2, 2.25, 10, 0.75}, // top edge
		{9, 2, 9.5, 0.5},    // bottom edge
		{3, 3.2, 3.6, 0.4},  // a wall thinner than a pixel
	}
	for _, c := range cases {
		if got := coverage(c.y, c.top, c.bottom); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("coverage(%d, %v, %v) = %v, want %v", c.y, c.top, c.bottom, got, c.want)
		}
	}
}

func TestBlendPixel(t *testing.T) {
	floor, wall := Pixel{Gray: 40}, Pixel{Gray: 200, Tint: TintRed}
	if got := blendPixel(floor, wall, 0.25); got != (Pixel{Gray: 80}) {
		t.Errorf("25%% wall = %+v, want gray 80, no tint", got)
	}
	if got := blendPixel(floor, wall, 0.75); got != (Pixel{Gray: 160, Tint: TintRed}) {
		t.Errorf("75%% wall = %+v, want gray 160, red", got)
	}
}

// The pixels at a wall's top and bottom edge are between the wall and the
// ceiling or floor; the ones inside are the full wall shade.
func TestDrawFrameSmoothEdges(t *testing.T) {
	f := newFrame(120, 40)
	DrawFrame(newState(t, 1.5, 3.5), f) // east wall 2.5 away, textures off
	x := f.W / 2
	top, height := wallLine(f, 2.5)
	wallGray := wallShade(2.5, 0)
	edge := int(top) // partly covered: top isn't a whole row
	if frac := top - math.Floor(top); frac == 0 {
		t.Fatalf("test needs a wall edge inside a pixel, top = %v", top)
	}
	got := f.Pix[edge*f.W+x].Gray
	if bg := backgroundShade(edge, f.H); got <= bg || got >= wallGray {
		t.Errorf("top edge pixel = %d, want between ceiling %d and wall %d", got, bg, wallGray)
	}
	if inside := f.Pix[(edge+1)*f.W+x].Gray; inside != wallGray {
		t.Errorf("pixel below the edge = %d, want the wall's %d", inside, wallGray)
	}
	bottom := int(top + height)
	if got, bg := f.Pix[bottom*f.W+x].Gray, backgroundShade(bottom, f.H); got <= bg || got >= wallGray {
		t.Errorf("bottom edge pixel = %d, want between floor %d and wall %d", got, bg, wallGray)
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

// The floor seen just below a wall's bottom edge is as far away as the
// wall, so floor and walls meet without a seam (V1).
func TestCastDistMatchesWalls(t *testing.T) {
	f := newFrame(120, 40)
	for _, dist := range []float64{1, 2.5, 6} {
		top, height := wallLine(f, dist)
		bottom := int(top + height) // the pixel row holding the wall's bottom edge
		d, ceiling := castDist(f, bottom)
		if ceiling || d < dist*0.9 || d > dist*1.1 {
			t.Errorf("wall at %v: floor row %d is at %v (ceiling %v), want about %v", dist, bottom, d, ceiling, dist)
		}
		if d, ceiling := castDist(f, int(top)); !ceiling || d < dist*0.9 || d > dist*1.1 {
			t.Errorf("wall at %v: ceiling row %d is at %v (ceiling %v), want about %v", dist, int(top), d, ceiling, dist)
		}
	}
}

// floorRows returns the pixels of rows from..to-1.
func floorRows(f *Frame, from, to int) []Pixel {
	return append([]Pixel(nil), f.Pix[from*f.W:to*f.W]...)
}

// The floor is fixed to the world: moving one whole cell shows the same
// floor (the texture repeats per cell), half a cell doesn't.
func TestFloorFollowsWorldPosition(t *testing.T) {
	f := newFrame(120, 40)
	f.Textures = true
	draw := func(x float64) []Pixel {
		DrawFrame(newState(t, x, 3.5), f)
		return floorRows(f, 70, 78) // floor in every case below (walls end higher)
	}
	a, b, c := draw(1.5), draw(2.5), draw(2.0)
	same, differs := true, false
	for i := range a {
		same = same && a[i] == b[i]
		differs = differs || a[i] != c[i]
	}
	if !same {
		t.Error("floor differs after moving exactly one cell, want the same")
	}
	if !differs {
		t.Error("floor is the same after moving half a cell, want it to move")
	}
}

// Floor and ceiling fade with distance, the ceiling stays darker than the
// floor, and with textures off both are the old flat gradient.
func TestFloorCeilingShading(t *testing.T) {
	f := newFrame(120, 40)
	f.Textures = true
	DrawFrame(newState(t, 1.5, 3.5), f)
	avg := func(y int) float64 {
		sum := 0
		for x := 0; x < f.W; x++ {
			sum += int(f.Pix[y*f.W+x].Gray)
		}
		return float64(sum) / float64(f.W)
	}
	if near, far := avg(f.H-1), avg(60); near <= far {
		t.Errorf("floor near %.0f, far %.0f: want near brighter", near, far)
	}
	if floor, ceiling := avg(f.H-1), avg(0); ceiling >= floor/2 {
		t.Errorf("ceiling %.0f vs floor %.0f at the same distance: want the ceiling much darker", ceiling, floor)
	}
	if near := avg(f.H - 1); near < 60 || near > 110 {
		t.Errorf("nearest floor row averages %.0f, want about the old 80", near)
	}

	f.Textures = false
	DrawFrame(newState(t, 1.5, 3.5), f)
	for _, y := range []int{0, f.H - 1} {
		if got, want := f.Pix[y*f.W+5].Gray, backgroundShade(y, f.H); got != want {
			t.Errorf("textures off, row %d = %d, want the flat %d", y, got, want)
		}
	}
}
