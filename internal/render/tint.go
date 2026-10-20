package render

import "github.com/gdamore/tcell/v2"

// Tint is a pixel's color on top of its gray brightness. The world is gray;
// color only marks important things (PRD Style, D23).
type Tint uint8

const (
	TintNone   Tint = iota // plain gray
	TintRed                // target blocks, shooter enemies
	TintGreen              // zombies (P1-06)
	TintYellow             // muzzle flash (P1-04)
)

// rgbColor turns a gray brightness and a tint into a 24-bit color for the
// half-block presenter. A tinted pixel keeps its brightness (fog still
// works) and shifts toward its hue.
func rgbColor(g uint8, t Tint) tcell.Color {
	v := int32(g)
	low := v / 4 // the channels a tint pulls down
	switch t {
	case TintRed:
		return tcell.NewRGBColor(v, low, low)
	case TintGreen:
		return tcell.NewRGBColor(low, v, low)
	case TintYellow:
		return tcell.NewRGBColor(v, v, low)
	}
	return tcell.NewRGBColor(v, v, v)
}

// basicColor is the plain terminal color for a tint, for the text
// presenters (shade, ASCII). Basic colors work without truecolor.
func basicColor(t Tint) (tcell.Color, bool) {
	switch t {
	case TintRed:
		return tcell.ColorRed, true
	case TintGreen:
		return tcell.ColorGreen, true
	case TintYellow:
		return tcell.ColorYellow, true
	}
	return tcell.ColorDefault, false
}

// cellTint picks one tint for a text cell made of two pixels: a tinted
// pixel wins, so a thin red edge still shows.
func cellTint(top, bottom Pixel) Tint {
	if top.Tint != TintNone {
		return top.Tint
	}
	return bottom.Tint
}
