package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Pistol numbers (D22). Balancing (P1-16, M18) may tune them. The shotgun
// and rifle (M11) are in gunStats.
const (
	MagazineSize = 25        // rounds per magazine
	StartReserve = 50        // spare rounds at the start
	FireInterval = 1.0 / 3.0 // seconds between shots (3 per second)
	ReloadTime   = 1.5       // seconds a reload takes
	FlashTime    = 0.1       // seconds the muzzle flash (and recoil) shows
	SwitchTime   = 0.35      // seconds after switching guns before the new one fires (M11)
)

// GunKind tells the player's guns apart (M11). The number keys 1, 2, 3
// pick them in this order.
type GunKind int

const (
	Pistol  GunKind = iota // the gun you start with
	Shotgun                // several pellets in a spread: strong up close, slow
	Rifle                  // accurate, high damage, slow
	NumGuns                // not a gun: how many there are
)

// GunStats are one gun's numbers.
type GunStats struct {
	Name         string  // for the HUD
	Mag          int     // rounds per magazine
	Interval     float64 // seconds between shots
	Reload       float64 // seconds a reload takes
	Damage       int     // per pellet
	Pellets      int     // pellets per shot (1 = a single bullet)
	Spread       float64 // radians from the center to the outermost pellet
	MaxReserve   int     // spare rounds cap
	BoxAmmo      int     // spare rounds an ammo box gives
	StartReserve int     // spare rounds that come with the gun
	// Arrives is how many seconds into a run the gun can first appear as a
	// pickup; the pistol is in hand from the start.
	Arrives float64
}

// gunStats holds every gun's numbers, so balancing is one place.
var gunStats = [NumGuns]GunStats{
	Pistol: {Name: "PISTOL", Mag: MagazineSize, Interval: FireInterval, Reload: ReloadTime,
		Damage: BulletDamage, Pellets: 1, MaxReserve: MaxReserve, BoxAmmo: PickupAmmo, StartReserve: StartReserve},
	Shotgun: {Name: "SHOTGUN", Mag: 6, Interval: 0.8, Reload: 2.0,
		Damage: 12, Pellets: 8, Spread: 0.1, MaxReserve: 30, BoxAmmo: 6, StartReserve: 12, Arrives: 30},
	Rifle: {Name: "RIFLE", Mag: 5, Interval: 1.0, Reload: 2.0,
		Damage: 100, Pellets: 1, MaxReserve: 25, BoxAmmo: 5, StartReserve: 10, Arrives: 90},
}

// Stats returns the gun's numbers.
func (k GunKind) Stats() GunStats {
	return gunStats[k]
}

// Weapon is one of the player's guns: a magazine, spare rounds, and timers.
type Weapon struct {
	Kind    GunKind
	Owned   bool // the player has found it (the pistol from the start)
	Mag     int  // rounds in the gun
	Reserve int  // spare rounds

	cooldown  float64 // seconds until the next shot is allowed
	reloading float64 // seconds left in the current reload; 0 = not reloading
	flash     float64 // seconds left of the muzzle flash; 0 = none
}

// NewWeapon returns a full pistol with the starting spare rounds.
func NewWeapon() Weapon {
	return newGun(Pistol)
}

// newGun returns gun k as it comes when found: full, with its spare rounds.
func newGun(k GunKind) Weapon {
	st := k.Stats()
	return Weapon{Kind: k, Owned: true, Mag: st.Mag, Reserve: st.StartReserve}
}

// StartGuns returns the guns at the start of a run: the pistol in hand,
// the others not found yet.
func StartGuns() [NumGuns]Weapon {
	var g [NumGuns]Weapon
	for k := range g {
		g[k].Kind = GunKind(k)
	}
	g[Pistol] = NewWeapon()
	return g
}

// Reloading reports whether a reload is in progress.
func (w Weapon) Reloading() bool {
	return w.reloading > 0
}

// Flashing reports whether the muzzle flash shows (right after a shot).
func (w Weapon) Flashing() bool {
	return w.flash > 0
}

