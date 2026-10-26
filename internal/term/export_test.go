package term

import "github.com/gdamore/tcell/v3"

// This file is only compiled for tests. Its exported names are visible to
// the external test package term_test (bytes_test.go), which can't see
// unexported ones.

// OpenTty opens the real tcell screen on a custom terminal connection:
// truecolor, xterm, and no startup questions to the terminal, so the bytes
// it writes don't depend on the machine.
func OpenTty(tt tcell.Tty) (*Tcell, error) {
	s, err := tcell.NewTerminfoScreenFromTty(tt,
		tcell.OptColors(1<<24), tcell.OptTerm("xterm-256color"), tcell.OptNegotiation(false))
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &Tcell{s: s}, nil
}
