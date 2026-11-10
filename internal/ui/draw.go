package ui

import (
	"fmt"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// MenuHints is the key help under every list menu.
func MenuHints() []Hint {
	return []Hint{{sym.UpDown, "select"}, {sym.LeftRight, "change"}, {sym.Enter, "choose"}, {"esc", "back"}}
}

// Draw puts the menu in a bordered panel centered on the screen (U1: dark
// panel, amber title, the selected item as an amber bar, a key-cap footer)
// and remembers where the items went, so ItemAt can map clicks to items.
// It only fills the screen's buffer; the caller sends it to the terminal
// with Show.
func Draw(s term.Screen, m *Menu) {
	labels := itemLines(m)
	hints := MenuHints()

	width := max(RuneLen(m.Title), hintsWidth(hints))
	for _, l := range m.Text {
		width = max(width, RuneLen(l))
	}
	for _, l := range labels {
		width = max(width, RuneLen(l))
	}
	width += 6 // the border and two spaces of padding on each side

	rows := 2 + len(labels) + 2 // title, blank, items, blank, hints
	if len(m.Text) > 0 {
		rows += len(m.Text) + 1
	}
	w, h := s.Size()
	top := max((h-rows-2)/2, 0)
	left := max((w-width)/2, 0)
	Panel(s, left, top, width, rows+2, BoxRound, StBorder, StPanel)

	y := top + 1
	Center(s, left, width, y, m.Title, StTitle)
	y += 2
	if len(m.Text) > 0 {
		for _, l := range m.Text {
			Center(s, left, width, y, l, StPanel)
			y++
		}
		y++
	}
	firstItem := y
	for i, l := range labels {
		st := StPanel
		switch {
		case m.Items[i].Disabled:
			st = StDim
		case i == m.Selected:
			st = StSelected
		}
		term.PutStr(s, left+1, y, " "+l+pad(width-3-RuneLen(l)), st)
		y++
	}
	Hints(s, left+1, width-2, y+1, hints, ColPanel)

	m.layout = layout{x: left, w: width, top: firstItem, n: len(labels)}
}

// itemLines formats each item: settings show "Label   ◀ value ▶" with the
// values lined up (an off/on setting shows a switch); the selected item
// gets the ▶ marker.
func itemLines(m *Menu) []string {
	labelW := 0
	for _, it := range m.Items {
		if it.Values != nil {
			labelW = max(labelW, RuneLen(it.Label))
		}
	}
	out := make([]string, len(m.Items))
	for i, it := range m.Items {
		marker := pad(RuneLen(sym.Select)) + " "
		if i == m.Selected {
			marker = sym.Select + " "
		}
		if it.Values == nil {
			out[i] = marker + it.Label
			continue
		}
		out[i] = fmt.Sprintf("%s%s   %s", marker, it.Label+pad(labelW-RuneLen(it.Label)), ValueText(it))
	}
	return out
}

// ValueText is how a setting's current value is shown: a switch for an
// off/on setting, else "◀ value ▶".
func ValueText(it Item) string {
	if len(it.Values) == 2 && it.Values[0] == "off" && it.Values[1] == "on" {
		return Switch(it.Value == 1)
	}
	return sym.Prev + " " + it.Values[it.Value] + " " + sym.Next
}
