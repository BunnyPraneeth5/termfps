package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// What the player has found out over all runs (M18c), for the guide:
// enemies met, guns found, and the parts of each map they have seen.

// gunName is how a gun is named in the save file.
func gunName(k game.GunKind) string {
	return strings.ToLower(k.Stats().Name)
}

// recordFinds notes the enemies the player met and the guns they found in
// the last game.Update. They're saved with the rest of the progress.
func (a *App) recordFinds() {
	for _, o := range a.state.Origins {
		if o.Event == game.EventEnemyNoticed && !slices.Contains(a.progress.Met, o.Kind.String()) {
			a.progress.Met = append(a.progress.Met, o.Kind.String())
		}
	}
	if g := gunName(a.state.Player.Gun); a.state.HasEvent(game.EventGunTaken) && !slices.Contains(a.progress.Found, g) {
		a.progress.Found = append(a.progress.Found, g)
	}
}

// met reports whether the player has met enemy kind k.
func (a *App) met(k game.EnemyKind) bool {
	return slices.Contains(a.progress.Met, k.String())
}

// found reports whether the player has found gun k (the pistol always).
func (a *App) found(k game.GunKind) bool {
	return k == game.Pistol || slices.Contains(a.progress.Found, gunName(k))
}

// explored returns the cells of map i seen over all runs, and whether the
// map has been played at all. A map never played gives nil, false.
func (a *App) explored(i int) (seen []bool, played bool) {
	m := a.maps[i]
	e, ok := a.progress.Explored[m.Name]
	if !ok || e.W != m.W || e.H != m.H {
		return nil, false // never played, or the map changed since
	}
	seen, err := save.UnpackCells(e.Cells, m.W*m.H)
	if err != nil {
		return nil, false // mergeExplored reports a damaged entry when it replaces it
	}
	return seen, true
}

// mergeExplored adds the run's seen cells to the map's explored record.
// A record for a different map size (the map was edited) starts over; a
// damaged one too, and the menus say so.
func (a *App) mergeExplored() {
	m := a.maps[a.mapIndex]
	seen := slices.Clone(a.state.Seen)
	if e, ok := a.progress.Explored[m.Name]; ok && e.W == m.W && e.H == m.H {
		old, err := save.UnpackCells(e.Cells, m.W*m.H)
		if err != nil {
			a.loadMsg = fmt.Sprintf("The explored part of %s in the save file was damaged and starts over: %v", m.Name, err)
		}
		for i := range old {
			seen[i] = seen[i] || old[i]
		}
	}
	a.progress.Explored[m.Name] = save.Explored{W: m.W, H: m.H, Cells: save.PackCells(seen)}
}

// exploredShare is the share (0-1) of a map's open cells (everything but
// walls) in seen.
func exploredShare(m *world.Map, seen []bool) float64 {
	open, saw := 0, 0
	for i, c := range m.Cells {
		if c.IsWall() {
			continue
		}
		open++
		if i < len(seen) && seen[i] {
			saw++
		}
	}
	if open == 0 {
		return 0
	}
	return float64(saw) / float64(open)
}
