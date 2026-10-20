package render

import "testing"

// The crosshair is a "+" of 9 pixels at the center, contrasting with what's
// under it; everything else is untouched.
func TestDrawCrosshair(t *testing.T) {
	f := newFrame(80, 24) // 80 x 46 pixels, center (40, 23)
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: 200}
	}
	f.Pix[23*f.W+42] = Pixel{Gray: 30} // a dark pixel under the right arm
	DrawCrosshair(f)

	want := map[[2]int]uint8{
		{40, 23}: 0, {38, 23}: 0, {39, 23}: 0, {41, 23}: 0,
		{42, 23}: 255, // was dark, so it turns white
		{40, 21}: 0, {40, 22}: 0, {40, 24}: 0, {40, 25}: 0,
	}
	changed := 0
	for i, p := range f.Pix {
		x, y := i%f.W, i/f.W
		if g, ok := want[[2]int{x, y}]; ok {
			if p.Gray != g {
				t.Errorf("crosshair pixel (%d, %d) = %d, want %d", x, y, p.Gray, g)
			}
			changed++
		} else if p.Gray != 200 {
			t.Errorf("pixel (%d, %d) = %d, want untouched 200", x, y, p.Gray)
		}
	}
	if changed != 9 {
		t.Errorf("crosshair has %d pixels, want 9", changed)
	}
}

func TestDrawCrosshairEmptyFrame(t *testing.T) {
	DrawCrosshair(&Frame{PixelAspect: 1}) // must not panic
}
