package render

import "github.com/BunnyPraneeth5/termfps/internal/game"

// pictureBackground is the gray behind a guide picture (M18a).
const pictureBackground = 0

// DrawPicture fills f with a dark background and img as large as fits,
// keeping its shape, centered: a still picture for the guide (M18a). A nil
// img leaves just the background.
func DrawPicture(f *Frame, img *Image) {
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: pictureBackground}
	}
	if img == nil || f.W <= 0 || f.H <= 0 {
		return
	}
	// The largest whole size that fits both ways: pixels are about square.
	h := f.H
	w := h * img.W / img.H
	if w > f.W {
		w = f.W
		h = max(w*img.H/img.W, 1)
	}
	drawImage(f, img, (f.W-w)/2, (f.H-h)/2, w, h)
}

// GunImage is the picture of gun k as held (M11), for the guide (M18a).
func GunImage(k game.GunKind) *Image {
	return gunImages[k]
}
