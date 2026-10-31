package render

import "github.com/BunnyPraneeth5/termfps/internal/world"

// DrawMapPicture fills f with a top-down picture of m (M18b, the guide):
// one square of pixels per map cell, as large as fits, centered. Walls are
// light, floor dark, props mid gray, acid green, doors yellow, red blocks
// and red barrels red, the start white.
func DrawMapPicture(f *Frame, m *world.Map) {
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: pictureBackground}
	}
	if m == nil || m.W <= 0 || m.H <= 0 || f.W <= 0 || f.H <= 0 {
		return
	}
	// Pixels per cell, the same both ways; below 1 the map is shrunk
	// (nearest cell per pixel).
	scale := min(float64(f.W)/float64(m.W), float64(f.H)/float64(m.H))
	w, h := int(float64(m.W)*scale), int(float64(m.H)*scale)
	x0, y0 := (f.W-w)/2, (f.H-h)/2
	sx, sy := int(m.SpawnX), int(m.SpawnY)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cx, cy := int(float64(x)/scale), int(float64(y)/scale)
			px := mapCellPixel(m.At(cx, cy))
			if cx == sx && cy == sy {
				px = Pixel{Gray: 255}
			}
			f.Pix[(y0+y)*f.W+x0+x] = px
		}
	}
}

// mapCellPixel is how one map cell looks in the map picture.
func mapCellPixel(c world.Cell) Pixel {
	switch {
	case c == world.Target || c == world.ExplosiveBarrel:
		return Pixel{Gray: 200, Tint: TintRed}
	case c == world.Door:
		return Pixel{Gray: 160, Tint: TintYellow}
	case c == world.Acid:
		return Pixel{Gray: 150, Tint: TintGreen}
	case c.IsWall():
		return Pixel{Gray: 150}
	case c.BlocksMove():
		return Pixel{Gray: 90}
	}
	return Pixel{Gray: 35} // floor, and props you can walk over
}
