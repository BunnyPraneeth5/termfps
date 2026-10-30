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
