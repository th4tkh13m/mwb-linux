//go:build linux

package tray

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	menuPath  = "/MenuBar"
	menuIface = "com.canonical.dbusmenu"
)

// State is the current setting values shown as checked items in the menu.
type State struct {
	Edge         string // "left" or "right"
	Clipboard    bool
	FileTransfer bool
	Profile      string   // profile in use
	Profiles     []string // all profile names
}

// Actions are called when the user picks a menu entry.
type Actions struct {
	// Set changes one setting; key is a config key, raw is its TOML value.
	Set           func(key, raw string)
	SwitchProfile func(name string)
	OpenSettings  func()
	ShowScreen    func()
	OpenConfig    func()
	Quit          func()
}

const menuIntro = `<node>
  <interface name="com.canonical.dbusmenu">
    <method name="GetLayout">
      <arg type="i" direction="in"/><arg type="i" direction="in"/><arg type="as" direction="in"/>
      <arg type="u" direction="out"/><arg type="(ia{sv}av)" direction="out"/>
    </method>
    <method name="GetGroupProperties">
      <arg type="ai" direction="in"/><arg type="as" direction="in"/>
      <arg type="a(ia{sv})" direction="out"/>
    </method>
    <method name="GetProperty">
      <arg type="i" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="out"/>
    </method>
    <method name="Event">
      <arg type="i" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="in"/><arg type="u" direction="in"/>
    </method>
    <method name="AboutToShow"><arg type="i" direction="in"/><arg type="b" direction="out"/></method>
    <signal name="LayoutUpdated"><arg type="u"/><arg type="i"/></signal>
    <property name="Version" type="u" access="read"/>
    <property name="TextDirection" type="s" access="read"/>
    <property name="Status" type="s" access="read"/>
    <property name="IconThemePath" type="as" access="read"/>
  </interface>` + introspect.IntrospectDataString + `</node>`

type layout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type entry struct {
	id       int32
	props    map[string]dbus.Variant
	children []entry
}

func sep(id int32) entry {
	return entry{id: id, props: map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}}
}

func label(id int32, text string, enabled bool) entry {
	return entry{id: id, props: map[string]dbus.Variant{
		"label": dbus.MakeVariant(text), "enabled": dbus.MakeVariant(enabled)}}
}

func toggle(id int32, text, kind string, on bool) entry {
	state := int32(0)
	if on {
		state = 1
	}
	return entry{id: id, props: map[string]dbus.Variant{
		"label": dbus.MakeVariant(text), "toggle-type": dbus.MakeVariant(kind),
		"toggle-state": dbus.MakeVariant(state)}}
}

const (
	idStatus   = 1
	idSep1     = 2
	idProfiles = 3
	idSettings = 4
	idSep2     = 5
	idLeft     = 6
	idRight    = 7
	idSep3     = 8
	idClip     = 9
	idFiles    = 10
	idSep4     = 11
	idScreen   = 12
	idConfig   = 13
	idSep5     = 14
	idQuit     = 15

	idProfileBase = 100 // profile i has id idProfileBase+i
)

func (t *Tray) entries() []entry {
	st := t.state()
	status := "Waiting for " + t.host
	t.mu.Lock()
	if t.connected {
		status = "Connected to " + t.host
	}
	t.mu.Unlock()
	profiles := entry{id: idProfiles, props: map[string]dbus.Variant{
		"label":            dbus.MakeVariant("Profile: " + st.Profile),
		"children-display": dbus.MakeVariant("submenu"),
	}}
	for i, name := range st.Profiles {
		profiles.children = append(profiles.children, toggle(int32(idProfileBase+i), name, "radio", name == st.Profile))
	}
	return []entry{
		label(idStatus, status, false),
		sep(idSep1),
		profiles,
		label(idSettings, "Settings\u2026", true),
		sep(idSep2),
		toggle(idLeft, "Switch at left edge", "radio", st.Edge != "right"),
		toggle(idRight, "Switch at right edge", "radio", st.Edge == "right"),
		sep(idSep3),
		toggle(idClip, "Clipboard sync", "checkmark", st.Clipboard),
		toggle(idFiles, "File transfer", "checkmark", st.FileTransfer),
		sep(idSep4),
		label(idScreen, "Show Windows screen", true),
		label(idConfig, "Open config file", true),
		sep(idSep5),
		label(idQuit, "Quit", true),
	}
}

