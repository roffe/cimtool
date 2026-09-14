//go:build linux

package gui

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"

	"github.com/godbus/dbus/v5"
)

// Native file dialogs via xdg-desktop-portal. Unlike sqweek/dialog's GTK
// dialogs these are safe when Fyne runs as a native Wayland client. We speak
// DBus directly on a private connection instead of using rymdport/portal:
// its v0.4.2 listens for signals on the session bus connection shared with
// Fyne's theme watcher and mistakes unrelated signals for the response.

type fileFilterRule struct {
	Type    uint32 // 0 = glob pattern
	Pattern string
}

type fileFilter struct {
	Name  string
	Rules []fileFilterRule
}

var binFilter = []fileFilter{{Name: "Bin file", Rules: []fileFilterRule{{Type: 0, Pattern: "*.bin"}}}}

// openFileNative returns the selected path, or "" if the user cancelled.
func openFileNative(title string) (string, error) {
	return portalFileDialog("org.freedesktop.portal.FileChooser.OpenFile", title, nil)
}

// saveFileNative returns the selected path, or "" if the user cancelled.
func saveFileNative(title, suggestedFilename string) (string, error) {
	return portalFileDialog("org.freedesktop.portal.FileChooser.SaveFile", title, map[string]dbus.Variant{
		"current_name": dbus.MakeVariant(suggestedFilename),
	})
}

func portalFileDialog(method, title string, extra map[string]dbus.Variant) (string, error) {
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := conn.Auth(nil); err != nil {
		return "", err
	}
	if err := conn.Hello(); err != nil {
		return "", err
	}

	// The portal emits the Response signal on a request object whose path is
	// derived from our unique bus name and the handle_token we pass along.
	token := fmt.Sprintf("cimtool%d", rand.Uint64())
	sender := strings.NewReplacer(":", "", ".", "_").Replace(conn.Names()[0])
	requestPath := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)

	matchOpts := func(path dbus.ObjectPath) []dbus.MatchOption {
		return []dbus.MatchOption{
			dbus.WithMatchObjectPath(path),
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
		}
	}
	if err := conn.AddMatchSignal(matchOpts(requestPath)...); err != nil {
		return "", err
	}

	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"filters":      dbus.MakeVariant(binFilter),
	}
	for k, v := range extra {
		options[k] = v
	}

	var handle dbus.ObjectPath
	desktop := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	if err := desktop.Call(method, 0, "", title, options).Store(&handle); err != nil {
		return "", err
	}
	if handle != requestPath {
		// Older portal versions ignore handle_token; listen on the returned path too.
		if err := conn.AddMatchSignal(matchOpts(handle)...); err != nil {
			return "", err
		}
	}

	for sig := range signals {
		if sig.Path != requestPath && sig.Path != handle {
			continue
		}
		if len(sig.Body) != 2 {
			return "", errors.New("unexpected response from portal")
		}
		if status, _ := sig.Body[0].(uint32); status != 0 {
			return "", nil // cancelled
		}
		results, _ := sig.Body[1].(map[string]dbus.Variant)
		uris, _ := results["uris"].Value().([]string)
		if len(uris) == 0 {
			return "", nil
		}
		path, err := url.PathUnescape(strings.TrimPrefix(uris[0], "file://"))
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", errors.New("portal connection closed")
}