// switchGun puts gun k in the player's hands, if they have it and aren't
// holding it already. The old gun's reload is lost; the new one needs
// SwitchTime before it fires.
func switchGun(s *State, k GunKind) {
	p := &s.Player
	if k < 0 || k >= NumGuns || k == p.Gun || !p.Guns[k].Owned {
		return
	}
	old := p.Weapon()
	old.reloading, old.flash = 0, 0
	p.Gun = k
	w := p.Weapon()
	w.cooldown = max(w.cooldown, SwitchTime)
	s.Events = append(s.Events, EventGunSwitched)
}

// cycleGun switches to the next owned gun up (dir = 1) or down (-1) the
// list, wrapping around.
func cycleGun(s *State, dir int) {
	for i := 1; i < int(NumGuns); i++ {
		k := GunKind((int(s.Player.Gun) + dir*i + 2*int(NumGuns)) % int(NumGuns))
		if s.Player.Guns[k].Owned {
			switchGun(s, k)
			return
		}
	}
}

// updateWeapon handles switching guns, advances the gun's timers, and
// handles the fire and reload inputs for one frame. Shots are hitscan
// along the view direction.
func updateWeapon(s *State, in Input, dt float64) {
	if in.SelectGun > 0 {
		switchGun(s, GunKind(in.SelectGun-1))
	}
	if in.CycleGun != 0 {
		cycleGun(s, in.CycleGun)
	}
	w := s.Player.Weapon() // a pointer: changes below modify the gun in hand
	st := w.Kind.Stats()

	w.cooldown = max(w.cooldown-dt, 0)
	w.flash = max(w.flash-dt, 0)
	if w.reloading > 0 {
		w.reloading -= dt
		if w.reloading <= 0 {
			w.reloading = 0
			take := min(st.Mag-w.Mag, w.Reserve)
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
	w.cooldown = st.Interval
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
	switch w.Kind {
	case Shotgun:
		s.Events = append(s.Events, EventShotgunFired)
	case Rifle:
		s.Events = append(s.Events, EventRifleFired)
	}
	shoot(s, st)
	if w.Mag == 0 {
		startReload(s) // auto-reload (does nothing without spare rounds)
	}
}

// startReload begins a reload if one isn't running, the magazine isn't
// full, and there are spare rounds.
func startReload(s *State) {
	w := s.Player.Weapon()
	if w.Reloading() || w.Mag == w.Kind.Stats().Mag || w.Reserve == 0 {
		return
	}
	w.reloading = w.Kind.Stats().Reload
	s.Events = append(s.Events, EventReloadStarted)
}

// shoot fires one shot of a gun with stats st: one bullet along the view
// direction, or for the shotgun several pellets spread evenly across
// st.Spread on each side, each nudged a little at random.
func shoot(s *State, st GunStats) {
	if st.Pellets <= 1 {
		shootRay(s, s.Player.Angle, st.Damage)
		return
	}
	step := 2 * st.Spread / float64(st.Pellets-1)
	for i := range st.Pellets {
		jitter := (s.Rand.Float64() - 0.5) * step / 2
		shootRay(s, s.Player.Angle-st.Spread+float64(i)*step+jitter, st.Damage)
	}
}

// shootRay casts one bullet from the player at angle. The nearest enemy in
// front of the first solid cell (or prop that blocks shots, M2) takes the
// damage; otherwise a red barrel there blows up (M14), or a red target
// block there is destroyed.
func shootRay(s *State, angle float64, damage int) {
	p := s.Player
	dirX, dirY := math.Cos(angle), math.Sin(angle)
	hit := world.CastRay(s.Map, p.X, p.Y, dirX, dirY) // unit dir: PerpDist = distance
	cover, cx, cy := world.ShotBlocker(s.Map, p.X, p.Y, dirX, dirY, hit.PerpDist)
	if i := shotHitsEnemy(s, dirX, dirY, math.Min(hit.PerpDist, cover)); i >= 0 {
		markHit(s, damageEnemy(s, i, damage)) // M16
		return
	}
	if cx >= 0 && s.Map.At(cx, cy) == world.ExplosiveBarrel {
		blowBarrel(s, cx, cy, 0) // M14
		return
	}
	if hit.Cell == world.Target && cover > hit.PerpDist {
		s.Map.Set(hit.MapX, hit.MapY, world.Empty)
		s.distCell = -1 // the map opened up: enemies need a fresh distance field
		s.Events = append(s.Events, EventTargetDestroyed)
	}
}
