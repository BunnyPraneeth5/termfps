package game

import (
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Weapon numbers (D22). Balancing (P1-16) may tune them.
const (
	MagazineSize = 25        // rounds per magazine
	StartReserve = 50        // spare rounds at the start
	FireInterval = 1.0 / 3.0 // seconds between shots (3 per second)
	ReloadTime   = 1.5       // seconds a reload takes
	FlashTime    = 0.1       // seconds the muzzle flash (and recoil) shows
)

// Weapon is the player's gun: a magazine, spare rounds, and timers.
type Weapon struct {
	Mag     int // rounds in the gun
	Reserve int // spare rounds

	cooldown  float64 // seconds until the next shot is allowed
	reloading float64 // seconds left in the current reload; 0 = not reloading
	flash     float64 // seconds left of the muzzle flash; 0 = none
}

// NewWeapon returns a full gun with the starting spare rounds.
func NewWeapon() Weapon {
	return Weapon{Mag: MagazineSize, Reserve: StartReserve}
}

// Reloading reports whether a reload is in progress.
func (w Weapon) Reloading() bool {
	return w.reloading > 0
}

// Flashing reports whether the muzzle flash shows (right after a shot).
func (w Weapon) Flashing() bool {
	return w.flash > 0
}

// updateWeapon advances the gun's timers and handles the fire and reload
// inputs for one frame. Shots are hitscan along the view direction.
func updateWeapon(s *State, in Input, dt float64) {
	w := &s.Player.Weapon // pointer: changes below modify the player's gun

	w.cooldown = max(w.cooldown-dt, 0)
	w.flash = max(w.flash-dt, 0)
	if w.reloading > 0 {
		w.reloading -= dt
		if w.reloading <= 0 {
			w.reloading = 0
			take := min(MagazineSize-w.Mag, w.Reserve)
			w.Mag += take
			w.Reserve -= take
			s.Events = append(s.Events, EventReloaded)
		}
	}

	if in.Reload {
		startReload(s)
	}

	if !in.Fire || w.Reloading() || w.cooldown > 0 {
		return
	}
	w.cooldown = FireInterval
	if w.Mag == 0 {
		if w.Reserve == 0 {
			s.Events = append(s.Events, EventEmptyClick)
		} else {
			startReload(s)
		}
		return
	}
	w.Mag--
	w.flash = FlashTime
	s.Events = append(s.Events, EventShotFired)
	shoot(s)
	if w.Mag == 0 {
		startReload(s) // auto-reload (does nothing without spare rounds)
	}
}

// startReload begins a reload if one isn't running, the magazine isn't
// full, and there are spare rounds.
func startReload(s *State) {
	w := &s.Player.Weapon
	if w.Reloading() || w.Mag == MagazineSize || w.Reserve == 0 {
		return
	}
	w.reloading = ReloadTime
	s.Events = append(s.Events, EventReloadStarted)
}

// shoot casts the center ray. The first solid cell stops the shot; a red
// target block there is destroyed. Enemies join this in P1-06.
func shoot(s *State) {
	p := s.Player
	dirX, dirY := p.Dir()
	hit := world.CastRay(s.Map, p.X, p.Y, dirX, dirY)
	if hit.Cell == world.Target {
		s.Map.Set(hit.MapX, hit.MapY, world.Empty)
		s.Events = append(s.Events, EventTargetDestroyed)
	}
}
