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
