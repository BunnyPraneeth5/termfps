package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// Guide pages (M18a) in a centered panel that looks like the menus (M18c,
// D53): a light box with the 3D view as a dark window inside it.

// pageKind is how a guide page is laid out.
type pageKind int

const (
	pageText pageKind = iota // text, with a 3D view on the left if it has one
	pageKeys                 // the keyboard and mouse picture (M18c)
	pageHUD                  // the labeled HUD example (M18c)
	pageMap                  // a big top-down map picture, the text under it (M18c)
)

// Panel and view box sizes.
const (
	guideMaxW   = 100 // the panel grows with the window up to this wide, then stays centered
	guideMaxH   = 40
	viewMinCols = 22 // the 3D view box: at least this wide...
	viewMaxCols = 40 // ...and at most this wide
)

// panelStyle is the guide panel: reversed, like the menus.
var panelStyle = term.Style{Reverse: true}

// rect is a block of screen cells.
type rect struct{ x, y, w, h int }

// guidePanel is where the panel goes on a w x h screen: centered, up to
// guideMaxW x guideMaxH, one cell in from each edge.
func guidePanel(w, h int) rect {
	pw, ph := min(w-2, guideMaxW), min(h-2, guideMaxH)
	return rect{x: (w - pw) / 2, y: (h - ph) / 2, w: pw, h: ph}
}

// viewSize is the 3D view box for a text page's content area: about a
// third of its width (viewMinCols to viewMaxCols), about square.
func viewSize(area rect) (cols, rows int) {
	cols = min(max(area.w*32/100, viewMinCols), viewMaxCols)
	rows = min(int(math.Round(float64(cols)/2.2)), area.h)
	return cols, rows
}

// guidePage shows one page of a section: Left/Right go to the other pages,
// Up/Down scroll (or pick a key on the controls page), Esc goes back.
type guidePage struct {
	back    screen
	section guideSection
	index   int // the page shown
	scroll  int // first text line shown
	keySel  int // the key lit on the controls page (M18c)

	// The 3D view (M18b): its Frame, a map's top-down picture, the game
	// state it draws, the page that state is for (-1 = none yet), and the
	// seconds since that page opened.
	view, mapPic render.Frame
	preview      *game.State
	shown        int
	t            float64
}

func (g *guidePage) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	n := len(g.section.entries)
	keys := g.section.entries[g.index].kind == pageKeys
	switch {
	case k == ui.KeyBack:
		a.switchTo(g.back)
	case k == ui.KeyLeft:
		g.index, g.scroll = (g.index+n-1)%n, 0
	case k == ui.KeyRight:
		g.index, g.scroll = (g.index+1)%n, 0
	case k == ui.KeyUp && keys:
		g.keySel = (g.keySel + len(guideKeys) - 1) % len(guideKeys)
	case k == ui.KeyDown && keys:
		g.keySel = (g.keySel + 1) % len(guideKeys)
	case k == ui.KeyUp:
		g.scroll = max(g.scroll-1, 0)
	case k == ui.KeyDown:
		g.scroll++ // drawing keeps it within the text
	}
}

func (g *guidePage) mouse(a *App, b mouseButton, down bool, x, y int) {
	switch {
	case b == mouseWheelUp:
		g.scroll = max(g.scroll-1, 0)
	case b == mouseWheelDown:
		g.scroll++
	}
}

// The key help at the bottom of a guide page.
const (
	guideHint     = "Left/Right other pages   Up/Down scroll   Esc back"
	guideKeysHint = "Left/Right other pages   Up/Down pick a key   Esc back"
)

