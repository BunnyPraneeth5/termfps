package render

import "github.com/BunnyPraneeth5/termfps/internal/term"

// Tint is a pixel's color on top of its gray brightness. The world is gray;
// color only marks important things (PRD Style, D23).
type Tint uint8

const (
	TintNone   Tint = iota // plain gray
	TintRed                // target blocks, shooter enemies
	TintGreen              // zombies (P1-06)
	TintYellow             // muzzle flash (P1-04)
	TintBlue               // armor (M10)
)

// rgbColor turns a gray brightness and a tint into a 24-bit color for the
// half-block presenter. A tinted pixel keeps its brightness (fog still
// works) and shifts toward its hue.
func rgbColor(g uint8, t Tint) term.Color {
	low := g / 4 // the channels a tint pulls down
	switch t {
	case TintRed:
		return term.RGB(g, low, low)
	case TintGreen:
		return term.RGB(low, g, low)
	case TintYellow:
		return term.RGB(g, g, low)
	case TintBlue:
		return term.RGB(low, g/2, g) // some green too, so dark blue stays visible
	}
	return term.RGB(g, g, g)
}

// basicColor is the plain terminal color for a tint, for the text
// presenters (shade, ASCII); the default color for no tint. Basic colors
// work without truecolor.
func basicColor(t Tint) term.Color {
	switch t {
	case TintRed:
		return term.ColorRed
	case TintGreen:
		return term.ColorGreen
	case TintYellow:
		return term.ColorYellow
	case TintBlue:
		return term.ColorBlue
	}
	return term.ColorDefault
}

// cellTint picks one tint for a text cell made of two pixels: a tinted
// pixel wins, so a thin red edge still shows.
func cellTint(top, bottom Pixel) Tint {
	if top.Tint != TintNone {
		return top.Tint
	}
	return bottom.Tint
}
