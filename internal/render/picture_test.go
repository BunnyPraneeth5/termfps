package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// A Frame with an offset is presented at that screen cell.
func TestPresentOffset(t *testing.T) {
	s := term.NewFake(30, 12)
	f := &Frame{PixelAspect: 1, OffsetX: 5, OffsetY: 3}
	f.Resize(4, 4)
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: 250}
	}
	PresentASCII(s, f)
	if str, _ := s.Get(5, 3); str == " " || str == "" {
		t.Errorf("cell (5, 3) = %q, want the Frame's first cell", str)
	}
	if str, _ := s.Get(4, 3); str != " " && str != "" {
		t.Errorf("cell (4, 3) = %q, want untouched (left of the offset)", str)
	}
	if str, _ := s.Get(0, 0); str != " " && str != "" {
		t.Errorf("cell (0, 0) = %q, want untouched", str)
	}
}

// The map picture: walls light, floor dark, the start white, acid green.
func TestDrawMapPicture(t *testing.T) {
	m, err := world.Parse([]byte("######\n#P..~#\n######\n"))
	if err != nil {
		t.Fatal(err)
	}
	f := &Frame{PixelAspect: 1}
	f.Resize(12, 6) // 2 pixels per cell
	DrawMapPicture(f, m)
	at := func(cx, cy int) Pixel { return f.Pix[(2*cy)*f.W+2*cx] }
	if at(0, 0).Gray != 150 || at(2, 1).Gray != 35 || at(1, 1).Gray != 255 || at(4, 1).Tint != TintGreen {
		t.Errorf("wall %+v, floor %+v, start %+v, acid %+v", at(0, 0), at(2, 1), at(1, 1), at(4, 1))
	}
}
