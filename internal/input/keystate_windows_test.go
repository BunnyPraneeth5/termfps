package input

import "testing"

func TestAsyncSupportedOnWindows(t *testing.T) {
	if err := AsyncSupported(); err != nil {
		t.Fatalf("AsyncSupported: %v", err)
	}
	AsyncInput() // must not panic; the result depends on the real keyboard
}
