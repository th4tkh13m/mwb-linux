package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DefaultProfile is the profile stored in config.toml itself. Additional
// profiles live in profiles/<name>.toml next to it.
const DefaultProfile = "Default"

var profileNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{0,39}$`)

// ValidProfileName reports whether name is usable as a profile name.
func ValidProfileName(name string) bool {
	return name != DefaultProfile && profileNameRE.MatchString(name) && !strings.HasSuffix(name, " ")
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mwb"), nil
}

// ProfilePath returns the config file for a profile name.
func ProfilePath(name string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if name == "" || name == DefaultProfile {
		return filepath.Join(dir, "config.toml"), nil
	}
	if !ValidProfileName(name) {
		return "", fmt.Errorf("invalid profile name %q", name)
	}
	return filepath.Join(dir, "profiles", name+".toml"), nil
}

// ListProfiles returns Default first, then the other profiles by name.
func ListProfiles() []string {
	names := []string{DefaultProfile}
	dir, err := configDir()
	if err != nil {
		return names
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "profiles"))
	var extra []string
	for _, e := range entries {
		n := strings.TrimSuffix(e.Name(), ".toml")
		if !e.IsDir() && n != e.Name() && ValidProfileName(n) {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	return append(names, extra...)
}

// ActiveProfile returns the profile in use, falling back to Default when the
// marker is missing or names a profile that no longer exists.
func ActiveProfile() string {
	dir, err := configDir()
	if err != nil {
		return DefaultProfile
	}
	data, err := os.ReadFile(filepath.Join(dir, "active-profile"))
	if err != nil {
		return DefaultProfile
	}
	name := strings.TrimSpace(string(data))
	for _, n := range ListProfiles() {
		if n == name {
			return name
		}
	}
	return DefaultProfile
}

// SetActiveProfile records which profile is in use.
func SetActiveProfile(name string) error {
	found := false
	for _, n := range ListProfiles() {
		found = found || n == name
	}
	if !found {
		return fmt.Errorf("no such profile %q", name)
	}
	dir, err := configDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "active-profile"), []byte(name+"\n"), 0o600)
}

// ActiveConfigPath returns the config file of the profile in use.
func ActiveConfigPath() string {
	p, err := ProfilePath(ActiveProfile())
	if err != nil {
		p, _ = ProfilePath(DefaultProfile)
	}
	return p
}

// CreateProfile writes a new profile file, copying fromName when it is set.
// The file is owner-only because it holds the security key.
func CreateProfile(name, fromName string) error {
	path, err := ProfilePath(name)
	if err != nil {
		return err
	}
	if name == DefaultProfile {
		return fmt.Errorf("profile %q already exists", name)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("profile %q already exists", name)
	}
	var data []byte
	if fromName != "" {
		src, err := ProfilePath(fromName)
		if err != nil {
			return err
		}
		if data, err = os.ReadFile(src); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// DeleteProfile removes a profile other than Default.
func DeleteProfile(name string) error {
	if name == DefaultProfile {
		return fmt.Errorf("the Default profile cannot be deleted")
	}
	path, err := ProfilePath(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// QuoteString renders s as a TOML basic string.
func QuoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			b.WriteString(fmt.Sprintf(`\u%04x`, r))
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// FormatFloat renders a TOML float that always has a decimal point.
func FormatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".") {
		s += ".0"
	}
	return s
}
