package ui

import (
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// The big words are 51 columns wide and their letters never touch: at
// least LetterGap empty columns between letters and WordGap between words
// (the crowded-text rule, D54).
func TestBigTextSpacing(t *testing.T) {
	for _, text := range []string{"TERM FPS", "YOU DIED", "YOU WON"} {
		for _, r := range text {
			if !HasBigLetter(r) {
				t.Fatalf("%q: the block font has no %q", text, r)
			}
		}
		w := BigWidth(text)
		if text != "YOU WON" && w != 51 {
			t.Errorf("BigWidth(%q) = %d, want 51", text, w)
		}
		s := term.NewFake(80, 24)
		BigText(s, 0, 0, text, FontRows, func(row, col int) term.Style { return term.Style{} })
		// A column is used if any of its rows has a block; runs of empty
		// columns between used ones are the gaps.
		used := make([]bool, w)
		for x := range used {
			for y := 0; y < FontRows; y++ {
				if str, _ := s.Get(x, y); str == "█" {
					used[x] = true
				}
			}
		}
		if !used[0] || !used[w-1] {
			t.Errorf("%q: the first or last column of its width is empty", text)
		}
		gap, gaps := 0, []int{}
		for x := 0; x < w; x++ {
			switch {
			case !used[x]:
				gap++
			case gap > 0:
				gaps = append(gaps, gap)
				gap = 0
			}
		}
		words := 0
		for _, g := range gaps {
			if g < LetterGap {
				t.Errorf("%q: a gap of %d columns between letters, want at least %d", text, g, LetterGap)
			}
			if g >= WordGap {
				words++
			}
		}
		if words != 1 {
			t.Errorf("%q: %d word gaps of %d or more columns, want 1 (gaps %v)", text, words, WordGap, gaps)
		}
	}
}

// BigText with fewer rows draws only those (the row-by-row animation).
func TestBigTextRows(t *testing.T) {
	s := term.NewFake(80, 24)
	BigText(s, 0, 0, "TERM", 2, func(row, col int) term.Style { return term.Style{} })
	if strings.Contains(s.Row(2), "█") || !strings.Contains(s.Row(1), "█") {
		t.Error("BigText with 2 rows drew a third row, or not the second")
	}
}

// The small logo says TERM FPS as two words.
func TestSmallLogoTwoWords(t *testing.T) {
	if RuneLen(smallLogo[0]) != RuneLen(smallLogo[1]) {
		t.Fatalf("small logo rows are %d and %d wide", RuneLen(smallLogo[0]), RuneLen(smallLogo[1]))
	}
	if !strings.Contains(smallLogo[0], "   ") || !strings.Contains(smallLogo[1], "   ") {
		t.Error("the small logo has no gap between TERM and FPS")
	}
}

func TestBarCells(t *testing.T) {
	for _, c := range []struct {
		frac float64
		want int
	}{{0, 0}, {0.01, 1}, {0.5, 5}, {0.76, 8}, {0.99, 9}, {1, 10}, {1.5, 10}, {-1, 0}} {
		if got := BarCells(10, c.frac); got != c.want {
			t.Errorf("BarCells(10, %v) = %d, want %d", c.frac, got, c.want)
		}
	}
}

// The simple symbol set draws bars in brackets and boxes with + - |, and
// uses only ASCII; the fancy one is the default.
func TestSymbolSets(t *testing.T) {
	defer UseSimple(false) // put the default back for the other tests
	s := term.NewFake(40, 5)
	Bar(s, 0, 0, 10, 0.7, term.Style{}, term.Style{})
	if got := strings.TrimSpace(s.Row(0)); got != "▰▰▰▰▰▰▰▱▱▱" {
		t.Errorf("fancy bar = %q", got)
	}
	UseSimple(true)
	if !Sym().Simple {
		t.Fatal("UseSimple(true) didn't switch the set")
	}
	end := Bar(s, 0, 1, 10, 0.7, term.Style{}, term.Style{})
	if got := strings.TrimSpace(s.Row(1)); got != "[#######---]" || end != BarWidth(10) {
		t.Errorf("simple bar = %q ending at %d, want [#######---] ending at %d", got, end, BarWidth(10))
	}
	Box(s, 0, 2, 6, 3, BoxRound, term.Style{})
	if got := strings.TrimSpace(s.Row(2)); got != "+----+" {
		t.Errorf("simple box top = %q, want +----+", got)
	}
	m := New("Test", nil, []Item{{Label: "FPS", Values: []string{"30", "60"}}, {Label: "Sound", Values: []string{"off", "on"}, Value: 1}})
	big := term.NewFake(80, 24)
	Draw(big, m)
	text := big.Text()
	for _, r := range text {
		if r > 126 {
			t.Fatalf("the simple set drew %q, which isn't plain ASCII", r)
		}
	}
	for _, want := range []string{"> FPS", "< 30 >", "[x on ]", " Up/Down  select", " Enter  choose"} {
		if !strings.Contains(text, want) {
			t.Errorf("simple menu doesn't show %q", want)
		}
	}
}

// A menu is drawn as a bordered panel with the selected item marked, and
// it fits 80x24 with a long list.
func TestDrawLook(t *testing.T) {
	s := term.NewFake(80, 24)
	items := make([]Item, 15)
	for i := range items {
		items[i] = Item{Label: "Item"}
	}
	m := New("SETTINGS", nil, items)
	Draw(s, m)
	text := s.Text()
	for _, want := range []string{"╭", "╰", "SETTINGS", "▶ Item", " ⏎  choose", " esc  back"} {
		if !strings.Contains(text, want) {
			t.Errorf("menu doesn't show %q", want)
		}
	}
	if _, st := s.Get(m.layout.x+3, m.layout.top); st != StSelected {
		t.Errorf("the selected item's style is %+v, want the amber selection bar", st)
	}
	if got, ok := m.ItemAt(40, m.layout.top+14); !ok || got != 14 {
		t.Errorf("the 15th item isn't on screen at 80x24 (ItemAt = %d, %v)", got, ok)
	}
}

func TestSwitchAndDots(t *testing.T) {
	if Switch(true) != "[■ on ]" || Switch(false) != "[  off]" {
		t.Errorf("switches: %q %q", Switch(true), Switch(false))
	}
	if got := Dots(1, 4); got != "○ ● ○ ○ 2/4" {
		t.Errorf("Dots(1, 4) = %q", got)
	}
	if got := Dots(2, 20); got != "3/20" {
		t.Errorf("Dots(2, 20) = %q, want only the numbers", got)
	}
}

// Hints are centered key caps: a chip per key, then what it does.
func TestHints(t *testing.T) {
	s := term.NewFake(40, 1)
	Hints(s, 0, 40, 0, []Hint{{"esc", "back"}}, ColPanel)
	i := strings.Index(s.Row(0), " esc  back")
	if i < 12 || i > 16 {
		t.Fatalf("hint at column %d of %q, want it centered", i, s.Row(0))
	}
	if _, st := s.Get(i+1, 0); st != StChip {
		t.Errorf("the key isn't drawn as a chip: %+v", st)
	}
}
