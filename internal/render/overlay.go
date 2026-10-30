package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// crosshairArm is how many pixels each arm of the crosshair reaches out
// from the center pixel.
const crosshairArm = 2

// DrawCrosshair draws a small "+" at the center of the Frame, where shots
// go. Each pixel is the opposite of what's under it (white on dark, black
// on bright), so it stays visible on any wall or floor. Call it after
// DrawFrame; it is part of the Frame, so every render mode shows it.
func DrawCrosshair(f *Frame) {
	if f.W <= 0 || f.H <= 0 {
		return
	}
	cx, cy := f.W/2, f.H/2
	invert(f, cx, cy)
	for d := 1; d <= crosshairArm; d++ {
		invert(f, cx-d, cy)
		invert(f, cx+d, cy)
		invert(f, cx, cy-d)
		invert(f, cx, cy+d)
	}
}

// invert sets the pixel at (x, y) to white or black, whichever stands out
// against its current gray. Pixels outside the Frame are skipped.
func invert(f *Frame, x, y int) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	p := &f.Pix[y*f.W+x]
	if p.Gray < 128 {
		*p = Pixel{Gray: 255}
	} else {
		*p = Pixel{Gray: 0}
	}
}

// hurtBorder is how many pixels wide the red "you got hit" border is.
const hurtBorder = 2

// DrawHurt tints the edges of the Frame red while the player is hurting.
// Call it last, after the weapon and crosshair.
func DrawHurt(f *Frame, p game.Player) {
	if !p.Hurting() {
		return
	}
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			if x >= hurtBorder && x < f.W-hurtBorder && y >= hurtBorder && y < f.H-hurtBorder {
				continue
			}
			px := &f.Pix[y*f.W+x]
			*px = Pixel{Gray: max(px.Gray, 160), Tint: TintRed}
		}
	}
}

// The guns in hand (M11: one per GunKind) and the muzzle flash, parsed
// from the built-in assets at startup.
var (
	gunImages = [game.NumGuns]*Image{
		game.Pistol:  mustParseImage("gun", assets.Gun),
		game.Shotgun: mustParseImage("shotgun", assets.Shotgun),
		game.Rifle:   mustParseImage("rifle", assets.Rifle),
	}
	flashImage = mustParseImage("flash", assets.Flash)
)

const (
	// gunHeight is the gun's height as a share of the view height.
	gunHeight = 0.38
	// recoilShare: on a shot the gun kicks down by this share of its height.
	recoilShare = 0.08
	// bobSway and bobLift: how far the gun moves sideways and up/down while
	// walking, as shares of its size.
	bobSway = 0.06
	bobLift = 0.05
)

// DrawWeapon draws the gun in hand at the bottom center of the Frame, swaying with
// the walking bob, kicked down while the muzzle flash shows, with the flash
// above the barrel. Call it after the 3D view and sprites, before the
// crosshair. Being part of the Frame, it shows in every render mode.
func DrawWeapon(f *Frame, p game.Player) {
	if f.W <= 0 || f.H <= 0 {
		return
	}
	gunImage := gunImages[p.Weapon().Kind]
	dh := max(int(math.Round(float64(f.H)*gunHeight)), 1)
	dw := max(dh*gunImage.W/gunImage.H, 1) // pixels are square, keep the shape

	x0 := f.W/2 - dw/2 + int(math.Round(math.Sin(p.Bob)*bobSway*float64(dw))) // 0 at rest
	y0 := f.H - dh + int(math.Round(math.Abs(math.Sin(p.Bob))*bobLift*float64(dh)))
	if p.Weapon().Flashing() {
		y0 += int(math.Round(recoilShare * float64(dh)))
	}
	drawImage(f, gunImage, x0, y0, dw, dh)

	if p.Weapon().Flashing() {
		fh := max(dh/3, 3)
		fw := fh * flashImage.W / flashImage.H
		// Centered on the barrel tip, overlapping it a little.
		drawImage(f, flashImage, x0+dw/2-fw/2, y0-fh+fh/4, fw, fh)
	}
}

// shakePixels is how far a full-strength shake (M16) pushes the view, as a
// share of the Frame height.
const shakePixels = 0.035

