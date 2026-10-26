package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// halfBlock is the upper half block. Its foreground color fills the top
// half of the cell and its background color fills the bottom half.
const halfBlock = "▀"

// PresentHalfBlock writes the Frame to the screen, two pixels per cell:
// top pixel = foreground, bottom pixel = background. It only fills the screen's
// buffer; the caller sends it to the terminal with Show.
func PresentHalfBlock(s term.Screen, f *Frame) {
	rows := f.H / 2
	for row := 0; row < rows; row++ {
		top := f.Pix[(2*row)*f.W : (2*row+1)*f.W]
		bottom := f.Pix[(2*row+1)*f.W : (2*row+2)*f.W]
		var prev term.Style // the style used for the cell to the left
		for x := 0; x < f.W; x++ {
			style := term.Style{
				Fg: rgbColor(top[x].Gray, top[x].Tint),
				Bg: rgbColor(bottom[x].Gray, bottom[x].Tint),
			}
			if x > 0 && closeStyle(style, prev, f.ColorThreshold) {
				style = prev
			}
			prev = style
			s.Put(x, row, halfBlock, style)
		}
	}
}

// closeStyle reports whether a's colors are each within threshold of b's,
// channel by channel (0 = off: never close). Only RGB colors can be close;
// both styles have no attributes.
//
// The presenters compare a cell with the style actually used to its left,
// not with that cell's own colors, so a long, slow gradient can't drift
// more than threshold from the truth: once it does, a new color is sent.
func closeStyle(a, b term.Style, threshold uint8) bool {
	return threshold > 0 && closeColor(a.Fg, b.Fg, threshold) && closeColor(a.Bg, b.Bg, threshold)
}

func closeColor(a, b term.Color, threshold uint8) bool {
	ar, ag, ab, aok := a.RGB()
	br, bg, bb, bok := b.RGB()
	return aok && bok && absDiff(ar, br) <= threshold && absDiff(ag, bg) <= threshold && absDiff(ab, bb) <= threshold
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

// Character ramps for the text presenters, brightest first. Strings (not
// runes) so Put can use them without allocating.
var (
	shadeRamp = []string{"█", "▓", "▒", "░", " "}
	asciiRamp = []string{"@", "%", "#", "*", "+", "=", "-", ":", ".", " "}
)

// PresentShade writes the Frame as one shade block per cell (stretch T19).
// It uses the terminal's default colors (basic colors for tinted cells), so
// it works without truecolor.
func PresentShade(s term.Screen, f *Frame) {
	presentRamp(s, f, shadeRamp)
}

// PresentASCII writes the Frame as one ASCII character per cell (stretch
// T19). Default colors and plain characters: the lightest mode for slow
// connections.
func PresentASCII(s term.Screen, f *Frame) {
	presentRamp(s, f, asciiRamp)
}

// presentRamp averages the two pixels of each cell and picks a character.
func presentRamp(s term.Screen, f *Frame, ramp []string) {
	rows := f.H / 2
	for row := 0; row < rows; row++ {
		top := f.Pix[(2*row)*f.W : (2*row+1)*f.W]
		bottom := f.Pix[(2*row+1)*f.W : (2*row+2)*f.W]
		for x := 0; x < f.W; x++ {
			avg := uint8((int(top[x].Gray) + int(bottom[x].Gray)) / 2)
			style := term.Style{Fg: basicColor(cellTint(top[x], bottom[x]))}
			s.Put(x, row, ramp[rampIndex(avg, len(ramp))], style)
		}
	}
}

// rampIndex maps a gray to a ramp position (0 = brightest character).
// The scale tops out at wallBase, the brightest gray DrawFrame makes, so
// the whole ramp is used and dim far walls don't vanish into blanks.
func rampIndex(g uint8, n int) int {
	level := int(math.Round(float64(g) / wallBase * float64(n-1)))
	level = min(max(level, 0), n-1)
	return n - 1 - level
}

// grayColor turns a brightness into a 24-bit gray.
func grayColor(g uint8) term.Color {
	return term.RGB(g, g, g)
}

// PresentMixed writes the Frame as one ASCII character per cell, like
// PresentASCII, but in truecolor: the character in the cell's color on a
// background of the same color at f.MixedBackground of the brightness
// (default a fifth), so dark gaps between characters fill in (P1-15, S3). The style is from S0ra's libasciic
// "Mixed" mode (D31). It needs truecolor, like half-block, and costs about
// the same: both send a foreground and a background color per cell.
func PresentMixed(s term.Screen, f *Frame) {
	rows := f.H / 2
	for row := 0; row < rows; row++ {
		top := f.Pix[(2*row)*f.W : (2*row+1)*f.W]
		bottom := f.Pix[(2*row+1)*f.W : (2*row+2)*f.W]
		var prev term.Style // the style used for the cell to the left
		for x := 0; x < f.W; x++ {
			avg := uint8((int(top[x].Gray) + int(bottom[x].Gray)) / 2)
			tint := cellTint(top[x], bottom[x])
			style := term.Style{
				Fg: rgbColor(avg, tint),
				Bg: rgbColor(toGray(float64(avg)*f.MixedBackground), tint),
			}
			if x > 0 && closeStyle(style, prev, f.ColorThreshold) {
				style = prev
			}
			prev = style
			s.Put(x, row, asciiRamp[rampIndex(avg, len(asciiRamp))], style)
		}
	}
}
