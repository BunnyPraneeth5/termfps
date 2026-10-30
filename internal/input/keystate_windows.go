package input

// T8 spike: read real key state with the Win32 GetAsyncKeyState function.
// Go only compiles files ending in _windows.go on Windows; keystate_other.go
// is used everywhere else.

import (
	"syscall"

	"github.com/BunnyPraneeth5/termfps/internal/game"
)

var (
	// Lazy: user32.dll is loaded the first time the function is used.
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

// Windows virtual-key codes. Letters use their uppercase ASCII value.
const (
	vkSpace = 0x20
	vkLeft  = 0x25
	vkUp    = 0x26
	vkRight = 0x27
	vkDown  = 0x28
)

// AsyncSupported returns an error if real key state can't be read here.
func AsyncSupported() error {
	return procGetAsyncKeyState.Find()
}

// AsyncInput reads which movement, sprint, fire and reload keys are physically
// down right now.
// It sees keys pressed in any window, so the caller must ignore it while
// the terminal doesn't have focus.
func AsyncInput() game.Input {
	return game.Input{
		Forward:     down('W') || down(vkUp),
		Back:        down('S') || down(vkDown),
		StrafeLeft:  down('A'),
		StrafeRight: down('D'),
		TurnLeft:    down('Q') || down(vkLeft),
		TurnRight:   down('E') || down(vkRight),
		Sprint:      down('F'), // M16
		Fire:        down(vkSpace),
		Reload:      down('R'),
	}
}

// down reports whether a key is held. The top bit (0x8000) of the result
// means "down right now".
func down(vk int) bool {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return uint16(r)&0x8000 != 0
}
