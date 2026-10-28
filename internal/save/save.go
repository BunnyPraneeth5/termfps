// Package save reads and writes save.json: the settings and the progress
// (unlocked maps, bests per map, total kills). See DECISIONS.md D25.
// Writes are atomic: a temp file, then a rename over the real file.
package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// Version is the save file format version.
const Version = 1

// Data is everything in save.json.
type Data struct {
	Version int `json:"version"`
	// Settings is nil until the game has saved once, so a first run uses
	// the flag defaults. A pointer can be nil; a plain struct can't.
	Settings *Settings `json:"settings,omitempty"`
	Progress Progress  `json:"progress"`
}

// Settings are the Settings menu values. The text after each field is a
// struct tag: it names the field in the JSON file.
type Settings struct {
	RenderMode     string  `json:"render_mode"` // game.RenderMode name, e.g. "half-block"
	FPS            int     `json:"fps"`
	MinimapAtStart bool    `json:"minimap_at_start"`
	Textures       bool    `json:"textures"`
	Sound          bool    `json:"sound"`
	Volume         int     `json:"volume"` // percent
	KeyMode        string  `json:"key_mode"`
	PixelAspect    float64 `json:"pixel_aspect"`
	// The S3 settings are pointers so a save file from before S3, which
	// doesn't have them, gives nil (use the default), not 0.
	MixedBackground *int `json:"mixed_background,omitempty"` // percent, 0-100
	ColorThreshold  *int `json:"color_threshold,omitempty"`  // per channel, 0 = off
	// RenderScale is "auto", "full" or "half" (V6); "" in files from before
	// V6 means auto.
	RenderScale string `json:"render_scale,omitempty"`
}

// Progress is what carries over between runs.
type Progress struct {
	UnlockedMaps int             `json:"unlocked_maps"` // how many maps are open, from the first
	Best         map[string]Best `json:"best"`          // by map name
	TotalKills   int             `json:"total_kills"`
}

// Best holds the best result on one map. Each field is its own best, so
// they can come from different runs.
type Best struct {
	Kills int     `json:"kills"`
	Time  float64 `json:"time"` // seconds survived
	Tier  int     `json:"tier"`
}

// Store is the save file's location.
type Store struct {
	Path string // e.g. %AppData%\termfps\save.json
}

// DefaultPath is save.json in the user's config folder (%AppData% on
// Windows, ~/.config on Linux).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the config folder: %w", err)
	}
	return filepath.Join(dir, "termfps", "save.json"), nil
}

// Load reads the save file. A missing file is a first run: empty Data and
// no problem. A file that can't be used is renamed to save.json.bad and
// empty Data comes back with problem saying what happened, for the menu.
func Load(s Store) (d Data, problem string) {
	raw, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Data{}, ""
	}
	if err != nil {
		// Unreadable but maybe fine (e.g. locked): don't rename it.
		return Data{}, fmt.Sprintf("Couldn't read the save file, using defaults: %v", err)
	}
	err = json.Unmarshal(raw, &d)
	if err == nil {
		err = d.check()
	}
	if err != nil {
		bad := s.Path + ".bad"
		if rerr := os.Rename(s.Path, bad); rerr != nil {
			return Data{}, fmt.Sprintf("The save file is damaged (%v) and couldn't be moved aside (%v). Using defaults.", err, rerr)
		}
		return Data{}, fmt.Sprintf("The save file was damaged (%v). Kept it as %s, using defaults.", err, filepath.Base(bad))
	}
	return d, ""
}

// check reports the first value that the game can't use.
func (d *Data) check() error {
	if d.Version != Version {
		return fmt.Errorf("version %d, want %d", d.Version, Version)
	}
	p := d.Progress
	if p.UnlockedMaps < 0 || p.TotalKills < 0 {
		return errors.New("negative progress")
	}
	if s := d.Settings; s != nil {
		if _, ok := game.ParseRenderMode(s.RenderMode); !ok {
			return fmt.Errorf("unknown render mode %q", s.RenderMode)
		}
		switch {
		case s.FPS < 1 || s.FPS > 240:
			return fmt.Errorf("fps %d out of range 1-240", s.FPS)
		case s.Volume < 0 || s.Volume > 100:
			return fmt.Errorf("volume %d out of range 0-100", s.Volume)
		case s.KeyMode != "async" && s.KeyMode != "events":
			return fmt.Errorf("unknown key mode %q", s.KeyMode)
		case s.PixelAspect < 0.5 || s.PixelAspect > 2.0:
			return fmt.Errorf("pixel aspect %v out of range 0.5-2.0", s.PixelAspect)
		case s.MixedBackground != nil && (*s.MixedBackground < 0 || *s.MixedBackground > 100):
			return fmt.Errorf("mixed background %d out of range 0-100", *s.MixedBackground)
		case s.ColorThreshold != nil && (*s.ColorThreshold < 0 || *s.ColorThreshold > 255):
			return fmt.Errorf("color threshold %d out of range 0-255", *s.ColorThreshold)
		case s.RenderScale != "" && s.RenderScale != "auto" && s.RenderScale != "full" && s.RenderScale != "half":
			return fmt.Errorf("unknown render scale %q", s.RenderScale)
		}
	}
	return nil
}

// Write saves d atomically: it writes save.json.tmp next to the save file,
// then renames it over save.json, so a crash mid-write never leaves half a
// file. The folder is created if needed.
func Write(s Store, d Data) error {
	d.Version = Version
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create save folder: %w", err)
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(tmp), err)
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp) // best effort; the rename error is what matters
		return fmt.Errorf("replace save file: %w", err)
	}
	return nil
}
