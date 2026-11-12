//go:build darwin

// Output devices via CoreAudio.
//
// The list is every device that has an output stream. The selection is the
// system default output device. A device id is the AudioObjectID printed in
// decimal, which is only stable until the device is unplugged.
package devices

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>

static OSStatus dev_get(AudioObjectID id, const AudioObjectPropertyAddress *addr,
		UInt32 *size, void *data) {
	return AudioObjectGetPropertyData(id, addr, 0, NULL, size, data);
}

static OSStatus dev_set(AudioObjectID id, const AudioObjectPropertyAddress *addr,
		UInt32 size, const void *data) {
	return AudioObjectSetPropertyData(id, addr, 0, NULL, size, data);
}

static int dev_has(AudioObjectID id, const AudioObjectPropertyAddress *addr) {
	return AudioObjectHasProperty(id, addr) ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"unsafe"

	"pixel-control-center/internal/system"
)

const (
	audioSystem = C.AudioObjectID(C.kAudioObjectSystemObject)
	elementMain = C.AudioObjectPropertyElement(C.kAudioObjectPropertyElementMain)
	scopeGlobal = C.AudioObjectPropertyScope(C.kAudioObjectPropertyScopeGlobal)
	scopeOutput = C.AudioObjectPropertyScope(C.kAudioDevicePropertyScopeOutput)

	selDevices       = C.AudioObjectPropertySelector(C.kAudioHardwarePropertyDevices)
	selDefaultOutput = C.AudioObjectPropertySelector(C.kAudioHardwarePropertyDefaultOutputDevice)
	selStreams       = C.AudioObjectPropertySelector(C.kAudioDevicePropertyStreams)
	selName          = C.AudioObjectPropertySelector(C.kAudioObjectPropertyName)
)

type darwinBackend struct{}

// New opens the CoreAudio device backend. It succeeds even when no output
// device is plugged in; State then reports an empty list.
func New() (Backend, error) { return darwinBackend{}, nil }

func (darwinBackend) Available() bool { return true }

func (darwinBackend) State() (string, []system.Option, error) {
	defaultDevice, err := readID(audioSystem, addr(selDefaultOutput, scopeGlobal))
	if err != nil {
		return "", nil, err
	}
	ids, err := deviceIDs()
	if err != nil {
		return "", nil, err
	}
	var options []system.Option
	for _, id := range ids {
		if !hasOutput(id) {
			continue
		}
		label, err := deviceName(id)
		if err != nil || label == "" {
			label = fmt.Sprintf("Output %d", uint32(id))
		}
		options = append(options, system.Option{ID: formatID(id), Label: label})
	}
	return formatID(defaultDevice), options, nil
}

func (darwinBackend) Select(id string) error {
	device, err := parseID(id)
	if err != nil {
		return err
	}
	address := addr(selDefaultOutput, scopeGlobal)
	status := C.dev_set(audioSystem, &address, C.UInt32(C.sizeof_AudioObjectID), unsafe.Pointer(&device))
	if status != 0 {
		return fmt.Errorf("CoreAudio set default output failed (%d)", int(status))
	}
	return nil
}

func addr(selector C.AudioObjectPropertySelector, scope C.AudioObjectPropertyScope) C.AudioObjectPropertyAddress {
	return C.AudioObjectPropertyAddress{
		mSelector: selector,
		mScope:    scope,
		mElement:  elementMain,
	}
}

func readID(object C.AudioObjectID, address C.AudioObjectPropertyAddress) (C.AudioObjectID, error) {
	var id C.AudioObjectID
	size := C.UInt32(C.sizeof_AudioObjectID)
	status := C.dev_get(object, &address, &size, unsafe.Pointer(&id))
	if status != 0 {
		return 0, fmt.Errorf("CoreAudio read device failed (%d)", int(status))
	}
	return id, nil
}

func deviceIDs() ([]C.AudioObjectID, error) {
	address := addr(selDevices, scopeGlobal)
	var size C.UInt32
	status := C.AudioObjectGetPropertyDataSize(audioSystem, &address, 0, nil, &size)
	if status != 0 {
		return nil, fmt.Errorf("CoreAudio device list size failed (%d)", int(status))
	}
	count := int(size) / C.sizeof_AudioObjectID
	if count == 0 {
		return nil, nil
	}
	ids := make([]C.AudioObjectID, count)
	status = C.dev_get(audioSystem, &address, &size, unsafe.Pointer(&ids[0]))
	if status != 0 {
		return nil, fmt.Errorf("CoreAudio device list failed (%d)", int(status))
	}
	return ids, nil
}

func hasOutput(device C.AudioObjectID) bool {
	address := addr(selStreams, scopeOutput)
	if C.dev_has(device, &address) == 0 {
		return false
	}
	var size C.UInt32
	if C.AudioObjectGetPropertyDataSize(device, &address, 0, nil, &size) != 0 {
		return false
	}
	return size > 0
}

func deviceName(device C.AudioObjectID) (string, error) {
	address := addr(selName, scopeGlobal)
	var value C.CFStringRef
	size := C.UInt32(unsafe.Sizeof(value))
	status := C.dev_get(device, &address, &size, unsafe.Pointer(&value))
	if status != 0 || value == 0 {
		return "", fmt.Errorf("CoreAudio device name failed (%d)", int(status))
	}
	defer C.CFRelease(C.CFTypeRef(value))
	buf := make([]C.char, 512)
	if C.CFStringGetCString(value, &buf[0], C.CFIndex(len(buf)), C.kCFStringEncodingUTF8) == 0 {
		return "", fmt.Errorf("CoreAudio device name was not UTF-8")
	}
	return C.GoString(&buf[0]), nil
}

func formatID(id C.AudioObjectID) string {
	return strconv.FormatUint(uint64(id), 10)
}

func parseID(text string) (C.AudioObjectID, error) {
	n, err := strconv.ParseUint(text, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("devices: unknown device %q", text)
	}
	return C.AudioObjectID(n), nil
}