func (g *guidePage) tick(a *App, now time.Time, dt float64) {
	s := a.scr
	s.Clear()
	w, h := s.Size()
	e := g.section.entries[g.index]
	p := guidePanel(w, h)
	for y := p.y; y < p.y+p.h; y++ {
		term.PutStr(s, p.x, y, strings.Repeat(" ", p.w), panelStyle)
	}
	title := fmt.Sprintf("GUIDE: %s: %s (%d/%d)", strings.ToUpper(g.section.name), e.name, g.index+1, len(g.section.entries))
	putCentered(s, p, p.y, title, term.Style{Reverse: true, Bold: true})
	hint := guideHint
	if e.kind == pageKeys {
		hint = guideKeysHint
	}
	putCentered(s, p, p.y+p.h-1, hint, panelStyle)

	area := rect{x: p.x + 2, y: p.y + 2, w: p.w - 4, h: p.h - 4}
	switch e.kind {
	case pageKeys:
		g.drawKeys(a, e, area)
	case pageHUD:
		g.drawHUDPage(a, e, area)
	case pageMap:
		g.drawMapPage(a, e, area, dt)
	default:
		if e.view.kind == viewNone {
			g.drawText(a, e.lines, area)
			break
		}
		cols, rows := viewSize(area)
		g.drawView(a, e.view, rect{x: area.x, y: area.y, w: cols, h: rows}, dt)
		g.drawText(a, e.lines, rect{x: area.x + cols + 2, y: area.y, w: area.w - cols - 2, h: area.h})
	}
	s.Show()
}

// putCentered writes text centered in panel p on row y.
func putCentered(s term.Screen, p rect, y int, text string, st term.Style) {
	n := len([]rune(text))
	term.PutStr(s, p.x+max((p.w-n)/2, 0), y, text, st)
}

// drawView draws the page's 3D view into box r (M18b), restarting it when
// the page changed.
func (g *guidePage) drawView(a *App, v guideView, r rect, dt float64) {
	if g.preview == nil || g.shown != g.index {
		g.preview, g.shown, g.t = a.newPreview(v), g.index, 0
	}
	g.t += dt
	g.view.Resize(r.w, 2*r.h)
	g.view.OffsetX, g.view.OffsetY = r.x, r.y
	a.drawView(&g.view, g.preview, v, g.t, dt)
	a.presentFrame(&g.view)
}

// drawText writes paragraphs into box r on the panel, wrapped, from the
// scroll position; when they don't all fit, the last row says so.
func (g *guidePage) drawText(a *App, paragraphs []string, r rect) {
	if r.h <= 0 || r.w <= 0 {
		return // no room left on this page (the keyboard fills it at 80x24)
	}
	lines := wrapParagraphs(paragraphs, r.w)
	rows := r.h
	if len(lines) > rows {
		rows-- // the last row tells there's more
	}
	g.scroll = min(g.scroll, max(len(lines)-rows, 0))
	for i := 0; i < rows && g.scroll+i < len(lines); i++ {
		term.PutStr(a.scr, r.x, r.y+i, lines[g.scroll+i], panelStyle)
	}
	if rows < r.h {
		more := "(more: Up/Down)"
		if g.scroll+rows >= len(lines) {
			more = "(the end: Up to go back)"
		}
		term.PutStr(a.scr, r.x, r.y+r.h-1, more, term.Style{Reverse: true, Dim: true})
	}
}

// wrapParagraphs wraps paragraphs to width, with a blank line between them.
func wrapParagraphs(paragraphs []string, width int) []string {
	var out []string
	for i, p := range paragraphs {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, wrap(p, max(width, 10))...)
	}
	return out
}

// drawMapPage draws a map page (M18c): a big top-down picture across the
// panel, explored parts only (all dark but the border if the map is locked
// or never played); under it the 360° turn once the map has been played,
// and the text.
func (g *guidePage) drawMapPage(a *App, e guideEntry, area rect, dt float64) {
	m := a.maps[e.mapIdx]
	seen, played := a.explored(e.mapIdx)
	if !played || a.mapLocked(e.mapIdx) {
		seen = make([]bool, m.W*m.H) // nothing seen: only the border shows
	}
	picRows := max(area.h*55/100, 4)
	g.mapPic.PixelAspect = 1
	g.mapPic.Resize(area.w, 2*picRows)
	g.mapPic.OffsetX, g.mapPic.OffsetY = area.x, area.y
	render.DrawMapPicture(&g.mapPic, m, seen)
	a.presentFrame(&g.mapPic)

	below := rect{x: area.x, y: area.y + picRows + 1, w: area.w, h: area.h - picRows - 1}
	if e.view.kind == viewMap {
		rows := below.h
		cols := min(int(math.Round(float64(rows)*2.2)), below.w/3)
		g.drawView(a, e.view, rect{x: below.x, y: below.y, w: cols, h: rows}, dt)
		below.x, below.w = below.x+cols+2, below.w-cols-2
	}
	g.drawText(a, e.lines, below)
}

