package game

import "testing"

// giveAll gives the player every gun, full.
func giveAll(s *State) {
	for k := range NumGuns {
		s.Player.Guns[k] = newGun(k)
	}
}

// You start with only the pistol in hand.
func TestStartGuns(t *testing.T) {
	s := newOpenState(t)
	g := s.Player.Guns
	if s.Player.Gun != Pistol || !g[Pistol].Owned || g[Shotgun].Owned || g[Rifle].Owned {
		t.Errorf("start: gun %d, owned %v %v %v; want the pistol only", s.Player.Gun, g[Pistol].Owned, g[Shotgun].Owned, g[Rifle].Owned)
	}
}

// Number keys pick owned guns only; the new gun waits SwitchTime to fire.
func TestSelectGun(t *testing.T) {
	s := newOpenState(t)
	Update(s, Input{SelectGun: 2}, frameDT)
	if s.Player.Gun != Pistol || count(s, EventGunSwitched) != 0 {
		t.Fatal("switched to a shotgun the player doesn't have")
	}
	giveAll(s)
	Update(s, Input{SelectGun: 2, Fire: true}, frameDT)
	if s.Player.Gun != Shotgun || count(s, EventGunSwitched) != 1 || count(s, EventShotFired) != 0 {
		t.Fatalf("after 2: gun %d, fired %d; want the shotgun, not firing yet", s.Player.Gun, count(s, EventShotFired))
	}
	if shots := run(s, Input{Fire: true}, 11, EventShotFired); shots != 1 {
		t.Errorf("%d shots in the 11 frames after switching, want 1 (the wait is %.2f s)", shots, SwitchTime)
	}
	Update(s, Input{SelectGun: 2}, frameDT)
	if count(s, EventGunSwitched) != 0 {
		t.Error("picking the gun already in hand switched again")
	}
}

// The wheel moves through owned guns and wraps around, skipping missing ones.
func TestCycleGun(t *testing.T) {
	s := newOpenState(t)
	s.Player.Guns[Rifle] = newGun(Rifle) // pistol and rifle, no shotgun
	Update(s, Input{CycleGun: 1}, frameDT)
	if s.Player.Gun != Rifle {
		t.Fatalf("next gun from the pistol = %d, want the rifle (no shotgun)", s.Player.Gun)
	}
	Update(s, Input{CycleGun: 1}, frameDT)
	if s.Player.Gun != Pistol {
		t.Fatalf("next gun from the rifle = %d, want the pistol (wraps)", s.Player.Gun)
	}
	Update(s, Input{CycleGun: -1}, frameDT)
	if s.Player.Gun != Rifle {
		t.Errorf("previous gun from the pistol = %d, want the rifle", s.Player.Gun)
	}
}

// Switching away cancels a reload; each gun keeps its own ammo.
func TestSwitchCancelsReload(t *testing.T) {
	s := newOpenState(t)
	giveAll(s)
	s.Player.Weapon().Mag = 3
	Update(s, Input{Reload: true}, frameDT)
	Update(s, Input{SelectGun: 3}, frameDT)
	Update(s, Input{SelectGun: 1}, frameDT)
	if w := s.Player.Weapon(); w.Reloading() || w.Mag != 3 {
		t.Errorf("pistol after switching away mid-reload: mag %d, reloading %v; want 3 and false", w.Mag, w.Reloading())
	}
	if s.Player.Guns[Rifle].Mag != Rifle.Stats().Mag {
		t.Error("the rifle's magazine changed while the pistol was used")
	}
}