// flatten returns every entry including submenu children.
func flatten(es []entry) []entry {
	var out []entry
	for _, e := range es {
		out = append(out, e)
		out = append(out, flatten(e.children)...)
	}
	return out
}

func (e entry) layout() layout {
	l := layout{ID: e.id, Props: e.props, Children: []dbus.Variant{}}
	for _, c := range e.children {
		l.Children = append(l.Children, dbus.MakeVariant(c.layout()))
	}
	return l
}

func (t *Tray) state() State {
	if t.getState != nil {
		return t.getState()
	}
	return State{Edge: "left", Clipboard: true}
}

// GetLayout implements com.canonical.dbusmenu.
func (t *Tray) GetLayout(parent, depth int32, names []string) (uint32, layout, *dbus.Error) {
	root := layout{ID: 0, Props: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}}
	for _, e := range t.entries() {
		root.Children = append(root.Children, dbus.MakeVariant(e.layout()))
	}
	return 1, root, nil
}

func (t *Tray) GetGroupProperties(ids []int32, names []string) ([]struct {
	ID    int32
	Props map[string]dbus.Variant
}, *dbus.Error) {
	var out []struct {
		ID    int32
		Props map[string]dbus.Variant
	}
	for _, e := range flatten(t.entries()) {
		out = append(out, struct {
			ID    int32
			Props map[string]dbus.Variant
		}{e.id, e.props})
	}
	return out, nil
}

func (t *Tray) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	for _, e := range flatten(t.entries()) {
		if e.id == id {
			if v, ok := e.props[name]; ok {
				return v, nil
			}
		}
	}
	return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no property %q on item %d", name, id))
}

func (t *Tray) AboutToShow(id int32) (bool, *dbus.Error) { return true, nil }

// Event handles clicks. Setting changes are applied by the Actions callbacks.
func (t *Tray) Event(id int32, eventID string, data dbus.Variant, ts uint32) *dbus.Error {
	if eventID != "clicked" || t.actions == nil {
		return nil
	}
	st := t.state()
	a := t.actions
	if id >= idProfileBase {
		if i := int(id - idProfileBase); i < len(st.Profiles) && st.Profiles[i] != st.Profile && a.SwitchProfile != nil {
			go a.SwitchProfile(st.Profiles[i])
		}
		return nil
	}
	switch id {
	case idSettings:
		if a.OpenSettings != nil {
			go a.OpenSettings()
		}
	case idLeft:
		if st.Edge == "right" && a.Set != nil {
			go a.Set("edge", `"left"`)
		}
	case idRight:
		if st.Edge != "right" && a.Set != nil {
			go a.Set("edge", `"right"`)
		}
	case idClip:
		if a.Set != nil {
			go a.Set("clipboard", fmt.Sprint(!st.Clipboard))
		}
	case idFiles:
		if a.Set != nil {
			go a.Set("file_transfer", fmt.Sprint(!st.FileTransfer))
		}
	case idScreen:
		if a.ShowScreen != nil {
			go a.ShowScreen()
		}
	case idConfig:
		if a.OpenConfig != nil {
			go a.OpenConfig()
		}
	case idQuit:
		if a.Quit != nil {
			go a.Quit()
		}
	}
	return nil
}

func (t *Tray) menuProps() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"Version":       dbus.MakeVariant(uint32(3)),
		"TextDirection": dbus.MakeVariant("ltr"),
		"Status":        dbus.MakeVariant("normal"),
		"IconThemePath": dbus.MakeVariant([]string{}),
	}
}

// menuProperties exposes the dbusmenu properties on the menu object.
type menuProperties struct{ t *Tray }

func (m menuProperties) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	if v, ok := m.t.menuProps()[prop]; ok {
		return v, nil
	}
	return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no property %s", prop))
}
func (m menuProperties) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	return m.t.menuProps(), nil
}
func (m menuProperties) Set(iface, prop string, v dbus.Variant) *dbus.Error {
	return dbus.MakeFailedError(fmt.Errorf("read-only"))
}
