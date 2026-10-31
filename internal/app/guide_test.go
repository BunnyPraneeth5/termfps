package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// section returns the guide section called name.
func section(t *testing.T, a *App, name string) guideSection {
	t.Helper()
	for _, s := range a.guideSections() {
		if s.name == name {
			return s
		}
	}
	t.Fatalf("no guide section %q", name)
	return guideSection{}
}

// pageText is all of a page's text in one string.
func pageText(e guideEntry) string {
	return strings.Join(e.lines, "\n")
}

// The title menu opens the guide; Esc comes back to the title.
func TestGuideFromTitle(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, down, down, down, enter) // Guide
	press(a, &now)
	if !screenHas(s, "GUIDE") || !screenHas(s, "Enemies") || !screenHas(s, "Maps") {
		t.Fatal("Guide on the title menu doesn't show the guide's sections")
	}
	press(a, &now, down, enter) // Enemies
	press(a, &now, enter)       // Zombie
	press(a, &now)
	if !screenHas(s, "Zombie (1/5)") || !screenHas(s, fmt.Sprintf("HP %d.", game.ZombieHP)) {
		t.Fatal("the zombie page isn't shown")
	}
	press(a, &now, right)
	press(a, &now)
	if !screenHas(s, "Shooter (2/5)") {
		t.Error("Right didn't go to the next page")
	}
	press(a, &now, esc, esc, esc) // page -> enemies -> guide -> title
	press(a, &now)
	if !screenHas(s, "TERMFPS") {
		t.Error("Esc three times didn't get back to the title menu")
	}
}

// The pause menu opens the guide over the frozen game and returns to it.
func TestGuideFromPause(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, enter, enter) // Play, Survival
	press(a, &now, esc, down, down, down, enter)
	press(a, &now)
	if !screenHas(s, "GUIDE") {
		t.Fatal("Guide on the pause menu doesn't open the guide")
	}
	press(a, &now, esc)
	press(a, &now)
	if !screenHas(s, "PAUSED") {
		t.Error("Esc in the guide didn't go back to the pause menu")
	}
}

// Every enemy kind, gun, pickup kind and map has a page.
func TestGuideCoversEverything(t *testing.T) {
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0], a.maps[0])
	if n := len(section(t, a, "Enemies").entries); n != int(game.KindExploder)+1 {
		t.Errorf("%d enemy pages, want %d", n, game.KindExploder+1)
	}
	if n := len(section(t, a, "Weapons").entries); n != int(game.NumGuns) {
		t.Errorf("%d weapon pages, want %d", n, game.NumGuns)
	}
	if n := len(section(t, a, "Pickups").entries); n != int(game.GunPickup)+1 {
		t.Errorf("%d pickup pages, want %d", n, game.GunPickup+1)
	}
	if n := len(section(t, a, "Maps").entries); n != len(a.maps) {
		t.Errorf("%d map pages, want %d", n, len(a.maps))
	}
}

// The pages' numbers come from the game code.
func TestGuideNumbersFromCode(t *testing.T) {
	a, _ := newMenuApp(t)
	enemies := section(t, a, "Enemies").entries
	checks := []struct {
		page, want string
	}{
		{pageText(enemies[0]), fmt.Sprintf("HP %d.", game.ZombieHP)},
		{pageText(enemies[3]), fmt.Sprintf("HP %d.", game.TankHP)},
		{pageText(enemies[2]), fmt.Sprintf("Comes from tier %d", game.FirstTier(game.KindRunner))},
		{pageText(section(t, a, "Weapons").entries[1]), fmt.Sprintf("%d rounds in the gun", game.Shotgun.Stats().Mag)},
		{pageText(section(t, a, "Pickups").entries[1]), fmt.Sprintf("+%d HP", game.MedkitHeal)},
		{pageText(section(t, a, "Hazards").entries[0]), fmt.Sprintf("up to %d damage", game.BarrelDamage)},
		{pageText(section(t, a, "Survival and waves").entries[0]), fmt.Sprintf("a new enemy every %s s", num(game.TierFor(2).SpawnEvery))},
		{pageText(section(t, a, "Survival and waves").entries[2]), fmt.Sprintf("%d waves", game.NumWaves)},
	}
	for _, c := range checks {
		if !strings.Contains(c.page, c.want) {
			t.Errorf("page %q doesn't say %q", c.page, c.want)
		}
	}
}

// A locked map shows only its name and that it's locked (a surprise).
func TestGuideLockedMap(t *testing.T) {
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0])
	maps := section(t, a, "Maps").entries
	if strings.Contains(pageText(maps[0]), "Locked") {
		t.Error("the first (open) map shows as locked")
	}
	locked := maps[1]
	if !strings.Contains(locked.name, "locked") || !strings.Contains(pageText(locked), "Locked") ||
		strings.Contains(pageText(locked), "cells") || locked.pic != nil {
		t.Errorf("locked map page %q: %q; want only that it's locked", locked.name, pageText(locked))
	}
	a.cfg.UnlockRule = "open"
	if strings.Contains(pageText(section(t, a, "Maps").entries[1]), "Locked") {
		t.Error("with all maps open, the second map is still locked in the guide")
	}
}

// Every page fits 80x24: the title and key help show, and scrolling down
// reaches its last line.
func TestGuideFits80x24(t *testing.T) {
	a, s := newMenuApp(t)
	for _, sec := range a.guideSections() {
		for i, e := range sec.entries {
			g := &guidePage{back: a.current, section: sec, index: i}
			g.tick(a, t0, 0)
			if !screenHas(s, "GUIDE: ") || !screenHas(s, "Esc back") {
				t.Fatalf("%s: title or key help missing at 80x24", e.name)
			}
			for range 40 {
				g.key(a, term.Key{Code: term.KeyDown})
			}
			g.tick(a, t0, 0)
			textX := 1 // as tick lays it out
			if e.pic != nil {
				textX = guidePicCols + 3
			}
			lines := g.lines(e, 80-textX-1)
			last := lines[len(lines)-1]
			if !screenHas(s, last) {
				t.Errorf("%s: scrolled to the end, but the last line %q isn't shown", e.name, last)
			}
		}
	}
}
