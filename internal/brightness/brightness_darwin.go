//go:build darwin

// Built-in display brightness via private macOS frameworks.
//
// Prefer DisplayServices Get/SetBrightness when
// DisplayServicesCanChangeBrightness is true on the built-in panel.
// Fall back to CoreDisplay user brightness only after a write probe.
// External-only setups report Available=false (v1 is built-in only).
// Symbols are resolved with purego/dlopen so this package stays cgo-free.
package brightness

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

const (
	displayServicesPath = "/System/Library/PrivateFrameworks/DisplayServices.framework/DisplayServices"
	coreDisplayPath     = "/System/Library/Frameworks/CoreDisplay.framework/CoreDisplay"
	coreGraphicsPath    = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"
)

type apiKind int

const (
	apiNone apiKind = iota
	apiDisplayServices
	apiCoreDisplay
)

var (
	loadOnce sync.Once
	loadErr  error
	loaded   apiKind
	display  uint32

	cgMainDisplayID       func() uint32
	cgGetActiveDisplayList func(maxDisplays uint32, displays *uint32, displayCount *uint32) int32
	cgDisplayIsBuiltin    func(display uint32) bool

	dsCan func(display uint32) bool
	dsGet func(display uint32, brightness *float32) int32
	dsSet func(display uint32, brightness float32) int32

	cdGet func(display uint32) float64
	cdSet func(display uint32, brightness float64)
)

type darwinBackend struct {
	kind    apiKind
	display uint32
}

// New resolves private brightness symbols. Always succeeds; Available may be false.
func New() (Backend, error) {
	loadOnce.Do(loadAPIs)
	return darwinBackend{kind: loaded, display: display}, nil
}

func (b darwinBackend) Available() bool { return b.kind != apiNone }

func (b darwinBackend) Level() (Level, error) {
	if b.kind == apiNone {
		return Level{Percent: 0, Available: false}, nil
	}
	switch b.kind {
	case apiDisplayServices:
		var value float32
		if status := dsGet(b.display, &value); status != 0 {
			return Level{}, fmt.Errorf("DisplayServicesGetBrightness failed (%d)", status)
		}
		return Level{Percent: UnitToPercent(value), Available: true}, nil
	case apiCoreDisplay:
		return Level{Percent: UnitToPercent(float32(cdGet(b.display))), Available: true}, nil
	default:
		return Level{Percent: 0, Available: false}, nil
	}
}

func (b darwinBackend) Set(percent int) error {
	if b.kind == apiNone {
		return nil
	}
	percent = ClampPercent(percent)
	switch b.kind {
	case apiDisplayServices:
		if status := dsSet(b.display, PercentToUnit(percent)); status != 0 {
			return fmt.Errorf("DisplayServicesSetBrightness failed (%d)", status)
		}
		return nil
	case apiCoreDisplay:
		cdSet(b.display, float64(PercentToUnit(percent)))
		return nil
	default:
		return nil
	}
}

func loadAPIs() {
	cg, err := purego.Dlopen(coreGraphicsPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		loadErr = fmt.Errorf("CoreGraphics: %w", err)
		return
	}
	purego.RegisterLibFunc(&cgMainDisplayID, cg, "CGMainDisplayID")
	purego.RegisterLibFunc(&cgGetActiveDisplayList, cg, "CGGetActiveDisplayList")
	purego.RegisterLibFunc(&cgDisplayIsBuiltin, cg, "CGDisplayIsBuiltin")

	builtin, ok := findBuiltinDisplay()
	if !ok {
		loadErr = fmt.Errorf("no built-in display")
		loaded = apiNone
		return
	}
	display = builtin

	if tryDisplayServices(builtin) {
		loaded = apiDisplayServices
		return
	}
	if tryCoreDisplay(builtin) {
		loaded = apiCoreDisplay
		return
	}
	loadErr = fmt.Errorf("built-in display has no usable brightness API")
	loaded = apiNone
}

func findBuiltinDisplay() (uint32, bool) {
	var count uint32
	if status := cgGetActiveDisplayList(0, nil, &count); status != 0 || count == 0 {
		// Fall back to main; still require IsBuiltin below.
		id := cgMainDisplayID()
		if id != 0 && cgDisplayIsBuiltin(id) {
			return id, true
		}
		return 0, false
	}
	ids := make([]uint32, count)
	if status := cgGetActiveDisplayList(count, &ids[0], &count); status != 0 {
		return 0, false
	}
	for _, id := range ids[:count] {
		if cgDisplayIsBuiltin(id) {
			return id, true
		}
	}
	return 0, false
}

func tryDisplayServices(id uint32) bool {
	lib, err := purego.Dlopen(displayServicesPath, purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return false
	}
	if _, err := purego.Dlsym(lib, "DisplayServicesCanChangeBrightness"); err != nil {
		return false
	}
	if _, err := purego.Dlsym(lib, "DisplayServicesGetBrightness"); err != nil {
		return false
	}
	if _, err := purego.Dlsym(lib, "DisplayServicesSetBrightness"); err != nil {
		return false
	}
	purego.RegisterLibFunc(&dsCan, lib, "DisplayServicesCanChangeBrightness")
	purego.RegisterLibFunc(&dsGet, lib, "DisplayServicesGetBrightness")
	purego.RegisterLibFunc(&dsSet, lib, "DisplayServicesSetBrightness")
	if !dsCan(id) {
		dsCan, dsGet, dsSet = nil, nil, nil
		return false
	}
	var value float32
	if dsGet(id, &value) != 0 {
		dsCan, dsGet, dsSet = nil, nil, nil
		return false
	}
	return true
}

func tryCoreDisplay(id uint32) bool {
	lib, err := purego.Dlopen(coreDisplayPath, purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return false
	}
	if _, err := purego.Dlsym(lib, "CoreDisplay_Display_GetUserBrightness"); err != nil {
		return false
	}
	if _, err := purego.Dlsym(lib, "CoreDisplay_Display_SetUserBrightness"); err != nil {
		return false
	}
	purego.RegisterLibFunc(&cdGet, lib, "CoreDisplay_Display_GetUserBrightness")
	purego.RegisterLibFunc(&cdSet, lib, "CoreDisplay_Display_SetUserBrightness")

	before := cdGet(id)
	target := before - 0.05
	if target < 0 {
		target = before + 0.05
	}
	if target < 0 {
		target = 0
	}
	if target > 1 {
		target = 1
	}
	cdSet(id, target)
	after := cdGet(id)
	cdSet(id, before)
	// External panels often report 1.0 and ignore writes.
	if abs64(after-target) > 0.02 {
		cdGet, cdSet = nil, nil
		return false
	}
	return true
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
