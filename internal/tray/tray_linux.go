//go:build linux

// Package tray exposes a StatusNotifierItem so desktops such as KDE Plasma show
// a system-tray icon reflecting the connection state.
package tray

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	itemPath  = "/StatusNotifierItem"
	itemIface = "org.kde.StatusNotifierItem"
	watcher   = "org.kde.StatusNotifierWatcher"
	iconName  = "mwb"
)

const introXML = `<node>
  <interface name="org.kde.StatusNotifierItem">
    <method name="Activate"><arg type="i" direction="in"/><arg type="i" direction="in"/></method>
    <method name="SecondaryActivate"><arg type="i" direction="in"/><arg type="i" direction="in"/></method>
    <method name="ContextMenu"><arg type="i" direction="in"/><arg type="i" direction="in"/></method>
    <method name="Scroll"><arg type="i" direction="in"/><arg type="s" direction="in"/></method>
    <signal name="NewStatus"><arg type="s"/></signal>
    <signal name="NewToolTip"/>
    <signal name="NewIcon"/>
    <signal name="NewTitle"/>
    <property name="Category" type="s" access="read"/>
    <property name="Id" type="s" access="read"/>
    <property name="Title" type="s" access="read"/>
    <property name="Status" type="s" access="read"/>
    <property name="WindowId" type="i" access="read"/>
    <property name="IconName" type="s" access="read"/>
    <property name="IconPixmap" type="a(iiay)" access="read"/>
    <property name="OverlayIconName" type="s" access="read"/>
    <property name="AttentionIconName" type="s" access="read"/>
    <property name="ItemIsMenu" type="b" access="read"/>
    <property name="Menu" type="o" access="read"/>
    <property name="ToolTip" type="(sa(iiay)ss)" access="read"/>
  </interface>` + introspect.IntrospectDataString + `</node>`

// Tray is a running status icon.
type Tray struct {
	conn *dbus.Conn
	mu   sync.Mutex
	// connected selects the "Active" (connected) or "Passive" (waiting) state.
	connected bool
	host      string
	getState  func() State
	actions   *Actions
}

// Start registers the tray icon. It returns an error if the session bus or a
// StatusNotifierWatcher is unavailable; callers should treat that as non-fatal.
func Start(host string, getState func() State, actions *Actions) (*Tray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}
	t := &Tray{conn: conn, host: host, getState: getState, actions: actions}
	if err := conn.Export(t, itemPath, itemIface); err != nil {
		return nil, err
	}
	if err := conn.Export(t, itemPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, err
	}
	if err := conn.Export(t, menuPath, menuIface); err != nil {
		return nil, err
	}
	if err := conn.Export(menuProperties{t}, menuPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, err
	}
	if err := conn.Export(introspect.Introspectable(menuIntro), menuPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	if err := conn.Export(introspect.Introspectable(introXML), itemPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("claim %s: %v", name, err)
	}
	call := conn.Object(watcher, "/StatusNotifierWatcher").Call(watcher+".RegisterStatusNotifierItem", 0, name)
	if call.Err != nil {
		return nil, fmt.Errorf("register with watcher: %w", call.Err)
	}
	slog.Info("tray icon registered")
	return t, nil
}

// SetConnected updates the icon state and tooltip.
func (t *Tray) SetConnected(ok bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.connected = ok
	t.mu.Unlock()
	status := "Passive"
	if ok {
		status = "Active"
	}
	_ = t.conn.Emit(itemPath, itemIface+".NewStatus", status)
	_ = t.conn.Emit(itemPath, itemIface+".NewIcon")
	_ = t.conn.Emit(itemPath, itemIface+".NewToolTip")
	_ = t.conn.Emit(menuPath, menuIface+".LayoutUpdated", uint32(1), int32(0))
}

// Close releases the bus connection.
func (t *Tray) Close() {
	if t != nil {
		_ = t.conn.Close()
	}
}

// StatusNotifierItem methods. The right-click menu is provided via dbusmenu.
func (t *Tray) Activate(x, y int32) *dbus.Error          { return nil }
func (t *Tray) ContextMenu(x, y int32) *dbus.Error       { return nil }
func (t *Tray) Scroll(d int32, o string) *dbus.Error     { return nil }
func (t *Tray) SecondaryActivate(x, y int32) *dbus.Error { return nil }

// Get and GetAll implement org.freedesktop.DBus.Properties.
func (t *Tray) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	props := t.props()
	v, ok := props[prop]
	if !ok {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no property %s", prop))
	}
	return v, nil
}

func (t *Tray) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	return t.props(), nil
}

func (t *Tray) Set(iface, prop string, v dbus.Variant) *dbus.Error {
	return dbus.MakeFailedError(fmt.Errorf("read-only"))
}

func (t *Tray) props() map[string]dbus.Variant {
	t.mu.Lock()
	connected := t.connected
	t.mu.Unlock()
	status, text := "Passive", "Waiting for "+t.host
	if connected {
		status, text = "Active", "Connected to "+t.host
	}
	type pixmap = iconPixmap
	pix := []pixmap{drawIcon(22, !connected), drawIcon(32, !connected), drawIcon(48, !connected)}
	return map[string]dbus.Variant{
		"Category":          dbus.MakeVariant("ApplicationStatus"),
		"Id":                dbus.MakeVariant("mwb"),
		"Title":             dbus.MakeVariant("Mouse Without Borders"),
		"Status":            dbus.MakeVariant(status),
		"WindowId":          dbus.MakeVariant(int32(0)),
		"IconName":          dbus.MakeVariant(""),
		"IconPixmap":        dbus.MakeVariant(pix),
		"OverlayIconName":   dbus.MakeVariant(""),
		"AttentionIconName": dbus.MakeVariant(""),
		"ItemIsMenu":        dbus.MakeVariant(true),
		"Menu":              dbus.MakeVariant(dbus.ObjectPath(menuPath)),
		"ToolTip": dbus.MakeVariant(struct {
			Icon string
			Pix  []pixmap
			Head string
			Body string
		}{"", pix, "Mouse Without Borders", text}),
	}
}
