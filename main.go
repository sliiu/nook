// Pixel Control Center — menu-bar shell.
//
// Left click toggles the full flyout under the status item. Right click opens
// the scene menu (stub until M4) and Settings. External volume changes open
// the short flyout for ~1.6s. Closing the flyout or the settings window must
// not quit the app.
package main

import (
	"embed"
	"io/fs"
	"log"
	"runtime"
	"sync"

	"github.com/egoist/mygo"

	"pixel-control-center/app"
	"pixel-control-center/internal/audio"
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
	bridge         *app.Control
	watcher        chan struct{}
	shellMu        sync.Mutex
	shellForm      = "hidden"
)

// trayShell adapts the package-level flyout to app.Shell.
type trayShell struct{}

func (trayShell) ShowFull()  { showFlyout("full") }
func (trayShell) ShowShort() { showFlyout("short") }
func (trayShell) Hide()      { hideFlyout() }
func (trayShell) Form() string {
	shellMu.Lock()
	defer shellMu.Unlock()
	return shellForm
}
func (trayShell) Panel() *mygo.Window { return panel }

func setForm(form string) {
	shellMu.Lock()
	shellForm = form
	shellMu.Unlock()
}

func showFlyout(form string) {
	if form != "full" && form != "short" {
		form = "short"
	}
	setForm(form)

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
			setForm("hidden")
		})
	} else if !panel.IsVisible() {
		panel.SetSize(flyoutWidth, flyoutHeight)
		panel.Show()
	} else {
		panel.SetSize(flyoutWidth, flyoutHeight)
	}
	app.PillForm.Emit(panel, app.FormPayload{Form: form})
}

func hideFlyout() {
	setForm("hidden")
	if panel != nil {
		panel.Hide()
		if panel != nil {
			app.PillForm.Emit(panel, app.FormPayload{Form: "hidden"})
		}
	}
}

func toggleFlyout() {
	shellMu.Lock()
	form := shellForm
	shellMu.Unlock()
	switch form {
	case "full":
		hideFlyout()
	default:
		showFlyout("full")
	}
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
		hideFlyout()
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

	watcher = make(chan struct{})
	go bridge.Watch(watcher)
}

func main() {
	web, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mygo.SetFrontend(web)

	backend, err := audio.New()
	if err != nil {
		log.Fatal("output volume unavailable: ", err)
	}
	bridge = app.NewControl(backend, trayShell{})
	mygo.Bind(bridge)

	mygo.App.SetName("Pixel Control Center")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		if watcher != nil {
			close(watcher)
			watcher = nil
		}
		if tray != nil {
			tray.Destroy()
		}
	})
	mygo.App.WhenReady(ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
