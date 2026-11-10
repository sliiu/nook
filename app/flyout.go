package app

import "github.com/egoist/mygo"

// Shell is the menu-bar surface Control drives for short/full flyouts.
// main.go implements it with the tray flyout window.
type Shell interface {
	ShowFull()
	ShowShort()
	Hide()
	Form() string
	Panel() *mygo.Window
}
