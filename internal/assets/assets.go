// Package assets holds the game's pixel-art images as text files, built
// into the binary. The format is described in render.ParseImage.
package assets

import _ "embed" // the blank import (_) enables the //go:embed directives below

// Gun is the player's weapon, seen from behind (P1-04).
//
//go:embed gun.txt
var Gun []byte

// Flash is the muzzle flash shown right after a shot (P1-04).
//
//go:embed flash.txt
var Flash []byte

// Zombie is the green melee enemy (P1-05 images, P1-06 behavior).
//
//go:embed zombie.txt
var Zombie []byte

// Shooter is the red ranged enemy (P1-05 images, P1-07 behavior).
//
//go:embed shooter.txt
var Shooter []byte

// Ammo is the ammo box pickup (P1-09).
//
//go:embed ammo.txt
var Ammo []byte

// Brick, Stone and Metal are 16x16 wall textures for '#', '%' and '&'
// (P1-10).
//
//go:embed brick.txt
var Brick []byte

//go:embed stone.txt
var Stone []byte

//go:embed metal.txt
var Metal []byte