// ShakeView moves the whole Frame by the player's shake push (M16), filling
// the uncovered edge with the edge pixels. Call it after the sprites and
// before the gun, so the world shakes and the gun stays put.
func ShakeView(f *Frame, p game.Player) {
	sx, sy := p.Shake()
	dx := int(math.Round(sx * shakePixels * float64(f.H)))
	dy := int(math.Round(sy * shakePixels * float64(f.H)))
	if f.W <= 0 || f.H <= 0 || (dx == 0 && dy == 0) {
		return
	}
	dx = min(max(dx, -f.W+1), f.W-1)
	dy = min(max(dy, -f.H+1), f.H-1)
	// Rows: moving down, fill from the bottom up so each row is copied
	// before it is overwritten (and the other way when moving up).
	row := func(y int) []Pixel { return f.Pix[y*f.W : (y+1)*f.W] }
	if dy > 0 {
		for y := f.H - 1; y >= 0; y-- {
			copy(row(y), row(max(y-dy, 0)))
		}
	} else if dy < 0 {
		for y := 0; y < f.H; y++ {
			copy(row(y), row(min(y-dy, f.H-1)))
		}
	}
	// Columns: copy handles overlapping slices, like C's memmove.
	for y := 0; y < f.H; y++ {
		r := row(y)
		if dx > 0 {
			copy(r[dx:], r[:f.W-dx])
			for x := 0; x < dx; x++ {
				r[x] = r[dx]
			}
		} else if dx < 0 {
			copy(r[:f.W+dx], r[-dx:])
			for x := f.W + dx; x < f.W; x++ {
				r[x] = r[f.W+dx-1]
			}
		}
	}
}

// DrawHitMarker draws four short diagonal ticks around the crosshair while
// the player's shot just hit an enemy (M16): white for a hit, red for a
// kill. Call it after DrawCrosshair.
func DrawHitMarker(f *Frame, p game.Player) {
	on, kill := p.HitMarker()
	if !on || f.W <= 0 || f.H <= 0 {
		return
	}
	px := Pixel{Gray: 255}
	if kill {
		px = Pixel{Gray: 230, Tint: TintRed}
	}
	cx, cy := f.W/2, f.H/2
	for d := crosshairArm + 1; d <= crosshairArm+2; d++ {
		for _, s := range [4][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			x, y := cx+s[0]*d, cy+s[1]*d
			if x >= 0 && y >= 0 && x < f.W && y < f.H {
				f.Pix[y*f.W+x] = px
			}
		}
	}
}

// Hit direction marker (M16): a red arc around the crosshair on the side
// the last hit came from (straight up = in front, down = behind).
const (
	hitArcRadius = 0.28 // share of the Frame height
	hitArcWidth  = 0.5  // radians the arc spans
	hitArcThick  = 2    // pixels
)

// DrawHitDirection draws the hit direction marker, fading as it ages.
// Call it after the gun and crosshair.
func DrawHitDirection(f *Frame, p game.Player) {
	angle, strength, ok := p.HitDirection()
	if !ok || f.W <= 0 || f.H <= 0 {
		return
	}
	rel := angle - p.Angle // 0 = straight ahead, growing clockwise like Angle
	gray := toGray(120 + 135*strength)
	cx, cy := float64(f.W)/2, float64(f.H)/2
	r0 := hitArcRadius * float64(f.H)
	steps := max(int(r0*hitArcWidth*2), 8) // enough points that the arc has no gaps
	for i := 0; i <= steps; i++ {
		a := rel - hitArcWidth/2 + hitArcWidth*float64(i)/float64(steps)
		for t := 0; t < hitArcThick; t++ {
			r := r0 + float64(t)
			// On screen, ahead is up (-y) and clockwise is to the right.
			x := int(math.Round(cx + r*math.Sin(a)))
			y := int(math.Round(cy - r*math.Cos(a)))
			if x >= 0 && y >= 0 && x < f.W && y < f.H {
				f.Pix[y*f.W+x] = Pixel{Gray: gray, Tint: TintRed}
			}
		}
	}
}
