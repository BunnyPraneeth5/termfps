package app

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Guide 3D views (M18b, D51): each object stands in a small display room and
// the camera slowly circles it; a map page turns all the way round inside
// the map. Everything is drawn by the game's own renderer.

// viewKind says what a guide page shows in 3D.
type viewKind int

const (
	viewNone   viewKind = iota // text only
	viewObject                 // a sprite in the display room
	viewGun                    // a gun on the floor, and in hand, firing now and then
	viewAcid                   // the display room with an acid pool
	viewMap                    // the camera turning round inside a map, and its top-down picture
)

// guideView is a page's 3D view.
type guideView struct {
	kind viewKind
	// sprite is the object for viewObject, given its place and the time
	// since the page opened (for an explosion, which grows and starts over).
	sprite func(x, y, t float64) render.Sprite
	gun    game.GunKind // viewGun
	mapIdx int          // viewMap: index into App.maps
}

// Display room and camera numbers.
const (
	orbitPeriod   = 10.0 // seconds for the camera to go once round an object
	mapTurnPeriod = 12.0 // seconds for a full turn inside a map
	gunFireEvery  = 2.0  // seconds between shots in a weapon view
	viewMinHeight = 0.5  // small objects (pickups) are shown at least this tall
	orbitPerTall  = 1.6  // the camera circles this many times the object's height away...
	orbitMin      = 1.2  // ...but never closer than this (cells)
	boomLoop      = 1.2  // seconds between explosions in the explosion view
)

// guideRoom is the display room: 9x9 with stone walls; the object stands in
// its middle. guideAcidRoom has an acid pool there instead.
const (
	guideRoom = "%%%%%%%%%\n%P......%\n%.......%\n%.......%\n%.......%\n%.......%\n%.......%\n%.......%\n%%%%%%%%%\n"

	guideAcidRoom = "%%%%%%%%%\n%P......%\n%.......%\n%..~~~..%\n%..~~~..%\n%..~~~..%\n%.......%\n%.......%\n%%%%%%%%%\n"

	roomCenter = 4.5
)

// spriteView shows s in the display room.
func spriteView(s render.Sprite) guideView {
	return guideView{kind: viewObject, sprite: func(x, y, _ float64) render.Sprite {
		s.X, s.Y = x, y
		return s
	}}
}

// boomView shows an explosion going off again and again.
func boomView() guideView {
	return guideView{kind: viewObject, sprite: func(x, y, t float64) render.Sprite {
		return render.BoomSprite(game.Boom{X: x, Y: y, Age: math.Mod(t, boomLoop) * game.BoomTime / boomLoop})
	}}
}

// orbit returns the camera for the display room at time t: on a circle of
// radius r round the middle, facing it, once round every orbitPeriod.
func orbit(t, r float64) (x, y, angle float64) {
	a := 2 * math.Pi * math.Mod(t, orbitPeriod) / orbitPeriod
	return roomCenter + r*math.Cos(a), roomCenter + r*math.Sin(a), math.Mod(a+math.Pi, 2*math.Pi)
}

// orbitRadius is how far from an object height cells tall the camera
// circles, so it fills most of the view box.
func orbitRadius(height float64) float64 {
	return max(orbitPerTall*height, orbitMin)
}

// newPreview makes the game state a view draws: the display room, or the
// map for a map view. It has no enemies and spawns nothing.
func (a *App) newPreview(v guideView) *game.State {
	var m *world.Map
	switch v.kind {
	case viewMap:
		m = a.maps[v.mapIdx] // only drawn, never changed
	case viewAcid:
		m = mustRoom(guideAcidRoom)
	default:
		m = mustRoom(guideRoom)
	}
	st := game.NewState(m)
	st.Spawning = false
	st.Enemies = nil
	if v.kind == viewGun {
		st.Player.Guns[v.gun] = game.Weapon{Kind: v.gun, Owned: true, Mag: v.gun.Stats().Mag, Reserve: 0}
		st.Player.Gun = v.gun
	}
	return st
}

// mustRoom parses a built-in display room; a broken one is a bug.
func mustRoom(layout string) *world.Map {
	m, err := world.Parse([]byte(layout))
	if err != nil {
		panic("guide room: " + err.Error())
	}
	return m
}

// drawView draws view v at time t (seconds since the page opened, dt since
// the last frame) into f, using the preview state st.
func (a *App) drawView(f *render.Frame, st *game.State, v guideView, t, dt float64) {
	f.PixelAspect, f.Textures = a.frame.PixelAspect, a.frame.Textures
	p := &st.Player
	var sprites []render.Sprite
	switch v.kind {
	case viewMap:
		p.X, p.Y = st.Map.SpawnX, st.Map.SpawnY
		p.Angle = 2 * math.Pi * math.Mod(t, mapTurnPeriod) / mapTurnPeriod
		for _, pr := range st.Map.Props {
			sprites = append(sprites, render.PropSprite(pr.Kind, pr.X, pr.Y))
		}
	case viewAcid:
		p.X, p.Y, p.Angle = orbit(t, 3)
	case viewGun:
		onFloor := game.Pickup{X: roomCenter, Y: roomCenter, Kind: game.GunPickup, Gun: v.gun}
		if v.gun == game.Pistol { // never lies on the floor: show its ammo
			onFloor.Kind = game.AmmoBox
		}
		s := render.PickupSprite(onFloor)
		s.Height = max(s.Height, viewMinHeight)
		p.X, p.Y, p.Angle = orbit(t, orbitRadius(s.Height))
		sprites = append(sprites, s)
	case viewObject:
		s := v.sprite(roomCenter, roomCenter, t)
		s.Height = max(s.Height, viewMinHeight)
		p.X, p.Y, p.Angle = orbit(t, orbitRadius(s.Height))
		sprites = append(sprites, s)
	}
	if v.kind == viewGun {
		// The gun in hand fires every gunFireEvery seconds: a real shot in
		// the room (into the wall), so the flash and recoil are the game's.
		fire := math.Floor(t/gunFireEvery) > math.Floor((t-dt)/gunFireEvery)
		p.Weapon().Mag = v.gun.Stats().Mag
		game.Update(st, game.Input{Fire: fire}, dt)
	} else {
		st.Time += dt // lamps flicker on a map
	}
	render.DrawFrame(st, f)
	render.DrawSprites(f, st.Player, sprites)
	if v.kind == viewGun {
		render.DrawWeapon(f, st.Player)
	}
}
