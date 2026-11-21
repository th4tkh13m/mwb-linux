//go:build linux

package capture

import (
	"os"
	"testing"
)

// TestMain hides the developer's real Wayland session so tests that call Run()
// in auto mode never reach a live compositor portal.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mwb-capture-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_RUNTIME_DIR", dir)
	os.Unsetenv("WAYLAND_DISPLAY")
	os.Unsetenv("DISPLAY")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
