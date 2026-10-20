package render

import (
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func TestParseImage(t *testing.T) {
	img, err := ParseImage([]byte(".9\r\nrY\r\n\r\n"))
	if err != nil {
		t.Fatalf("ParseImage: %v", err)
	}
	if img.W != 2 || img.H != 2 {
		t.Fatalf("size = %dx%d, want 2x2", img.W, img.H)
	}
	if img.Opaque[0] {
		t.Error("'.' is opaque, want transparent")
	}
	if !img.Opaque[1] || img.Pix[1].Gray != 252 {
		t.Errorf("'9' = %+v, want opaque gray 252", img.Pix[1])
	}
	if img.Pix[2].Tint != TintRed || img.Pix[3].Tint != TintYellow || img.Pix[3].Gray <= img.Pix[2].Gray {
		t.Errorf("'r', 'Y' = %+v, %+v, want dim red and bright yellow", img.Pix[2], img.Pix[3])
	}
}

func TestParseImageErrors(t *testing.T) {
	cases := map[string]string{
		"":          "empty",
		"..\n...":   "row 2 has 3 pixels",
		"..\n.x.\n": "",
		".x":        "row 1, column 2: unknown character 'x'",
	}
	for input, want := range cases {
		_, err := ParseImage([]byte(input))
		if err == nil {
			t.Errorf("ParseImage(%q) succeeded, want an error", input)
			continue
		}
		if want != "" && !strings.Contains(err.Error(), want) {
			t.Errorf("ParseImage(%q) error = %q, want it to contain %q", input, err, want)
		}
	}
}

// Every built-in asset must parse, so a broken one fails here, not at
// game start.
func TestBuiltInAssetsParse(t *testing.T) {
	for name, data := range map[string][]byte{"gun": assets.Gun, "flash": assets.Flash} {
		if _, err := ParseImage(data); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Nearest-neighbor 2x scaling repeats each pixel; transparent pixels leave
// the Frame as it was; drawing past the edge is clipped.
func TestDrawImageScales(t *testing.T) {
	img, err := ParseImage([]byte("9.\n.9"))
	if err != nil {
		t.Fatal(err)
	}
	f := &Frame{PixelAspect: 1}
	f.Resize(6, 6)
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: 7}
	}
	drawImage(f, img, 1, 1, 4, 4)
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			want := uint8(7)
			if (x >= 1 && x <= 2 && y >= 1 && y <= 2) || (x >= 3 && x <= 4 && y >= 3 && y <= 4) {
				want = 252
			}
			if got := f.Pix[y*f.W+x].Gray; got != want {
				t.Errorf("pixel (%d, %d) = %d, want %d", x, y, got, want)
			}
		}
	}
	drawImage(f, img, 4, 4, 4, 4) // half outside: must not panic
}

func gunFrame(p game.Player) *Frame {
	f := newFrame(120, 40) // 120 x 78 pixels
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: 1}
	}
	DrawWeapon(f, p)
	return f
}

// The gun covers the bottom center and leaves the top of the view alone.
func TestDrawWeaponPlacement(t *testing.T) {
	f := gunFrame(game.Player{Weapon: game.NewWeapon()})
	if p := f.Pix[(f.H-1)*f.W+f.W/2]; p.Gray == 1 {
		t.Error("bottom center pixel untouched, want the gun")
	}
	for i := 0; i < (f.H/2)*f.W; i++ {
		if f.Pix[i].Gray != 1 {
			t.Fatalf("gun drew in the top half at (%d, %d)", i%f.W, i/f.W)
		}
	}
	for y := 0; y < f.H; y++ {
		if f.Pix[y*f.W].Gray != 1 || f.Pix[y*f.W+f.W-1].Gray != 1 {
			t.Fatalf("gun reaches the side edges at row %d", y)
		}
	}
}

// countTint counts pixels with a tint.
func countTint(f *Frame, tint Tint) int {
	n := 0
	for _, p := range f.Pix {
		if p.Tint == tint {
			n++
		}
	}
	return n
}

// The yellow flash shows only right after a shot.
func TestDrawWeaponFlash(t *testing.T) {
	m := mustOpenMap(t)
	s := game.NewState(m)
	if n := countTint(gunFrame(s.Player), TintYellow); n != 0 {
		t.Errorf("%d yellow pixels before shooting, want 0", n)
	}
	game.Update(s, game.Input{Fire: true}, 1.0/30)
	if n := countTint(gunFrame(s.Player), TintYellow); n == 0 {
		t.Error("no yellow flash right after a shot")
	}
	for range 4 { // 0.13 s later
		game.Update(s, game.Input{}, 1.0/30)
	}
	if n := countTint(gunFrame(s.Player), TintYellow); n != 0 {
		t.Errorf("%d yellow pixels 0.13 s after the shot, want 0", n)
	}
}

// topRow returns the first Frame row the gun covers.
func topRow(f *Frame) int {
	for i, p := range f.Pix {
		if p.Gray != 1 && p.Tint == TintNone {
			return i / f.W
		}
	}
	return -1
}

// The gun kicks down while flashing.
func TestDrawWeaponRecoil(t *testing.T) {
	m := mustOpenMap(t)
	s := game.NewState(m)
	rest := topRow(gunFrame(s.Player))
	game.Update(s, game.Input{Fire: true}, 1.0/30)
	if kicked := topRow(gunFrame(s.Player)); kicked <= rest {
		t.Errorf("gun top at row %d while flashing, want lower than %d", kicked, rest)
	}
}

// mustOpenMap is a small room for tests that run game.Update.
func mustOpenMap(t *testing.T) *world.Map {
	t.Helper()
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return m
}
