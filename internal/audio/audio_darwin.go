//go:build darwin

// Default-output scalar volume via CoreAudio.
//
// kAudioDevicePropertyVolumeScalar is 0.0–1.0 on the main element. A device
// with no hardware knob uses the virtual main volume ('vmvc'). Scope is
// output only for v1. A muted device reads as 0. A non-zero write clears
// mute. Mute itself does not write a new level. A device with neither knob
// is not an error: it reads as 0 with Adjustable false, and a volume or mute
// write on it changes nothing.
package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>

static OSStatus vol_get(AudioObjectID id, const AudioObjectPropertyAddress *addr,
		UInt32 *size, void *data) {
	return AudioObjectGetPropertyData(id, addr, 0, NULL, size, data);
}

static OSStatus vol_set(AudioObjectID id, const AudioObjectPropertyAddress *addr,
		UInt32 size, const void *data) {
	return AudioObjectSetPropertyData(id, addr, 0, NULL, size, data);
}

static int vol_has(AudioObjectID id, const AudioObjectPropertyAddress *addr) {
	return AudioObjectHasProperty(id, addr) ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	audioSystem = C.AudioObjectID(C.kAudioObjectSystemObject)
	elementMain = C.AudioObjectPropertyElement(C.kAudioObjectPropertyElementMain)
	scopeGlobal = C.AudioObjectPropertyScope(C.kAudioObjectPropertyScopeGlobal)
	scopeOutput = C.AudioObjectPropertyScope(C.kAudioDevicePropertyScopeOutput)

	selDefaultOutput = C.AudioObjectPropertySelector(C.kAudioHardwarePropertyDefaultOutputDevice)
	selVolumeScalar  = C.AudioObjectPropertySelector(C.kAudioDevicePropertyVolumeScalar)
	selMute          = C.AudioObjectPropertySelector(C.kAudioDevicePropertyMute)
	// 'vmvc'. Virtual main volume when the device has no hardware knob.
	selVirtualVolume = C.AudioObjectPropertySelector(0x766d7663)
)

type darwinBackend struct{}

// New opens the CoreAudio output backend. Fails when there is no default
// output device.
func New() (Backend, error) {
	if _, err := defaultOutput(); err != nil {
		return nil, err
	}
	return darwinBackend{}, nil
}

func (darwinBackend) Level() (Level, error) {
	device, err := defaultOutput()
	if err != nil {
		return Level{}, err
	}
	addr, err := volumeAddress(device)
	if err != nil {
		return Level{Percent: 0, Muted: false, Adjustable: false}, nil
	}
	if readMute(device) {
		return Level{Percent: 0, Muted: true, Adjustable: true}, nil
	}
	scalar, err := readF32(device, addr)
	if err != nil {
		return Level{}, err
	}
	return Level{Percent: ScalarToPercent(scalar), Muted: false, Adjustable: true}, nil
}

func (darwinBackend) Set(percent int) error {
	percent = ClampPercent(percent)
	device, err := defaultOutput()
	if err != nil {
		return err
	}
	addr, err := volumeAddress(device)
	if err != nil {
		return nil
	}
	if err := writeF32(device, addr, PercentToScalar(percent)); err != nil {
		return err
	}
	if percent > 0 {
		return writeMute(device, false)
	}
	return nil
}

func (darwinBackend) SetMute(muted bool) error {
	device, err := defaultOutput()
	if err != nil {
		return err
	}
	return writeMute(device, muted)
}

func (darwinBackend) DefaultKey() (*uint64, error) {
	device, err := defaultOutput()
	if err != nil {
		return nil, nil
	}
	key := uint64(device)
	return &key, nil
}

func addr(selector C.AudioObjectPropertySelector, scope C.AudioObjectPropertyScope) C.AudioObjectPropertyAddress {
	return C.AudioObjectPropertyAddress{
		mSelector: selector,
		mScope:    scope,
		mElement:  elementMain,
	}
}

func statusErr(what string, status C.OSStatus) error {
	return fmt.Errorf("CoreAudio %s failed (%d)", what, int(status))
}

func has(device C.AudioObjectID, address C.AudioObjectPropertyAddress) bool {
	return C.vol_has(device, &address) != 0
}

func defaultOutput() (C.AudioObjectID, error) {
	address := addr(selDefaultOutput, scopeGlobal)
	var device C.AudioObjectID
	size := C.UInt32(C.sizeof_AudioObjectID)
	status := C.vol_get(audioSystem, &address, &size, unsafe.Pointer(&device))
	if status != 0 {
		return 0, statusErr("default output", status)
	}
	if device == 0 {
		return 0, fmt.Errorf("no default output device")
	}
	return device, nil
}

func volumeAddress(device C.AudioObjectID) (C.AudioObjectPropertyAddress, error) {
	main := addr(selVolumeScalar, scopeOutput)
	if has(device, main) {
		return main, nil
	}
	virtual := addr(selVirtualVolume, scopeOutput)
	if has(device, virtual) {
		return virtual, nil
	}
	return C.AudioObjectPropertyAddress{}, fmt.Errorf("default output device has no scalar volume")
}

func readF32(device C.AudioObjectID, address C.AudioObjectPropertyAddress) (float32, error) {
	var value C.Float32
	size := C.UInt32(C.sizeof_Float32)
	status := C.vol_get(device, &address, &size, unsafe.Pointer(&value))
	if status != 0 {
		return 0, statusErr("read volume", status)
	}
	return float32(value), nil
}

func writeF32(device C.AudioObjectID, address C.AudioObjectPropertyAddress, value float32) error {
	cvalue := C.Float32(value)
	status := C.vol_set(device, &address, C.UInt32(C.sizeof_Float32), unsafe.Pointer(&cvalue))
	if status != 0 {
		return statusErr("write volume", status)
	}
	return nil
}

func readMute(device C.AudioObjectID) bool {
	address := addr(selMute, scopeOutput)
	if !has(device, address) {
		return false
	}
	var muted C.UInt32
	size := C.UInt32(C.sizeof_UInt32)
	status := C.vol_get(device, &address, &size, unsafe.Pointer(&muted))
	if status != 0 {
		return false
	}
	return muted != 0
}

func writeMute(device C.AudioObjectID, muted bool) error {
	address := addr(selMute, scopeOutput)
	if !has(device, address) {
		return nil
	}
	var value C.UInt32
	if muted {
		value = 1
	}
	status := C.vol_set(device, &address, C.UInt32(C.sizeof_UInt32), unsafe.Pointer(&value))
	if status != 0 {
		return statusErr("write mute", status)
	}
	return nil
}
