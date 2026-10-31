package app

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
)

// Every enemy, weapon, pickup, hazard and open map has a 3D view; text
// pages and locked maps don't (M18b).
func TestGuideViews(t *testing.T) {
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0]) // the second one is locked
	for _, name := range []string{"Enemies", "Weapons", "Pickups", "Hazards"} {
		for _, e := range section(t, a, name).entries {
			if e.view.kind == viewNone {
				t.Errorf("%s: %s has no 3D view", name, e.name)
			}
		}
	}
	maps := section(t, a, "Maps").entries
	if maps[0].view.kind != viewMap || maps[1].view.kind != viewNone {
		t.Errorf("map views: open %d, locked %d; want a map view and none", maps[0].view.kind, maps[1].view.kind)
	}
	for _, name := range []string{"Controls and HUD", "Survival and waves", "Settings", "Tips"} {
		for _, e := range section(t, a, name).entries {
			if e.view.kind != viewNone {
				t.Errorf("text page %s has a view", e.name)
			}
		}
	}
}

// The camera goes all the way round the object, always facing it.
func TestOrbit(t *testing.T) {
	const r = 2
	x0, y0, _ := orbit(0, r)
	xh, yh, _ := orbit(orbitPeriod/2, r)
	xf, yf, _ := orbit(orbitPeriod, r)
	if math.Abs(x0+xh-2*roomCenter) > 1e-9 || math.Abs(y0+yh-2*roomCenter) > 1e-9 {
		t.Errorf("half way round: (%.2f, %.2f) and (%.2f, %.2f) aren't opposite", x0, y0, xh, yh)
	}
	if math.Abs(xf-x0) > 1e-9 || math.Abs(yf-y0) > 1e-9 {
		t.Error("a full period doesn't come back to the start")
	}
	for _, tt := range []float64{0, 1.3, 4, 7.7} {
		x, y, a := orbit(tt, r)
		fx, fy := x+math.Cos(a)*r, y+math.Sin(a)*r
		if math.Hypot(fx-roomCenter, fy-roomCenter) > 1e-9 {
			t.Errorf("at %.1f s the camera faces (%.2f, %.2f), not the middle", tt, fx, fy)
		}
	}
}

// A view draws the object, and the picture changes as the camera moves.
func TestViewDrawsAndMoves(t *testing.T) {
	a, _ := newMenuApp(t)
	v := spriteView(render.EnemySprite(game.Enemy{Kind: game.KindRunner}))
	st := a.newPreview(v)
	var f render.Frame
	f.Resize(guidePicCols, 2*guidePicRows)
	a.drawView(&f, st, v, 0, 0)
	blue, first := 0, append([]render.Pixel(nil), f.Pix...)
	for _, p := range f.Pix {
		if p.Tint == render.TintBlue {
			blue++
		}
	}
	if blue == 0 {
		t.Fatal("the runner view shows no blue runner")
	}
	a.drawView(&f, st, v, orbitPeriod/4, orbitPeriod/4)
	same := 0
	for i := range f.Pix {
		if f.Pix[i] == first[i] {
			same++
		}
	}
	if same == len(f.Pix) {
		t.Error("the picture didn't change a quarter of the way round")
	}
}

// A map view turns round at the map's start.
func TestMapViewTurns(t *testing.T) {
	a, _ := newMenuApp(t)
	v := guideView{kind: viewMap}
	st := a.newPreview(v)
	var f render.Frame
	f.Resize(guidePicCols, 2*guidePicRows)
	a.drawView(&f, st, v, mapTurnPeriod/4, 0)
	if p := st.Player; p.X != a.maps[0].SpawnX || math.Abs(p.Angle-math.Pi/2) > 1e-9 {
		t.Errorf("a quarter turn in: camera at (%.1f, %.1f) facing %.2f, want the start facing pi/2", p.X, p.Y, p.Angle)
	}
}

// The gun in a weapon view fires every gunFireEvery seconds.
func TestGunViewFires(t *testing.T) {
	a, _ := newMenuApp(t)
	v := guideView{kind: viewGun, gun: game.Rifle}
	st := a.newPreview(v)
	var f render.Frame
	f.Resize(guidePicCols, 2*guidePicRows)
	shots := 0
	const dt = 1.0 / 30
	for i := 1; i <= int(2*gunFireEvery*30)+1; i++ {
		a.drawView(&f, st, v, float64(i)*dt, dt)
		if st.HasEvent(game.EventShotFired) {
			shots++
		}
	}
	if shots != 2 || st.Player.Gun != game.Rifle {
		t.Errorf("%d shots in %.0f s with the %s, want 2", shots, 2*gunFireEvery, st.Player.Weapon().Kind.Stats().Name)
	}
}
