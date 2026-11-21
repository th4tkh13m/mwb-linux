package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky-verma/mwb-linux/internal/config"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "mwb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "# my config\nhost = \"192.168.1.7\"\nkey = \"DefaultKey123456\"\nname = \"linux\"\ncapture_backend = \"portal\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestApply(t *testing.T) {
	dir := setupHome(t)
	restart, err := Apply(Result{Saved: true, Active: "Office", Profiles: []Profile{
		{Name: "Default", IsDefault: true},
		{Name: "Office", IsNew: true, CopyFrom: "Default", Dirty: true, Host: "10.1.2.3", Key: "OfficeKey123456",
			Port: "15100", RemoteWidth: "2560", RemoteHeight: "1440", Edge: "right", Clipboard: false,
			FileTransfer: true, CaptureBackend: "auto", Accel: "1.5", Inbound: "1.0"},
	}})
	if err != nil || !restart {
		t.Fatalf("restart=%v err=%v", restart, err)
	}
	if config.ActiveProfile() != "Office" {
		t.Fatalf("active = %s", config.ActiveProfile())
	}
	c, err := config.Load(filepath.Join(dir, "profiles", "Office.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "10.1.2.3" || c.Key != "OfficeKey123456" || c.RemoteWidth != 2560 || c.Edge != "right" || c.ClipboardEnabled() || c.AccelMultiplier != 1.5 || c.Name != "linux" {
		t.Fatalf("unexpected profile: %+v", c)
	}
	def, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	if !strings.Contains(string(def), "# my config") || !strings.Contains(string(def), "DefaultKey123456") {
		t.Fatalf("default profile was modified: %s", def)
	}

	// An unusable profile cannot be made active.
	if _, err := Apply(Result{Saved: true, Active: "Empty", Profiles: []Profile{{Name: "Empty", IsNew: true, Dirty: true}}}); err == nil {
		t.Fatal("profile without host/key must be rejected")
	}
	// Deleting the profile in use falls back to Default and restarts.
	restart, err = Apply(Result{Saved: true, Active: "Office", Deleted: []string{"Office"}})
	if err != nil || !restart || config.ActiveProfile() != config.DefaultProfile {
		t.Fatalf("delete active: restart=%v err=%v active=%s", restart, err, config.ActiveProfile())
	}
}

// TestWindowEndToEnd drives the real QML window offscreen.
func TestWindowEndToEnd(t *testing.T) {
	if _, err := qmlBinary(); err != nil {
		t.Skip(err)
	}
	dir := setupHome(t)
	t.Setenv("QT_QPA_PLATFORM", "offscreen")
	orig := qmlSource
	defer func() { qmlSource = orig }()
	src := strings.TrimRight(string(orig), "\n")
	driver := `
    Timer {
        interval: 500
        running: true
        onTriggered: {
            win.addProfile("Office", true)
            win.setField("host", "10.1.2.3")
            win.setField("key", "OfficeKey123456")
            win.setField("edge", "right")
            win.active = "Office"
            win.save()
        }
    }
}
`
	qmlSource = []byte(src[:strings.LastIndex(src, "}")] + driver)

	restart, err := Run()
	if err != nil {
		t.Fatal(err)
	}
	if !restart || config.ActiveProfile() != "Office" {
		t.Fatalf("restart=%v active=%s", restart, config.ActiveProfile())
	}
	c, err := config.Load(filepath.Join(dir, "profiles", "Office.toml"))
	if err != nil || c.Host != "10.1.2.3" || c.Edge != "right" || c.Key != "OfficeKey123456" || c.CaptureBackend != "portal" {
		t.Fatalf("profile = %+v err=%v", c, err)
	}
}
