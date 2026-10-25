// Package ui holds the menus: a menu model (items, selection, values) that
// doesn't know about the terminal, and Draw, which puts a menu on a
// term.Screen. Only draw.go uses term, so the model is tested without one.
package ui

// Key is a menu key, independent of the physical key that produced it.
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyBack
)

// Item is one menu line: an action (Values is nil) or a setting that
// cycles through Values.
type Item struct {
	Label    string
	Values   []string // a setting's choices; nil for an action item
	Value    int      // index into Values
	Disabled bool     // shown, but can't be selected (e.g. a locked map)
}

// Result says what a key or click did.
type Result int

const (
	None    Result = iota // nothing to do (e.g. only the selection moved)
	Chosen                // an action item was picked
	Changed               // a setting's value changed
	Back                  // Esc: leave this menu
)

// Menu is a list of items with one selected.
type Menu struct {
	Title    string
	Text     []string // lines between the title and the items
	Items    []Item
	Selected int

	// layout is where Draw last put the items, for mouse clicks.
	layout layout
}

// layout records the screen rectangle of the item rows.
type layout struct {
	x, w int // columns x to x+w-1
	top  int // screen row of item 0; item i is on row top+i
	n    int // items drawn (0 before the first Draw)
}

// New returns a menu with the first enabled item selected.
func New(title string, text []string, items []Item) *Menu {
	m := &Menu{Title: title, Text: text, Items: items, Selected: -1}
	m.move(1)
	return m
}

// Key applies one menu key.
func (m *Menu) Key(k Key) Result {
	switch k {
	case KeyUp:
		m.move(-1)
	case KeyDown:
		m.move(1)
	case KeyLeft:
		return m.change(-1)
	case KeyRight:
		return m.change(1)
	case KeyEnter:
		return m.activate()
	case KeyBack:
		return Back
	}
	return None
}

// Click selects item i and acts on it like Enter. Disabled items and
// positions outside the list do nothing.
func (m *Menu) Click(i int) Result {
	if i < 0 || i >= len(m.Items) || m.Items[i].Disabled {
		return None
	}
	m.Selected = i
	return m.activate()
}

// ItemAt returns the item drawn at screen cell (x, y) by the last Draw.
func (m *Menu) ItemAt(x, y int) (int, bool) {
	l := m.layout
	i := y - l.top
	if x < l.x || x >= l.x+l.w || i < 0 || i >= l.n {
		return 0, false
	}
	return i, true
}

// Current returns the selected item's value text ("" for an action item).
func (m *Menu) Current() string {
	it := m.Items[m.Selected]
	if it.Values == nil {
		return ""
	}
	return it.Values[it.Value]
}

// activate is Enter: an action is chosen; a setting moves to its next value.
func (m *Menu) activate() Result {
	if m.Selected < 0 {
		return None // every item is disabled
	}
	if m.Items[m.Selected].Values == nil {
		return Chosen
	}
	return m.change(1)
}

// change moves the selected setting's value by d, wrapping around.
func (m *Menu) change(d int) Result {
	if m.Selected < 0 {
		return None
	}
	it := &m.Items[m.Selected] // a pointer, so the change sticks in the slice
	n := len(it.Values)
	if n < 2 {
		return None
	}
	it.Value = ((it.Value+d)%n + n) % n // the extra +n keeps it >= 0
	return Changed
}

// move steps the selection by d (+1 or -1), wrapping around and skipping
// disabled items. With every item disabled, Selected becomes -1.
func (m *Menu) move(d int) {
	n := len(m.Items)
	i := m.Selected
	for range n { // "range n" loops n times
		i = ((i+d)%n + n) % n
		if !m.Items[i].Disabled {
			m.Selected = i
			return
		}
	}
	m.Selected = -1
}
