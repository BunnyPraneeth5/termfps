package render

import "github.com/gdamore/tcell/v2"

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
				Foreground(grayColor(top[x].Gray)).
				Background(grayColor(bottom[x].Gray))
			s.Put(x, row, halfBlock, style)
		}
	}
}

// grayColor turns a brightness into a 24-bit gray.
func grayColor(g uint8) tcell.Color {
	v := int32(g)
	return tcell.NewRGBColor(v, v, v)
}
