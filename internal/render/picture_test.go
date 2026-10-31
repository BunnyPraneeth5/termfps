package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// DrawPicture fits the image, keeps its shape and centers it; with no image
// only the background is left (M18a).
func TestDrawPicture(t *testing.T) {
	f := &Frame{PixelAspect: 1}
	f.Resize(40, 20)
	DrawPicture(f, ZombieImage) // 12x16: fits the height (20), so 15 wide
	minX, maxX, minY, maxY := f.W, -1, f.H, -1
	for i, p := range f.Pix {
		if p.Gray != pictureBackground || p.Tint != TintNone {
			x, y := i%f.W, i/f.W
			minX, maxX, minY, maxY = min(minX, x), max(maxX, x), min(minY, y), max(maxY, y)
		}
	}
	if maxY-minY+1 < f.H-2 || maxX-minX+1 > f.W/2 {
		t.Errorf("picture spans x %d-%d, y %d-%d; want nearly full height, kept narrow", minX, maxX, minY, maxY)
	}
	if mid := (minX + maxX) / 2; mid < f.W/2-2 || mid > f.W/2+2 {
		t.Errorf("picture centered at x %d, want about %d", mid, f.W/2)
	}
	DrawPicture(f, nil)
	for _, p := range f.Pix {
		if p.Gray != pictureBackground {
			t.Fatal("DrawPicture(nil) left something besides the background")
		}
	}
}

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