// keyGroup is a group of keys on the controls page, each its own color.
type keyGroup int

const (
	groupMove keyGroup = iota
	groupFight
	groupView
	groupOther
)

// groupColors and groupNames, in group order.
var (
	groupColors = [...]term.Color{term.ColorGreen, term.ColorRed, term.ColorBlue, term.ColorYellow}
	groupNames  = [...]string{"move", "fight", "view", "other"}
)

// guideKey is a key the game uses: its cap (as drawn), group and what it does.
type guideKey struct {
	cap   string
	group keyGroup
	desc  string
}

// guideKeys are the keys and mouse buttons the game uses, in the order
// Up/Down steps through them. The mouse ones have caps "LMB", "RMB" and
// "Wheel" (drawn as L, R and the wheel).
var guideKeys = []guideKey{
	{"w", groupMove, "W (or the Up arrow): move forward."},
	{"s", groupMove, "S (or the Down arrow): move back."},
	{"a", groupMove, "A: step left."},
	{"d", groupMove, "D: step right."},
	{"q", groupMove, "Q (or the Left arrow): turn left."},
	{"e", groupMove, "E (or the Right arrow): turn right."},
	{"↑", groupMove, "Up arrow: move forward, like W."},
	{"↓", groupMove, "Down arrow: move back, like S."},
	{"←", groupMove, "Left arrow: turn left, like Q."},
	{"→", groupMove, "Right arrow: turn right, like E."},
	{"Space", groupFight, "Space: shoot. Hold it to keep firing."},
	{"r", groupFight, "R: reload. An empty gun reloads by itself."},
	{"1", groupFight, "1: the pistol."},
	{"2", groupFight, "2: the shotgun, once you've found it."},
	{"3", groupFight, "3: the rifle, once you've found it."},
	{"LMB", groupFight, "Left mouse button: shoot, like Space."},
	{"RMB", groupFight, "Right mouse button: reload, like R."},
	{"Wheel", groupFight, "Mouse wheel: the next or previous gun you have."},
	{"m", groupView, "M: show or hide the minimap."},
	{"v", groupView, "V: switch the look: half-block, shade, ASCII, mixed."},
	{"f", groupOther, fmt.Sprintf("F (hold): sprint, %sx speed, while you have stamina.", num(game.SprintMul))},
	{"Esc", groupOther, "Esc: pause; in menus and the guide, go back."},
	{"Ctrl", groupOther, "Ctrl+C: quit at once, from anywhere."},
}

// hudParts names the parts of the HUD line, by how each part starts.
var hudParts = []struct{ prefix, name string }{
	{"HP", "health, armor"},
	{"PISTOL", "gun: in it / spare"}, {"SHOTGUN", "gun: in it / spare"}, {"RIFLE", "gun: in it / spare"},
	{"KILLS", "kills"},
	{"TIER", "tier or wave"}, {"WAVE", "tier or wave"},
	{"STAMINA", "stamina"},
	{"FPS", "frames per second"},
}

// hudPartName names one part of the HUD line.
func hudPartName(part string) string {
	for _, p := range hudParts {
		if strings.HasPrefix(part, p.prefix) {
			return p.name
		}
	}
	switch {
	case strings.Contains(part, ":") && len(part) <= 6:
		return "time survived"
	case strings.Contains(part, "°"):
		return "facing"
	}
	return "render mode"
}

// guideHUD is the example HUD on the HUD page.
func guideHUD() render.HUD {
	p := game.Player{HP: 76, Armor: 50, Guns: game.StartGuns(), Stamina: 0.6}
	p.Guns[game.Pistol].Mag, p.Guns[game.Pistol].Reserve = 18, 50
	return render.HUD{Player: p, Kills: 23, Tier: 2, Time: 187, FPS: 60, Mode: game.ModeHalfBlock}
}

