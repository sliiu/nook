//go:build darwin

// Keyboard backlight via the private CoreBrightness framework.
//
// KeyboardBrightnessClient has no public header. The methods below are the
// ones the class exports on macOS 26: brightnessForKeyboard:, setBrightness:
// forKeyboard: and copyKeyboardBacklightIDs. The first keyboard is used. The
// client is one shared object and the framework expects its calls on the main
// thread.
package keyboard

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const coreBrightnessPath = "/System/Library/PrivateFrameworks/CoreBrightness.framework/CoreBrightness"

var (
	loadOnce sync.Once
	client   objc.ID

	selIDs = objc.RegisterName("copyKeyboardBacklightIDs")
	selGet = objc.RegisterName("brightnessForKeyboard:")
	selSet = objc.RegisterName("setBrightness:forKeyboard:")
)

type darwinBackend struct{ ok bool }

// New loads the keyboard client. It always succeeds; Available is false when
// the class is missing or this keyboard has no backlight.
func New() (Backend, error) {
	loadOnce.Do(load)
	return darwinBackend{ok: client != 0}, nil
}

func load() {
	if _, err := purego.Dlopen(coreBrightnessPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		return
	}
	class := objc.GetClass("KeyboardBrightnessClient")
	if class == 0 {
		return
	}
	client = objc.ID(class).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
}

func (b darwinBackend) Available() bool {
	return b.ok && keyboardID() != 0
}

func (b darwinBackend) Level() (Level, error) {
	id := keyboardID()
	if !b.ok || id == 0 {
		return Level{}, nil
	}
	var value float32
	client.Send(selGet, id, &value)
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return Level{Percent: int(value*100 + 0.5), Available: true}, nil
}

func (b darwinBackend) Set(percent int) error {
	id := keyboardID()
	if !b.ok || id == 0 {
		return nil
	}
	percent = ClampPercent(percent)
	var ok bool
	client.Send(selSet, float32(percent)/100, id, &ok)
	if !ok {
		return fmt.Errorf("keyboard: set backlight failed")
	}
	return nil
}

// keyboardID is the first backlight the client reports, or 0 when there is none.
func keyboardID() uint64 {
	if client == 0 {
		return 0
	}
	list := objc.ID(client.Send(selIDs))
	if list == 0 {
		return 0
	}
	defer list.Send(objc.RegisterName("release"))
	count := uint64(list.Send(objc.RegisterName("count")))
	if count == 0 {
		return 0
	}
	number := objc.ID(list.Send(objc.RegisterName("objectAtIndex:"), uint64(0)))
	if number == 0 {
		return 0
	}
	var value uint64
	number.Send(objc.RegisterName("getValue:size:"), unsafe.Pointer(&value), uint64(unsafe.Sizeof(value)))
	return value
}
