package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func testMenu() *Menu {
	return New("Test", nil, []Item{
		{Label: "Play"},
		{Label: "Locked", Disabled: true},
		{Label: "FPS", Values: []string{"30", "60"}},
		{Label: "Quit"},
	})
}

func TestNewSelectsFirstEnabled(t *testing.T) {
	m := New("", nil, []Item{{Label: "a", Disabled: true}, {Label: "b"}})
	if m.Selected != 1 {
		t.Errorf("Selected = %d, want 1 (first enabled)", m.Selected)
	}
	none := New("", nil, []Item{{Label: "a", Disabled: true}})
	if none.Selected != -1 || none.Key(KeyEnter) != None || none.Key(KeyRight) != None {
		t.Error("menu with only disabled items: want Selected -1 and no results")
	}
}

// Up/Down move, wrap around, and skip disabled items.
func TestUpDownSkipDisabledAndWrap(t *testing.T) {
	m := testMenu()
	want := []int{2, 3, 0, 2} // Down from Play skips Locked; Quit wraps to Play
	for i, w := range want {
		m.Key(KeyDown)
		if m.Selected != w {
			t.Fatalf("Down #%d: Selected = %d, want %d", i+1, m.Selected, w)
		}
	}
	m.Key(KeyUp)
	m.Key(KeyUp)
	if m.Selected != 3 {
		t.Errorf("Up Up from FPS: Selected = %d, want 3 (wrapped)", m.Selected)
	}
}

func TestEnterChoosesActions(t *testing.T) {
	m := testMenu()
	if r := m.Key(KeyEnter); r != Chosen {
		t.Errorf("Enter on Play = %v, want Chosen", r)
	}
	if r := m.Key(KeyLeft); r != None {
		t.Errorf("Left on an action = %v, want None", r)
	}
	if r := m.Key(KeyBack); r != Back {
		t.Errorf("Esc = %v, want Back", r)
	}
}

// Left/Right and Enter change a setting's value, wrapping around.
func TestSettingValues(t *testing.T) {
	m := testMenu()
	m.Selected = 2
	if r := m.Key(KeyRight); r != Changed || m.Current() != "60" {
		t.Errorf("Right: %v, value %q; want Changed, 60", r, m.Current())
	}
	if m.Key(KeyRight); m.Current() != "30" {
		t.Errorf("Right at the end: value %q, want 30 (wrapped)", m.Current())
	}
	if m.Key(KeyLeft); m.Current() != "60" {
		t.Errorf("Left at the start: value %q, want 60 (wrapped)", m.Current())
	}
	if r := m.Key(KeyEnter); r != Changed || m.Current() != "30" {
		t.Errorf("Enter on a setting: %v, value %q; want Changed, 30", r, m.Current())
	}
}

func TestClick(t *testing.T) {
	m := testMenu()
	if r := m.Click(3); r != Chosen || m.Selected != 3 {
		t.Errorf("click Quit: %v, Selected %d; want Chosen, 3", r, m.Selected)
	}
	if r := m.Click(1); r != None || m.Selected != 3 {
		t.Errorf("click Locked: %v, Selected %d; want None, selection kept", r, m.Selected)
	}
	if r := m.Click(9); r != None {
		t.Errorf("click outside the list = %v, want None", r)
	}
}

// Draw puts every item on screen and ItemAt maps their rows back to them.
func TestDrawAndItemAt(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 24)
	m := testMenu()
	if _, ok := m.ItemAt(40, 12); ok {
		t.Error("ItemAt found an item before the first Draw")
	}
	Draw(s, m)

	for i, it := range m.Items {
		row := -1
		for y := 0; y < 24; y++ {
			if strings.Contains(rowText(s, y), it.Label) {
				row = y
			}
		}
		if row < 0 {
			t.Fatalf("item %q not drawn", it.Label)
		}
		if got, ok := m.ItemAt(40, row); !ok || got != i {
			t.Errorf("ItemAt(40, %d) = %d, %v; want %d, true", row, got, ok, i)
		}
	}
	if !strings.Contains(screenText(s), "< 30 >") {
		t.Error("setting value not drawn")
	}
	if _, ok := m.ItemAt(0, m.layout.top); ok {
		t.Error("ItemAt found an item left of the box")
	}
}

func rowText(s tcell.Screen, row int) string {
	w, _ := s.Size()
	var b strings.Builder
	for x := 0; x < w; x++ {
		str, _, _ := s.Get(x, row)
		b.WriteString(str)
	}
	return b.String()
}

func screenText(s tcell.Screen) string {
	_, h := s.Size()
	var b strings.Builder
	for y := 0; y < h; y++ {
		b.WriteString(rowText(s, y) + "\n")
	}
	return b.String()
}
