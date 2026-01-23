//go:build darwin

package main

import "log"

// QuitTray no-op on macOS to avoid AppDelegate conflict with systray.
func QuitTray() {}

// startTray on macOS: disable systray to avoid duplicate AppDelegate symbols.
func (a *App) startTray() {
	log.Println("tray: disabled on darwin (systray AppDelegate conflict)")
}
