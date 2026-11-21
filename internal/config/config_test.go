// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "192.168.1.50"
key = "MySecurityKey1234"
name = "testbox"
port = 15100
`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "192.168.1.50" {
		t.Errorf("host = %q, want %q", cfg.Host, "192.168.1.50")
	}
	if cfg.Key != "MySecurityKey1234" {
		t.Errorf("key = %q", cfg.Key)
	}
	if cfg.Name != "testbox" {
		t.Errorf("name = %q", cfg.Name)
	}
	if cfg.Port != 15100 {
		t.Errorf("port = %d, want 15100", cfg.Port)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 15100 {
		t.Errorf("default port = %d, want 15100", cfg.Port)
	}
	if cfg.Name == "" {
		t.Error("name should default to hostname")
	}
	if cfg.AccelMultiplier != 2.0 {
		t.Errorf("default accel_multiplier = %v, want 2.0", cfg.AccelMultiplier)
	}
	if cfg.InboundMultiplier != 1.0 {
		t.Errorf("default inbound_multiplier = %v, want 1.0", cfg.InboundMultiplier)
	}
	if cfg.KeyboardLayout != "auto" {
		t.Errorf("default keyboard_layout = %q, want auto", cfg.KeyboardLayout)
	}
	if cfg.CaptureBackend != "auto" {
		t.Errorf("default capture_backend = %q, want auto", cfg.CaptureBackend)
	}
}

func TestAccelMultiplierExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
accel_multiplier = 0.5
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccelMultiplier != 0.5 {
		t.Errorf("accel_multiplier = %v, want 0.5", cfg.AccelMultiplier)
	}
}

func TestKeyboardLayoutExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
keyboard_layout = "de"
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.KeyboardLayout != "de" {
		t.Errorf("keyboard_layout = %q, want de", cfg.KeyboardLayout)
	}
}

func TestCaptureBackendExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
capture_backend = "portal"
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CaptureBackend != "portal" {
		t.Errorf("capture_backend = %q, want portal", cfg.CaptureBackend)
	}
}

func TestCaptureBackendRejectsUnknownValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
capture_backend = "wayland"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unknown capture_backend should be rejected")
	}
}

func TestClipboardEnabledDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ClipboardEnabled() {
		t.Error("clipboard should default to enabled when the key is absent")
	}
}

func TestClipboardDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
clipboard = false
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClipboardEnabled() {
		t.Error("clipboard = false should disable clipboard sharing")
	}
}

func TestClipboardEnabledExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
clipboard = true
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ClipboardEnabled() {
		t.Error("clipboard = true should enable clipboard sharing")
	}
}

func TestLoadConfigValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Missing host
	if err := os.WriteFile(path, []byte(`key = "SomeKeyHere!1234"`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("should fail without host")
	}

	// Missing key
	if err := os.WriteFile(path, []byte(`host = "10.0.0.1"`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("should fail without key")
	}
}

func TestLoadTightensConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
`), 0664); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0664); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config permissions = %04o, want 0600", got)
	}
}

func TestLegacyCryptoDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LegacyCryptoEnabled() {
		t.Error("an absent legacy_crypto should mean the current PowerToys scheme")
	}
}

func TestLegacyCryptoEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
host = "10.0.0.1"
key = "SomeKeyHere!1234"
legacy_crypto = true
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LegacyCryptoEnabled() {
		t.Error("legacy_crypto = true should select the pre-0.101 scheme")
	}
}

func TestUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("host = \"h\"\nkey = \"k\"\nedge = \"left\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(path, "edge", `"right"`); err != nil {
		t.Fatal(err)
	}
	if err := Update(path, "clipboard", "false"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Edge != "right" || cfg.ClipboardEnabled() || cfg.Key != "k" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode())
	}
}

func TestProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "mwb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("host = \"h\"\nkey = \"k\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ActiveProfile(); got != DefaultProfile {
		t.Fatalf("active = %q", got)
	}
	if err := CreateProfile("Office", DefaultProfile); err != nil {
		t.Fatal(err)
	}
	if err := CreateProfile("Office", ""); err == nil {
		t.Fatal("duplicate create should fail")
	}
	if got := ListProfiles(); len(got) != 2 || got[1] != "Office" {
		t.Fatalf("profiles = %v", got)
	}
	if err := SetActiveProfile("Office"); err != nil {
		t.Fatal(err)
	}
	if ActiveConfigPath() != filepath.Join(dir, "profiles", "Office.toml") {
		t.Fatalf("path = %s", ActiveConfigPath())
	}
	if info, _ := os.Stat(ActiveConfigPath()); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode())
	}
	if err := DeleteProfile("Office"); err != nil {
		t.Fatal(err)
	}
	if got := ActiveProfile(); got != DefaultProfile {
		t.Fatalf("after delete active = %q", got)
	}
	if DeleteProfile(DefaultProfile) == nil || ValidProfileName("../x") || ValidProfileName("Default") {
		t.Fatal("validation too lax")
	}
	if QuoteString("a\"b\\c") != `"a\"b\\c"` || FormatFloat(2) != "2.0" {
		t.Fatal("formatting wrong")
	}
}
