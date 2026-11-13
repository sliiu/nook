// Nook — menu-bar shell.
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
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"

	"pixel-control-center/app"
	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
	"pixel-control-center/internal/controls"
	"pixel-control-center/internal/devices"
	"pixel-control-center/internal/keyboard"
	"pixel-control-center/internal/media"
	"pixel-control-center/internal/power"
	"pixel-control-center/internal/prefs"
	"pixel-control-center/internal/scenes"
	"pixel-control-center/internal/system"
)

//go:embed all:frontend
var frontendFS embed.FS

//go:embed frontend/scenes/home.html
var factoryHome string

//go:embed frontend/assets/tray.png
var trayPNG []byte

const (
	defaultFlyout = 360
)

var (
	tray           *mygo.Tray
	panel          *mygo.Window
	settingsWindow *mygo.Window
	bridge         *app.Control
	store          *scenes.Store
	settings       *prefs.Store
	watcher        chan struct{}
	shellMu        sync.Mutex
	shellForm      = "hidden"
	flyoutW        = defaultFlyout
	flyoutH        = defaultFlyout
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

func flyoutSize() (int, int) {
	shellMu.Lock()
	defer shellMu.Unlock()
	return flyoutW, flyoutH
}

// resizeFlyout is called when the scene reports its own size.
func resizeFlyout(w, h int) {
	shellMu.Lock()
	flyoutW, flyoutH = w, h
	shellMu.Unlock()
	if panel != nil {
		panel.SetSize(w, h)
	}
}

func showFlyout(form string) {
	fw, fh := flyoutSize()
	if form != "full" && form != "short" {
		form = "short"
	}
	setForm(form)

	if panel == nil {
		panel = mygo.NewFlyout(mygo.FlyoutOptions{
			Tray:      tray,
			Placement: mygo.PlacementBottom,
			Gap:       4,
			Width:     fw,
			Height:    fh,
			Focusable: true,
			URL:       "/",
		})
		panel.OnClosed(func() {
			panel = nil
			setForm("hidden")
		})
	} else if !panel.IsVisible() {
		panel.SetSize(fw, fh)
		panel.Show()
	} else {
		panel.SetSize(fw, fh)
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
		Width:           940,
		Height:          640,
		MinWidth:        900,
		MinHeight:       560,
		UseContentSize:  true,
		BackgroundColor: "#1e1b24",
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
	items := []*mygo.MenuItem{}
	if store != nil {
		current := store.Current().ID
		for _, sc := range store.List() {
			id := sc.ID
			items = append(items, &mygo.MenuItem{
				Label:   sc.Name,
				Type:    mygo.MenuItemRadio,
				Checked: id == current,
				Click:   func(*mygo.MenuItem, *mygo.Window) { _ = bridge.SelectScene(id) },
			})
		}
		items = append(items, mygo.Separator())
	}
	items = append(items,
		&mygo.MenuItem{Label: tr("settings"), Click: func(*mygo.MenuItem, *mygo.Window) { openSettings() }},
		updateMenuItem(),
		&mygo.MenuItem{Label: tr("quit"), Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	)
	menu := mygo.NewMenu(items)
	tray.SetMenu(menu)
	defer tray.SetMenu(nil)
	tray.PopUpMenu()
}

var menuText = map[string]map[string]string{
	"zh": {"settings": "设置…", "quit": "退出 Nook", "open": "打开"},
	"en": {"settings": "Settings…", "quit": "Quit Nook", "open": "Open"},
	"ja": {"settings": "設定…", "quit": "Nook を終了", "open": "開く"},
}

func tr(key string) string {
	lang := "en"
	if settings != nil {
		lang = settings.Resolved()
	}
	return menuText[lang][key]
}

// langChanged fans the new language out to every window and to the
// updater, whose window speaks the language the app shows.
func langChanged() {
	info := bridge.GetLang()
	mygo.App.SetLocale(info.Resolved)
	for _, w := range mygo.Windows() {
		app.LangChanged.Emit(w, info)
	}
}

// updateMenuItem is "Check for Updates…" in the language the app shows.
// The popup menu is built on each right click, so the label follows a
// language change without rebuilding anything else.
func updateMenuItem() *mygo.MenuItem {
	return updater.MenuItem()
}

// migrateOldData copies scenes.json from the pre-rename data directory once.
func migrateOldData(dir string) {
	target := filepath.Join(dir, "scenes.json")
	if _, err := os.Stat(target); err == nil {
		return
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "Pixel Control Center", "scenes.json"))
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(target, raw, 0o644)
}

// scenesChanged fans the new selection out to every window.
func scenesChanged() {
	if store == nil {
		return
	}
	payload := app.ScenesPayload{Current: store.Current().ID}
	for _, w := range mygo.Windows() {
		app.ScenesChanged.Emit(w, payload)
	}
}

func ready() {
	var err error
	tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayPNG,
		IconIsTemplate: true,
		ToolTip:        "Nook",
	})
	if err != nil {
		log.Fatal("tray: ", err)
	}
	if runtime.GOOS == "linux" {
		tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: tr("open"), Click: func(*mygo.MenuItem, *mygo.Window) { toggleFlyout() }},
			{Label: tr("settings"), Click: func(*mygo.MenuItem, *mygo.Window) { openSettings() }},
			updateMenuItem(),
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}))
	} else {
		tray.OnClick(toggleFlyout)
		tray.OnRightClick(popupMenu)
	}

	watcher = make(chan struct{})
	go app.Watch(bridge, watcher)
}

