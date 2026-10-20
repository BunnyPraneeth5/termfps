package render

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

// halfBlock is the upper half block. Its foreground color fills the top
// half of the cell and its background color fills the bottom half.
const halfBlock = "▀"

// PresentHalfBlock writes the Frame to the screen, two pixels per cell:
// top pixel = foreground, bottom pixel = background. It only fills tcell's
// buffer; the caller sends it to the terminal with Show.
func PresentHalfBlock(s tcell.Screen, f *Frame) {
	rows := f.H / 2
	for row := 0; row < rows; row++ {
		top := f.Pix[(2*row)*f.W : (2*row+1)*f.W]
		bottom := f.Pix[(2*row+1)*f.W : (2*row+2)*f.W]
		for x := 0; x < f.W; x++ {
			style := tcell.StyleDefault.
				Foreground(rgbColor(top[x].Gray, top[x].Tint)).
				Background(rgbColor(bottom[x].Gray, bottom[x].Tint))
			s.Put(x, row, halfBlock, style)
		}
	}
}

// Character ramps for the text presenters, brightest first. Strings (not
// runes) so tcell's Put can use them without allocating.
var (
	shadeRamp = []string{"█", "▓", "▒", "░", " "}
	asciiRamp = []string{"@", "%", "#", "*", "+", "=", "-", ":", ".", " "}
)

// PresentShade writes the Frame as one shade block per cell (stretch T19).
// It uses the terminal's default colors (basic colors for tinted cells), so
// it works without truecolor.
func PresentShade(s tcell.Screen, f *Frame) {
	presentRamp(s, f, shadeRamp)
}

// PresentASCII writes the Frame as one ASCII character per cell (stretch
// T19). Default colors and plain characters: the lightest mode for slow
// connections.
func PresentASCII(s tcell.Screen, f *Frame) {
	presentRamp(s, f, asciiRamp)
}

// presentRamp averages the two pixels of each cell and picks a character.
func presentRamp(s tcell.Screen, f *Frame, ramp []string) {
	rows := f.H / 2
	for row := 0; row < rows; row++ {
		top := f.Pix[(2*row)*f.W : (2*row+1)*f.W]
		bottom := f.Pix[(2*row+1)*f.W : (2*row+2)*f.W]
		for x := 0; x < f.W; x++ {
			avg := uint8((int(top[x].Gray) + int(bottom[x].Gray)) / 2)
			style := tcell.StyleDefault
			if c, ok := basicColor(cellTint(top[x], bottom[x])); ok {
				style = style.Foreground(c)
			}
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
func grayColor(g uint8) tcell.Color {
	v := int32(g)
	return tcell.NewRGBColor(v, v, v)
}
