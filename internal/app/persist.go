package app

import (
	"fmt"
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/save"
)

// Saved is the save file as loaded at startup (P1-13).
type Saved struct {
	Store    *save.Store // where to save; nil = don't save (no config folder)
	Progress save.Progress
	Message  string // a problem to show in the menus; "" = none
}

// ApplySettings returns base with the saved settings s on top. note says
// what couldn't be used on this computer ("" = everything was used).
// save.Load has already checked the values.
func ApplySettings(base Config, s save.Settings) (c Config, note string) {
	c = base
	c.Mode, _ = game.ParseRenderMode(s.RenderMode) // "_" ignores the ok result: Load checked the name
	c.FPS = s.FPS
	c.MinimapAtStart = s.MinimapAtStart
	c.Textures = s.Textures
	c.Sound = s.Sound
	c.Volume = s.Volume
	c.PixelAspect = s.PixelAspect
	// Missing in save files from before S3: keep base's defaults.
	if s.MixedBackground != nil {
		c.MixedBackground = *s.MixedBackground
	}
	if s.ColorThreshold != nil {
		c.ColorThreshold = *s.ColorThreshold
	}
	if s.RenderScale != "" { // missing before V6: keep base's
		c.RenderScale = s.RenderScale
	}
	if s.UnlockRule != "" { // missing before M7
		c.UnlockRule = s.UnlockRule
	}
	if s.KeyMode == "async" && input.AsyncSupported() != nil {
		note = fmt.Sprintf("The saved key mode async doesn't work here; using %s.", base.KeyMode)
	} else {
		c.KeyMode = s.KeyMode
	}
	return c, note
}

// useSaved takes the progress and the load problem from the save file.
func (a *App) useSaved(s Saved) {
	a.store = s.Store
	a.loadMsg = s.Message
	a.progress = s.Progress
	if a.progress.Best == nil {
		a.progress.Best = map[string]save.Best{} // a nil map can't be written to
	}
	a.progress.UnlockedMaps = min(max(a.progress.UnlockedMaps, 1), len(a.maps))
	a.switchTo(a.titleMenu()) // rebuilt, so it shows the load problem
}

// settings are the current settings in save-file form.
func (a *App) settings() save.Settings {
	return save.Settings{
		RenderMode:     a.state.Mode.String(),
		FPS:            a.cfg.FPS,
		MinimapAtStart: a.cfg.MinimapAtStart,
		Textures:       a.cfg.Textures,
		Sound:          a.cfg.Sound,
		Volume:         a.cfg.Volume,
		KeyMode:        a.cfg.KeyMode,
		PixelAspect:    a.cfg.PixelAspect,
		// Pointers to copies, so the save data never points into the config.
		MixedBackground: ptr(a.cfg.MixedBackground),
		ColorThreshold:  ptr(a.cfg.ColorThreshold),
		RenderScale:     a.cfg.RenderScale,
		UnlockRule:      a.cfg.UnlockRule,
	}
}

// ptr returns a pointer to a copy of v. [T any] makes it generic: it works
// for any type.
func ptr[T any](v T) *T { return &v }

// persist writes settings and progress to the save file. A failure is
// shown in the menus until a later save works.
func (a *App) persist() {
	if a.store == nil {
		return
	}
	s := a.settings()
	if err := save.Write(*a.store, save.Data{Settings: &s, Progress: a.progress}); err != nil {
		a.saveErr = "Couldn't save: " + err.Error()
		return
	}
	a.saveErr = ""
}

// endRun records the current run's result (bests on its map, total kills)
// and saves. It does nothing if the run was already recorded.
func (a *App) endRun() {
	if !a.runActive {
		return
	}
	a.runActive = false
	st := a.state
	name := a.maps[a.mapIndex].Name
	b := a.progress.Best[name] // a missing key gives the zero Best
	b.Kills = max(b.Kills, st.Kills)
	b.Time = max(b.Time, st.Time)
	b.Tier = max(b.Tier, st.Tier)
	a.progress.Best[name] = b
	a.progress.TotalKills += st.Kills
	a.persist()
}

// messages are the save and sound problems for the menus, wrapped to fit
// the box.
func (a *App) messages() []string {
	var out []string
	for _, m := range []string{a.loadMsg, a.saveErr, a.soundProblem()} {
		if m != "" {
			out = append(out, wrap(m, 60)...)
		}
	}
	return out
}

// wrap splits text into lines of at most width characters, at spaces.
// A single word longer than width gets a line of its own.
func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
