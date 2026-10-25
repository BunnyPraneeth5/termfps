package game

// RenderMode picks how the 3D view is drawn (T19). It lives in game state so
// a V key press is handled by Update like any other input; render reads it.
type RenderMode int

const (
	ModeHalfBlock RenderMode = iota // two 24-bit gray pixels per cell (default)
	ModeShade                       // one shade block per cell: █▓▒░
	ModeASCII                       // one ASCII character per cell: @%#*+=-:.
	numModes                        // not a mode: how many modes there are
)

// Next returns the mode after m: half-block -> shade -> ASCII -> half-block.
func (m RenderMode) Next() RenderMode {
	return (m + 1) % numModes
}

// String names the mode for the HUD. A String method makes fmt print the
// name instead of the number (RenderMode satisfies the fmt.Stringer
// interface just by having this method).
func (m RenderMode) String() string {
	switch m {
	case ModeHalfBlock:
		return "half-block"
	case ModeShade:
		return "shade"
	case ModeASCII:
		return "ascii"
	}
	return "unknown"
}

// ParseRenderMode is the reverse of String, for modes read from the save
// file. ok is false for an unknown name.
func ParseRenderMode(name string) (m RenderMode, ok bool) {
	for m := RenderMode(0); m < numModes; m++ {
		if m.String() == name {
			return m, true
		}
	}
	return 0, false
}
