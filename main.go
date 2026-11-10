// Pixel Control Center — menu-bar shell.
//
// Left click toggles the flyout under the status item. Right click opens
// the scene menu (stub in M0) and Settings. Closing the flyout or the
// settings window must not quit the app.
package main

import (
	"embed"
	"io/fs"
	"log"
	"runtime"

	"github.com/egoist/mygo"
)

//go:embed all:frontend
var frontendFS embed.FS

//go:embed frontend/assets/tray.png
var trayPNG []byte

const (
	flyoutWidth  = 360
	flyoutHeight = 360
)

var (
	tray           *mygo.Tray
	panel          *mygo.Window
	settingsWindow *mygo.Window
)

func showFlyout() {
	if panel == nil {
		panel = mygo.NewFlyout(mygo.FlyoutOptions{
			Tray:      tray,
			Placement: mygo.PlacementBottom,
			Gap:       4,
			Width:     flyoutWidth,
			Height:    flyoutHeight,
			Focusable: true,
			URL:       "/",
		})
		panel.OnClosed(func() {
			panel = nil
		})
		return
	}
	if panel.IsVisible() {
		panel.Hide()
		return
	}
	panel.SetSize(flyoutWidth, flyoutHeight)
	panel.Show()
}

func toggleFlyout() {
	if panel != nil && panel.IsVisible() {
		panel.Hide()
		return
	}
	showFlyout()
}

func openSettings() {
	if settingsWindow != nil && !settingsWindow.IsDestroyed() {
		settingsWindow.Show()
		settingsWindow.Focus()
		mygo.App.Focus()
		return
	}
	settingsWindow = mygo.NewWindow(mygo.WindowOptions{
		Title:           "设置",
		URL:             "/settings.html",
		Width:           480,
		Height:          560,
		MinWidth:        480,
		MinHeight:       560,
		UseContentSize:  true,
		BackgroundColor: "#1c1724",
	})
	settingsWindow.OnClosed(func() {
		settingsWindow = nil
	})
	settingsWindow.Show()
	mygo.App.Focus()
}

func popupMenu() {
	if panel != nil && panel.IsVisible() {
		panel.Hide()
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Home", Type: mygo.MenuItemRadio, Checked: true},
		mygo.Separator(),
		{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { openSettings() }},
		{Role: mygo.RoleQuit},
	})
	tray.SetMenu(menu)
	defer tray.SetMenu(nil)
	tray.PopUpMenu()
}

func ready() {
	var err error
	tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayPNG,
		IconIsTemplate: true,
		ToolTip:        "Pixel Control Center",
	})
	if err != nil {
		log.Fatal("tray: ", err)
	}
	if runtime.GOOS == "linux" {
		tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Open", Click: func(*mygo.MenuItem, *mygo.Window) { toggleFlyout() }},
			{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { openSettings() }},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}))
	} else {
		tray.OnClick(toggleFlyout)
		tray.OnRightClick(popupMenu)
	}
}

func main() {
	web, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mygo.SetFrontend(web)

	mygo.App.SetName("Pixel Control Center")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		if tray != nil {
			tray.Destroy()
		}
	})
	mygo.App.WhenReady(ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
