package ui

// Symbols is one set of the special characters the screens draw with (U1).
// Fancy is the default; Simple uses only characters every console font has,
// for old console windows and fonts that show "?" or empty boxes instead.
type Symbols struct {
	Simple bool

	Heart, Armor, Stamina string // HUD bar labels
	BarOn, BarOff         string // HUD and guide bars
	BarOpen, BarClose     string // around a bar ("" in the fancy set)
	MagOn, MagOff         string // the magazine strip: a round, a used round
	LevelOn, LevelOff     string // settings bars (volume)
	Select                string // marks the selected item
	Prev, Next            string // around a setting's value
	SwitchOn              string // the mark in an on switch
	DotOn, DotOff         string // page dots
	Check, Star           string
	Sep                   string // between parts of a line: " · "
	Arrow                 string // starts a description line
	Locked                string
	Up, Down, Left, Right string // keys, as key caps
	UpDown, LeftRight     string
	Enter                 string
	Times                 string // in sizes: 80 × 24
	Then                  string // "this leads to that"
	// Box corners and lines, per BoxKind: top-left, top-right, bottom-left,
	// bottom-right, horizontal, vertical.
	Boxes [3][6]string
}

var fancy = Symbols{
	Heart: "♥", Armor: "◆", Stamina: "»",
	BarOn: "▰", BarOff: "▱",
	MagOn: "❙", MagOff: "❙",
	LevelOn: "▮", LevelOff: "▯",
	Select: "▶", Prev: "◀", Next: "▶", SwitchOn: "■",
	DotOn: "●", DotOff: "○", Check: "✓", Star: "✦",
	Sep: " · ", Arrow: "›", Locked: "▒",
	Up: "↑", Down: "↓", Left: "←", Right: "→", UpDown: "↑↓", LeftRight: "←→", Enter: "⏎",
	Times: "×", Then: "→",
	Boxes: [3][6]string{
		BoxSingle: {"┌", "┐", "└", "┘", "─", "│"},
		BoxDouble: {"╔", "╗", "╚", "╝", "═", "║"},
		BoxRound:  {"╭", "╮", "╰", "╯", "─", "│"},
	},
}

var simple = Symbols{
	Simple: true,
	Heart:  "HP", Armor: "AR", Stamina: "ST",
	BarOn: "#", BarOff: "-", BarOpen: "[", BarClose: "]",
	MagOn: "|", MagOff: ".",
	LevelOn: "#", LevelOff: "-",
	Select: ">", Prev: "<", Next: ">", SwitchOn: "x",
	DotOn: "*", DotOff: ".", Check: "OK", Star: "*",
	Sep: " - ", Arrow: ">", Locked: "x",
	Up: "Up", Down: "Down", Left: "Left", Right: "Right", UpDown: "Up/Down", LeftRight: "Left/Right", Enter: "Enter",
	Times: "x", Then: "->",
	Boxes: [3][6]string{
		{"+", "+", "+", "+", "-", "|"},
		{"+", "+", "+", "+", "-", "|"},
		{"+", "+", "+", "+", "-", "|"},
	},
}

// sym is the set in use. It's a package-level variable (shared by every
// screen) because the Symbols setting applies to the whole game at once;
// only the main goroutine draws, so nothing else reads it.
var sym = &fancy

// Sym returns the symbol set in use.
func Sym() *Symbols { return sym }

// UseSimple switches to the simple set (true) or the fancy one (false).
func UseSimple(on bool) {
	if on {
		sym = &simple
	} else {
		sym = &fancy
	}
}