// One shotgun shot up close puts every pellet into a zombie; far away
// only some hit.
func TestShotgunSpread(t *testing.T) {
	damageAt := func(dist int) int {
		row := "#P" + repeat(".", dist-1) + "Z" + repeat(".", 3) + "#"
		wall := repeat("#", len(row))
		s := stateFrom(t, wall+"\n"+row+"\n"+wall+"\n")
		giveAll(s)
		s.Player.Gun = Shotgun
		s.Enemies[0].HP = 1000 // survives every pellet, so all can be counted
		Update(s, Input{Fire: true}, frameDT)
		return 1000 - s.Enemies[0].HP
	}
	st := Shotgun.Stats()
	if got := damageAt(2); got != st.Damage*st.Pellets {
		t.Errorf("shotgun at 2 cells did %d, want all %d pellets (%d)", got, st.Pellets, st.Damage*st.Pellets)
	}
	if far := damageAt(8); far >= st.Damage*st.Pellets || far == 0 {
		t.Errorf("shotgun at 8 cells did %d, want some pellets but not all", far)
	}
}

// The rifle kills a full-health zombie with one shot.
func TestRifleDamage(t *testing.T) {
	s := stateFrom(t, "##########\n#P.....Z.#\n##########\n")
	s.Player.Guns[Rifle] = newGun(Rifle)
	s.Player.Gun = Rifle
	Update(s, Input{Fire: true}, frameDT)
	if len(s.Enemies) != 0 || count(s, EventRifleFired) != 1 || count(s, EventShotFired) != 1 {
		t.Errorf("after one rifle shot: %d enemies left, want the zombie dead", len(s.Enemies))
	}
	if s.Player.Weapon().Mag != Rifle.Stats().Mag-1 {
		t.Errorf("rifle magazine %d, want %d", s.Player.Weapon().Mag, Rifle.Stats().Mag-1)
	}
}

// Walking over a gun gives it, full, and puts it in hand.
func TestTakeGun(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Kind: GunPickup, Gun: Shotgun}}
	Update(s, Input{}, frameDT)
	w := s.Player.Weapon()
	if s.Player.Gun != Shotgun || !w.Owned || w.Mag != Shotgun.Stats().Mag || w.Reserve != Shotgun.Stats().StartReserve {
		t.Errorf("after taking the shotgun: gun %d, %+v; want it in hand, full", s.Player.Gun, *w)
	}
	if len(s.Pickups) != 0 || count(s, EventGunTaken) != 1 {
		t.Error("gun pickup not taken")
	}
}

// Ammo goes to its own gun, and stays on the floor for a gun you don't have.
func TestAmmoPerGun(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	shells := Pickup{X: s.Player.X, Y: s.Player.Y, Kind: AmmoBox, Gun: Shotgun, Amount: Shotgun.Stats().BoxAmmo}
	s.Pickups = []Pickup{shells}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 1 {
		t.Fatal("shells taken without a shotgun")
	}
	s.Player.Guns[Shotgun] = newGun(Shotgun)
	Update(s, Input{}, frameDT)
	if got, want := s.Player.Guns[Shotgun].Reserve, Shotgun.Stats().StartReserve+Shotgun.Stats().BoxAmmo; got != want || len(s.Pickups) != 0 {
		t.Errorf("shotgun reserve %d after shells, want %d", got, want)
	}
	if s.Player.Guns[Pistol].Reserve != StartReserve {
		t.Error("shells changed the pistol's ammo")
	}
}

// The shotgun shows up as a pickup once its time has come, and only once.
func TestGunArrives(t *testing.T) {
	s := survivalState(t)
	if _, ok := gunDue(s); ok {
		t.Fatal("a gun is due at the start")
	}
	s.Time = Shotgun.Stats().Arrives
	if g, ok := gunDue(s); !ok || g != Shotgun {
		t.Fatalf("at %.0f s gunDue = %d, %v; want the shotgun", s.Time, g, ok)
	}
	s.Pickups = append(s.Pickups, newPickup(s, 1.5, 1.5))
	if _, ok := gunDue(s); ok {
		t.Error("shotgun due again while one lies on the map")
	}
	s.Time = Rifle.Stats().Arrives
	if g, ok := gunDue(s); !ok || g != Rifle {
		t.Errorf("at %.0f s gunDue = %d, %v; want the rifle", s.Time, g, ok)
	}
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}