// drawHUDPage draws an example HUD line with each part pointed out and
// named (M18c), then the page's notes.
func (g *guidePage) drawHUDPage(a *App, e guideEntry, area rect) {
	s := a.scr
	line := render.HUDLine(guideHUD(), area.w)
	term.PutStr(s, area.x, area.y, line+strings.Repeat(" ", max(area.w-len([]rune(line)), 0)), term.Style{})
	parts := strings.Split(line, " | ")
	starts := make([]int, len(parts))
	col := 0
	for i, p := range parts {
		starts[i] = col
		col += len([]rune(p)) + 3
	}
	// Labels on two levels so neighbors don't overlap: even parts on the
	// first, odd ones on the second; each stops before the next one on
	// its level.
	for i := range parts {
		level := i % 2
		end := area.w // an upper label stops at the next part's line down, a lower one at the next lower label
		if next := i + 1 + level; next < len(parts) {
			end = starts[next] - 1
		}
		for r := 1; r <= 1+2*level; r++ {
			term.PutStr(s, area.x+starts[i], area.y+r, "│", panelStyle)
		}
		label := []rune("└ " + hudPartName(parts[i]))
		label = label[:min(len(label), max(end-starts[i], 0))]
		term.PutStr(s, area.x+starts[i], area.y+2+2*level, string(label), panelStyle)
	}
	g.drawText(a, e.lines, rect{x: area.x, y: area.y + 6, w: area.w, h: area.h - 6})
}

// keyCap is one key on the keyboard picture: its label and its width in
// cells, borders included. An empty label is a gap.
type keyCap struct {
	cap string
	w   int
}

// keyboardWidth is how wide the keyboard picture is: it fits the guide
// panel at 80 columns exactly.
const keyboardWidth = 74

// keyboardRows is the keyboard picture (after the user's reference
// picture, M18c): each key a rounded box 3 lines tall, the arrow keys at
// the bottom right with up above down, as on a laptop. Every row is
// keyboardWidth wide.
var keyboardRows = [][]keyCap{
	{{"Esc", 5}, {"1", 5}, {"2", 5}, {"3", 5}, {"4", 5}, {"5", 5}, {"6", 5}, {"7", 5}, {"8", 5}, {"9", 5}, {"0", 5}, {"-", 5}, {"=", 5}, {"Bksp", 9}},
	{{"Tab", 9}, {"q", 5}, {"w", 5}, {"e", 5}, {"r", 5}, {"t", 5}, {"y", 5}, {"u", 5}, {"i", 5}, {"o", 5}, {"p", 5}, {"[", 5}, {"]", 5}, {"\\", 5}},
	{{"Caps", 10}, {"a", 5}, {"s", 5}, {"d", 5}, {"f", 5}, {"g", 5}, {"h", 5}, {"j", 5}, {"k", 5}, {"l", 5}, {";", 5}, {"'", 5}, {"Enter", 9}},
	{{"Shift", 12}, {"z", 5}, {"x", 5}, {"c", 5}, {"v", 5}, {"b", 5}, {"n", 5}, {"m", 5}, {",", 5}, {".", 5}, {"/", 5}, {"", 2}, {"↑", 5}, {"", 5}},
	{{"Ctrl", 6}, {"Alt", 5}, {"Space", 48}, {"←", 5}, {"↓", 5}, {"→", 5}},
}

// keyLook is how a key cap is drawn: the style of its border and of its
// label. Unused keys are dim; used ones take their group's color; the
// selected one is filled with it.
func keyLook(cap string, sel string) (border, label term.Style) {
	for _, k := range guideKeys {
		if k.cap != cap {
			continue
		}
		c := groupColors[k.group]
		if cap == sel {
			return term.Style{Fg: c, Bold: true}, term.Style{Fg: c, Reverse: true, Bold: true}
		}
		return term.Style{Fg: c}, term.Style{Fg: c, Bold: true}
	}
	return term.Style{Dim: true}, term.Style{Dim: true}
}

