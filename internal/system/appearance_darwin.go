//go:build darwin

// Appearance via the private SkyLight framework.
//
// SLSGetAppearanceThemeLegacy returns 0 for light and 1 for dark.
// SLSGetAppearanceThemeSwitchesAutomatically is the "auto" setting, which
// wins over the theme while it is on. Setting auto off keeps the current
// theme. A write can prompt for permission to control system events; a
// refusal makes later writes report unavailable.
package system

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

const skyLightPath = "/System/Library/PrivateFrameworks/SkyLight.framework/SkyLight"

const (
	AppearanceLight = "light"
	AppearanceDark  = "dark"
	AppearanceAuto  = "auto"
)

var (
	appearanceOnce sync.Once
	appearanceOK   bool

	getTheme func() int32
	setTheme func(theme int32) int32
	getAuto  func() bool
	setAuto  func(enabled bool)
)

type appearanceBackend struct{ ok bool }

// NewAppearance resolves the appearance symbols. It always succeeds;
// Available is false when a symbol is missing.
func NewAppearance() (Backend, error) {
	appearanceOnce.Do(loadAppearance)
	return appearanceBackend{ok: appearanceOK}, nil
}

func loadAppearance() {
	lib, err := purego.Dlopen(skyLightPath, purego.RTLD_LAZY)
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&getTheme, lib, "SLSGetAppearanceThemeLegacy")
	purego.RegisterLibFunc(&setTheme, lib, "SLSSetAppearanceThemeLegacy")
	purego.RegisterLibFunc(&getAuto, lib, "SLSGetAppearanceThemeSwitchesAutomatically")
	purego.RegisterLibFunc(&setAuto, lib, "SLSSetAppearanceThemeSwitchesAutomatically")
	appearanceOK = getTheme != nil && setTheme != nil && getAuto != nil && setAuto != nil
}

func (b appearanceBackend) Available() bool { return b.ok }

func (b appearanceBackend) State() (string, []Option, error) {
	if !b.ok {
		return "", nil, nil
	}
	selected := AppearanceLight
	if getTheme() != 0 {
		selected = AppearanceDark
	}
	if getAuto() {
		selected = AppearanceAuto
	}
	return selected, appearanceOptions(), nil
}

func (b appearanceBackend) Select(id string) error {
	if !b.ok {
		return nil
	}
	switch id {
	case AppearanceAuto:
		setAuto(true)
	case AppearanceLight, AppearanceDark:
		var theme int32
		if id == AppearanceDark {
			theme = 1
		}
		if status := setTheme(theme); status != 0 {
			return fmt.Errorf("appearance: set theme failed (%d)", status)
		}
		setAuto(false)
	default:
		return fmt.Errorf("appearance: unknown %q", id)
	}
	return nil
}

func appearanceOptions() []Option {
	return []Option{
		{ID: AppearanceLight, Label: "Light"},
		{ID: AppearanceDark, Label: "Dark"},
		{ID: AppearanceAuto, Label: "Auto"},
	}
}
