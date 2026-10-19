//go:build !windows

// The line above is a build constraint: this file is compiled everywhere
// except Windows, where keystate_windows.go is used instead.

package input

import (
	"errors"

	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// AsyncSupported returns an error: real key state is Windows-only (T8 spike).
func AsyncSupported() error {
	return errors.New("-keys async only works on Windows")
}

// AsyncInput is never called when AsyncSupported fails.
func AsyncInput() game.Input {
	return game.Input{}
}
