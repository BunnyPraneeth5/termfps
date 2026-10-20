package render

import (
	"errors"
	"fmt"
	"strings"
)

// Image is a small pixel-art picture (gun, flash, enemies) with
// transparent pixels.
type Image struct {
	W, H   int
	Pix    []Pixel // row-major, index = y*W + x
	Opaque []bool  // false = transparent: the Frame shows through
}

// ParseImage reads an image from text, one character per pixel:
//
//	.        transparent
//	0-9      gray, from black (0) to bright (9)
//	r g y    dim red, green, yellow
//	R G Y    bright red, green, yellow
//
// Like maps, it accepts \r\n line endings and ignores trailing empty
// lines. All rows must be the same width.
func ParseImage(data []byte) (*Image, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil, errors.New("image is empty")
	}
	img := &Image{W: len(lines[0]), H: len(lines)}
	for y, line := range lines {
		if len(line) != img.W {
			return nil, fmt.Errorf("image row %d has %d pixels, expected %d", y+1, len(line), img.W)
		}
		for x := 0; x < len(line); x++ {
			p, opaque, err := imagePixel(line[x])
			if err != nil {
				return nil, fmt.Errorf("image row %d, column %d: %w", y+1, x+1, err)
			}
			img.Pix = append(img.Pix, p)
			img.Opaque = append(img.Opaque, opaque)
		}
	}
	return img, nil
}

// imagePixel decodes one character of an image file.
func imagePixel(c byte) (p Pixel, opaque bool, err error) {
	switch {
	case c == '.':
		return Pixel{}, false, nil
	case c >= '0' && c <= '9':
		return Pixel{Gray: (c - '0') * 28}, true, nil // 0..252
	}
	switch c {
	case 'r':
		return Pixel{Gray: 140, Tint: TintRed}, true, nil
	case 'R':
		return Pixel{Gray: 230, Tint: TintRed}, true, nil
	case 'g':
		return Pixel{Gray: 140, Tint: TintGreen}, true, nil
	case 'G':
		return Pixel{Gray: 230, Tint: TintGreen}, true, nil
	case 'y':
		return Pixel{Gray: 170, Tint: TintYellow}, true, nil
	case 'Y':
		return Pixel{Gray: 250, Tint: TintYellow}, true, nil
	}
	return Pixel{}, false, fmt.Errorf("unknown character %q (use . 0-9 r g y R G Y)", c)
}

// mustParseImage is for the built-in assets: a broken asset is a bug, so it
// panics at startup (before the terminal opens). Tests parse every asset.
func mustParseImage(name string, data []byte) *Image {
	img, err := ParseImage(data)
	if err != nil {
		panic(fmt.Sprintf("render: built-in image %s: %v", name, err))
	}
	return img
}

// drawImage draws img into f scaled to dw x dh pixels with its top-left
// corner at (x0, y0), using nearest-neighbor scaling. Transparent pixels
// and pixels outside the Frame are skipped.
func drawImage(f *Frame, img *Image, x0, y0, dw, dh int) {
	if dw <= 0 || dh <= 0 {
		return
	}
	for dy := 0; dy < dh; dy++ {
		y := y0 + dy
		if y < 0 || y >= f.H {
			continue
		}
		sy := dy * img.H / dh
		for dx := 0; dx < dw; dx++ {
			x := x0 + dx
			if x < 0 || x >= f.W {
				continue
			}
			i := sy*img.W + dx*img.W/dw
			if img.Opaque[i] {
				f.Pix[y*f.W+x] = img.Pix[i]
			}
		}
	}
}
