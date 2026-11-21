//go:build !linux

package tray

type State struct {
	Edge                    string
	Clipboard, FileTransfer bool
}
type Actions struct {
	Set                          func(key, raw string)
	ShowScreen, OpenConfig, Quit func()
}

// Tray is a no-op on platforms without StatusNotifierItem support.
type Tray struct{}

func Start(host string, getState func() State, actions *Actions) (*Tray, error) { return nil, nil }
func (t *Tray) SetConnected(bool)                                               {}
func (t *Tray) Close()                                                          {}
