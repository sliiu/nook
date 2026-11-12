//go:build darwin

// Wi-Fi power via the public CoreWLAN framework.
//
// CWInterface.powerOn reads the default interface. setPower:error: is the
// write, and macOS has required a Location Services style entitlement for it
// since macOS 15. This app does not have that entitlement, so a write is
// attempted and a refusal comes back as an error. The read still works.
package power

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const coreWLANPath = "/System/Library/Frameworks/CoreWLAN.framework/CoreWLAN"

var (
	wifiOnce sync.Once
	wifiOK   bool

	selSharedClient = objc.RegisterName("sharedWiFiClient")
	selInterface    = objc.RegisterName("interface")
	selPowerOn      = objc.RegisterName("powerOn")
	selSetPower     = objc.RegisterName("setPower:error:")
)

type wifiBackend struct{ ok bool }

// NewWiFi loads CoreWLAN. It always succeeds; Available is false when the
// machine has no Wi-Fi interface.
func NewWiFi() (Backend, error) {
	wifiOnce.Do(loadWiFi)
	return wifiBackend{ok: wifiOK}, nil
}

func loadWiFi() {
	if _, err := purego.Dlopen(coreWLANPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		return
	}
	wifiOK = objc.GetClass("CWWiFiClient") != 0
}

func (b wifiBackend) Available() bool {
	return b.ok && wifiInterface() != 0
}

func (b wifiBackend) On() (bool, error) {
	iface := wifiInterface()
	if !b.ok || iface == 0 {
		return false, nil
	}
	return iface.Send(selPowerOn) != 0, nil
}

func (b wifiBackend) Set(on bool) error {
	iface := wifiInterface()
	if !b.ok || iface == 0 {
		return nil
	}
	var failure objc.ID
	ok := iface.Send(selSetPower, on, &failure) != 0
	if ok {
		return nil
	}
	if failure != 0 {
		text := failure.Send(objc.RegisterName("localizedDescription")).Send(objc.RegisterName("UTF8String"))
		return fmt.Errorf("wifi: %s", cstring(uintptr(text)))
	}
	return fmt.Errorf("wifi: set power failed")
}

func wifiInterface() objc.ID {
	class := objc.GetClass("CWWiFiClient")
	if class == 0 {
		return 0
	}
	client := objc.ID(class).Send(selSharedClient)
	if client == 0 {
		return 0
	}
	return objc.ID(client.Send(selInterface))
}
