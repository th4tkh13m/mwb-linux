// Package settings shows the profile and settings editor and applies its result.
package settings

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/lucky-verma/mwb-linux/internal/config"
)

//go:embed settings.qml
var qmlSource []byte

// Profile is one editable profile. Numeric fields are strings because the form
// keeps them as typed text; an empty value means "leave unset".
type Profile struct {
	Name           string `json:"name"`
	IsDefault      bool   `json:"isDefault"`
	IsNew          bool   `json:"isNew"`
	Dirty          bool   `json:"dirty"`
	CopyFrom       string `json:"copyFrom"`
	Host           string `json:"host"`
	Key            string `json:"key"`
	Machine        string `json:"machine"`
	Port           string `json:"port"`
	RemoteWidth    string `json:"remoteWidth"`
	RemoteHeight   string `json:"remoteHeight"`
	Edge           string `json:"edge"`
	Clipboard      bool   `json:"clipboard"`
	FileTransfer   bool   `json:"fileTransfer"`
	CaptureBackend string `json:"captureBackend"`
	Accel          string `json:"accel"`
	Inbound        string `json:"inbound"`
}

// Result is what the window writes on Save.
type Result struct {
	Saved    bool      `json:"saved"`
	Active   string    `json:"active"`
	Deleted  []string  `json:"deleted"`
	Profiles []Profile `json:"profiles"`
}

type form struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
}

// qmlBinary finds a Qt 6 QML runner.
func qmlBinary() (string, error) {
	for _, n := range []string{"qml6", "qml-qt6", "/usr/lib/qt6/bin/qml", "/usr/lib64/qt6/bin/qml"} {
		if p, err := exec.LookPath(n); err == nil {
			return p, nil
		}
	}
	return "", errors.New("qml6 not found (install qt6-declarative, or qml6-module-qtquick-controls on Debian/Ubuntu)")
}

func intStr(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func floatStr(v float64) string {
	if v <= 0 {
		return ""
	}
	return config.FormatFloat(v)
}

// loadProfile reads a profile file without requiring it to be complete.
func loadProfile(name string) (Profile, error) {
	path, err := config.ProfilePath(name)
	if err != nil {
		return Profile{}, err
	}
	var c config.Config
	if _, err := toml.DecodeFile(path, &c); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Profile{}, fmt.Errorf("read profile %q: %w", name, err)
	}
	p := Profile{
		Name: name, IsDefault: name == config.DefaultProfile,
		Host: c.Host, Key: c.Key, Machine: c.Name,
		Port: intStr(c.Port), RemoteWidth: intStr(c.RemoteWidth), RemoteHeight: intStr(c.RemoteHeight),
		Edge: c.Edge, Clipboard: c.ClipboardEnabled(), FileTransfer: c.FileTransferEnabled(),
		CaptureBackend: c.CaptureBackend, Accel: floatStr(c.AccelMultiplier), Inbound: floatStr(c.InboundMultiplier),
	}
	if p.Port == "" {
		p.Port = "15100"
	}
	if p.RemoteWidth == "" {
		p.RemoteWidth = "1920"
	}
	if p.RemoteHeight == "" {
		p.RemoteHeight = "1080"
	}
	if p.Edge == "" {
		p.Edge = "left"
	}
	if p.CaptureBackend == "" {
		p.CaptureBackend = "auto"
	}
	if p.Accel == "" {
		p.Accel = "2.0"
	}
	if p.Inbound == "" {
		p.Inbound = "1.0"
	}
	return p, nil
}

// Run shows the settings window and applies what the user saves. It reports
// whether the running client must restart to pick the change up.
func Run() (restart bool, err error) {
	bin, err := qmlBinary()
	if err != nil {
		return false, err
	}
	f := form{Active: config.ActiveProfile()}
	for _, n := range config.ListProfiles() {
		p, err := loadProfile(n)
		if err != nil {
			return false, err
		}
		f.Profiles = append(f.Profiles, p)
	}

	dir, err := os.MkdirTemp("", "mwb-settings-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)
	qml, in, out := filepath.Join(dir, "settings.qml"), filepath.Join(dir, "in.json"), filepath.Join(dir, "out.json")
	data, _ := json.Marshal(f)
	if err := os.WriteFile(qml, qmlSource, 0o600); err != nil {
		return false, err
	}
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return false, err
	}

	cmd := exec.Command(bin, qml, "--", in, out)
	cmd.Env = append(os.Environ(), "QML_XHR_ALLOW_FILE_READ=1", "QML_XHR_ALLOW_FILE_WRITE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("settings window failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	raw, err := os.ReadFile(out)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil // cancelled
	}
	if err != nil {
		return false, err
	}
	var res Result
	if err := json.Unmarshal(raw, &res); err != nil {
		return false, fmt.Errorf("settings result: %w", err)
	}
	return Apply(res)
}

func setStr(path, key, val string) error {
	if val == "" {
		return nil
	}
	return config.Update(path, key, config.QuoteString(val))
}

func setNum(path, key, val string, float bool) error {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	if float {
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || f <= 0 {
			return fmt.Errorf("%s: invalid number %q", key, val)
		}
		return config.Update(path, key, config.FormatFloat(f))
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return fmt.Errorf("%s: invalid number %q", key, val)
	}
	return config.Update(path, key, strconv.Itoa(n))
}

func writeProfile(p Profile) error {
	path, err := config.ProfilePath(p.Name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	machine := p.Machine
	if len(machine) > 15 {
		machine = machine[:15]
	}
	steps := []error{
		setStr(path, "host", strings.TrimSpace(p.Host)),
		setStr(path, "key", p.Key),
		setStr(path, "name", machine),
		setNum(path, "port", p.Port, false),
		setNum(path, "remote_width", p.RemoteWidth, false),
		setNum(path, "remote_height", p.RemoteHeight, false),
		setNum(path, "accel_multiplier", p.Accel, true),
		setNum(path, "inbound_multiplier", p.Inbound, true),
		config.Update(path, "clipboard", strconv.FormatBool(p.Clipboard)),
		config.Update(path, "file_transfer", strconv.FormatBool(p.FileTransfer)),
	}
	if p.Edge == "left" || p.Edge == "right" {
		steps = append(steps, setStr(path, "edge", p.Edge))
	}
	switch p.CaptureBackend {
	case "auto", "portal", "x11":
		steps = append(steps, setStr(path, "capture_backend", p.CaptureBackend))
	}
	return errors.Join(steps...)
}

// Apply writes a saved Result to the profile files and reports whether the
// running client must restart.
func Apply(res Result) (restart bool, err error) {
	if !res.Saved {
		return false, nil
	}
	before := config.ActiveProfile()
	deleted := map[string]bool{}
	for _, n := range res.Deleted {
		if err := config.DeleteProfile(n); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		deleted[n] = true
	}
	changed := map[string]bool{}
	for _, p := range res.Profiles {
		if p.IsNew {
			from := p.CopyFrom
			if err := config.CreateProfile(p.Name, from); err != nil {
				return false, err
			}
		}
		if p.IsNew || p.Dirty {
			if err := writeProfile(p); err != nil {
				return false, err
			}
			changed[p.Name] = true
		}
	}
	active := res.Active
	if deleted[active] {
		active = config.DefaultProfile
	}
	if _, err := config.Load(mustPath(active)); err != nil {
		return false, fmt.Errorf("profile %q cannot be used: %w", active, err)
	}
	if err := config.SetActiveProfile(active); err != nil {
		return false, err
	}
	return active != before || changed[active] || deleted[before], nil
}

func mustPath(name string) string {
	p, _ := config.ProfilePath(name)
	return p
}