// drawKeyCap draws one key as a rounded box at (x, y), w wide, 3 lines tall.
func drawKeyCap(s term.Screen, x, y, w int, cap, sel string) {
	border, label := keyLook(cap, sel)
	inner := w - 2
	pad := inner - len([]rune(cap))
	text := strings.Repeat(" ", pad/2) + cap + strings.Repeat(" ", pad-pad/2)
	term.PutStr(s, x, y, "╭"+strings.Repeat("─", inner)+"╮", border)
	term.PutStr(s, x, y+1, "│", border)
	term.PutStr(s, x+1, y+1, text, label)
	term.PutStr(s, x+w-1, y+1, "│", border)
	term.PutStr(s, x, y+2, "╰"+strings.Repeat("─", inner)+"╯", border)
}

// mouseWidth is the boxed mouse picture's width; it goes right of the
// keyboard when the panel has room (2 cells apart).
const mouseWidth = 11

// drawKeys draws the controls page (M18c): the keyboard (and mouse) on a
// dark inset with the game's keys colored by group, a legend, what the
// selected key does, and the page's notes under it when there's room.
func (g *guidePage) drawKeys(a *App, e guideEntry, area rect) {
	s := a.scr
	sel := guideKeys[g.keySel].cap
	bigMouse := area.w >= keyboardWidth+2+mouseWidth
	insetW := keyboardWidth
	if bigMouse {
		insetW += 2 + mouseWidth
	}
	rows := 3 * len(keyboardRows)
	for y := area.y; y < area.y+rows; y++ { // the dark inset
		term.PutStr(s, area.x, y, strings.Repeat(" ", insetW), term.Style{})
	}
	for r, caps := range keyboardRows {
		x := area.x
		for _, k := range caps {
			if k.cap != "" {
				drawKeyCap(s, x, area.y+3*r, k.w, k.cap, sel)
			}
			x += k.w
		}
	}

	y := area.y + rows + 1
	x := area.x
	for gi, name := range groupNames {
		term.PutStr(s, x, y, "   ", term.Style{Fg: groupColors[gi], Reverse: true})
		term.PutStr(s, x+4, y, name, panelStyle)
		x += 5 + len(name) + 1
	}
	term.PutStr(s, x, y, "   ", term.Style{Dim: true})
	term.PutStr(s, x+4, y, "unused", panelStyle)
	x += 4 + len("unused") + 2 // the small mouse fits after it at 80 columns

	// The mouse: a boxed picture right of the keyboard when there's room,
	// else small at the end of the legend line.
	mouseLook := func(cap string) term.Style { _, l := keyLook(cap, sel); return l }
	if bigMouse {
		mx := area.x + keyboardWidth + 2
		pic := []string{"╭───┬─┬───╮", "│   │ │   │", "├───┴─┴───┤", "│         │", "│  mouse  │", "╰─────────╯"}
		for i, l := range pic {
			term.PutStr(s, mx, area.y+i, l, term.Style{})
		}
		term.PutStr(s, mx+1, area.y+1, " L ", mouseLook("LMB"))
		term.PutStr(s, mx+5, area.y+1, "↕", mouseLook("Wheel"))
		term.PutStr(s, mx+7, area.y+1, " R ", mouseLook("RMB"))
	} else {
		term.PutStr(s, x, y, "mouse:", panelStyle)
		term.PutStr(s, x+7, y, " L ", mouseLook("LMB"))
		term.PutStr(s, x+11, y, "↕", mouseLook("Wheel"))
		term.PutStr(s, x+13, y, " R ", mouseLook("RMB"))
	}

	desc := fmt.Sprintf("%d/%d  %s", g.keySel+1, len(guideKeys), guideKeys[g.keySel].desc)
	term.PutStr(s, area.x, y+2, desc, term.Style{Reverse: true, Bold: true})
	g.drawText(a, e.lines, rect{x: area.x, y: y + 4, w: area.w, h: area.h - (y + 4 - area.y)})
}
