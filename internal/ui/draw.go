package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// Menu styles: a reversed box (like the in-game notices), the selected item
// in normal colors so it stands out, disabled items dim.
var (
	boxStyle      = term.Style{Reverse: true}
	selectedStyle = term.Style{Bold: true}
	disabledStyle = term.Style{Reverse: true, Dim: true}
)

// hint is the key help under every menu.
const hint = "Up/Down select   Left/Right change   Enter choose   Esc back"

// Draw puts the menu in a box centered on the screen and remembers where the
// items went, so ItemAt can map clicks to items. It only fills the screen's
// buffer; the caller sends it to the terminal with Show.
func Draw(s term.Screen, m *Menu) {
	labels := itemLines(m)

	lines := []string{m.Title, ""}
	if len(m.Text) > 0 {
		lines = append(lines, m.Text...)
		lines = append(lines, "")
	}
	firstItem := len(lines)
	lines = append(lines, labels...)
	lines = append(lines, "", hint)

	width := 0
	for _, l := range lines {
		width = max(width, utf8.RuneCountInString(l))
	}
	width += 4 // two spaces of padding on each side

	w, h := s.Size()
	top := max((h-len(lines)-2)/2, 0)
	left := max((w-width)/2, 0)
	term.PutStr(s, left, top, strings.Repeat(" ", width), boxStyle)
	for i, l := range lines {
		style := boxStyle
		row := padCenter(l, width)
		if i >= firstItem && i < firstItem+len(labels) {
			item := i - firstItem
			row = "  " + padRight(l, width-2) // items line up on the left
			switch {
			case m.Items[item].Disabled:
				style = disabledStyle
			case item == m.Selected:
				style = selectedStyle
			}
		}
		term.PutStr(s, left, top+1+i, row, style)
	}
	term.PutStr(s, left, top+1+len(lines), strings.Repeat(" ", width), boxStyle)

	m.layout = layout{x: left, w: width, top: top + 1 + firstItem, n: len(labels)}
}

// itemLines formats each item: settings show "Label   < value >" with the
// values lined up; selected items get a "> " marker.
func itemLines(m *Menu) []string {
	labelW := 0
	for _, it := range m.Items {
		if it.Values != nil {
			labelW = max(labelW, utf8.RuneCountInString(it.Label))
		}
	}
	out := make([]string, len(m.Items))
	for i, it := range m.Items {
		marker := "  "
		if i == m.Selected {
			marker = "> "
		}
		if it.Values == nil {
			out[i] = marker + it.Label
			continue
		}
		out[i] = fmt.Sprintf("%s%s   < %s >", marker, padRight(it.Label, labelW), it.Values[it.Value])
	}
	return out
}

// padRight pads s with spaces to n characters.
func padRight(s string, n int) string {
	return s + strings.Repeat(" ", max(n-utf8.RuneCountInString(s), 0))
}

// padCenter centers s in n characters.
func padCenter(s string, n int) string {
	pad := max((n-utf8.RuneCountInString(s))/2, 0)
	return padRight(strings.Repeat(" ", pad)+s, n)
}