// openSceneLink installs the catalog scene a nook:// link names, then shows
// it in Settings. The fetch runs off the main thread.
func openSceneLink(link string) {
	go func() {
		if _, err := bridge.InstallFromLink(link); err != nil {
			log.Printf("scene link: %v", err)
			openSettings()
			mygo.Dialog.Message(mygo.MessageOptions{
				Type:    mygo.MessageError,
				Title:   "Nook",
				Message: err.Error(),
			})
			return
		}
		openSettings()
	}()
}

func main() {
	mygo.Use(updater.Plugin)
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.App.OnOpenURL(openSceneLink)
	if err := mygo.App.RegisterURLScheme("nook"); err != nil {
		log.Printf("nook: url scheme: %v", err)
	}

	web, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mygo.SetFrontend(web)

	audioBackend, err := audio.New()
	if err != nil {
		log.Fatal("output volume unavailable: ", err)
	}
	brightBackend, err := brightness.New()
	if err != nil {
		log.Fatal("brightness backend: ", err)
	}
	bridge = app.NewControl(audioBackend, brightBackend, trayShell{})
	app.AttachCatalog(bridge, controlCatalog(audioBackend, inputBackend(), brightBackend))
	mygo.Bind(bridge)

	mygo.App.SetName("Nook")
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
	mygo.App.WhenReady(func() {
		dir, err := mygo.App.Path(mygo.PathUserData)
		if err != nil {
			log.Fatal("user data dir: ", err)
		}
		migrateOldData(dir)
		settings = prefs.Open(filepath.Join(dir, "settings.json"))
		app.AttachPrefs(bridge, settings, langChanged)
		// The plugin follows App.Locale, which otherwise stays the system
		// language and ignores the choice kept in settings.
		mygo.App.SetLocale(settings.Resolved())
		updater.OnChange(func() {
			info := bridge.UpdateStatus()
			for _, w := range mygo.Windows() {
				app.UpdateChanged.Emit(w, info)
			}
		})
		store, err = scenes.Open(filepath.Join(dir, "scenes.json"), factoryHome)
		if err != nil {
			log.Fatal("scenes: ", err)
		}
		app.AttachScenes(bridge, store, scenesChanged)
		app.AttachFlyoutSize(bridge, resizeFlyout)
		ready()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// controlCatalog builds the scene catalog. A backend that fails to open is
// left out, and its control then reads as unavailable.
func controlCatalog(output audio.Backend, input audio.Backend, display brightness.Backend) []controls.Control {
	open := func(name string, fn func() (any, error)) any {
		v, err := fn()
		if err != nil {
			log.Printf("nook: %s unavailable: %v", name, err)
			return nil
		}
		return v
	}
	return controls.Desktop(controls.Hardware{
		Output:     output,
		Input:      input,
		Display:    display,
		Keyboard:   open("keyboard", func() (any, error) { return keyboard.New() }).(keyboard.Backend),
		Devices:    open("devices", func() (any, error) { return devices.New() }).(devices.Backend),
		Media:      open("media", func() (any, error) { return media.New() }).(media.Backend),
		Appearance: open("appearance", func() (any, error) { return system.NewAppearance() }).(system.Backend),
		WiFi:       open("wifi", func() (any, error) { return power.NewWiFi() }).(power.Backend),
	})

}

// inputBackend opens the microphone. No default input device is not fatal;
// the control reads as unavailable.
func inputBackend() audio.Backend {
	b, err := audio.NewInput()
	if err != nil {
		log.Printf("nook: microphone unavailable: %v", err)
		return nil
	}
	return b
}
