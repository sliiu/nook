//go:build darwin

// Default-device scalar volume via CoreAudio.
//
// New is the default output device, scope output. NewInput is the default
// input device (the microphone), scope input.
//
// kAudioDevicePropertyVolumeScalar is 0.0–1.0 on the main element. A device
// with no hardware knob uses the virtual main volume ('vmvc'). A muted device
// reads as 0. A non-zero write clears mute. Mute itself does not write a new
// level. A device with neither knob is not an error: it reads as 0 with
// Adjustable false, and a volume or mute write on it changes nothing.
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
	scopeInput  = C.AudioObjectPropertyScope(C.kAudioDevicePropertyScopeInput)

	selDefaultOutput = C.AudioObjectPropertySelector(C.kAudioHardwarePropertyDefaultOutputDevice)
	selDefaultInput  = C.AudioObjectPropertySelector(C.kAudioHardwarePropertyDefaultInputDevice)
	selVolumeScalar  = C.AudioObjectPropertySelector(C.kAudioDevicePropertyVolumeScalar)
	selMute          = C.AudioObjectPropertySelector(C.kAudioDevicePropertyMute)
	// 'vmvc'. Virtual main volume when the device has no hardware knob.
	selVirtualVolume = C.AudioObjectPropertySelector(0x766d7663)
)

// darwinBackend is one default device. scope selects output or input on it.
type darwinBackend struct {
	scope          C.AudioObjectPropertyScope
	label          string
	deviceSelector C.AudioObjectPropertySelector
}

func outputBackend() darwinBackend {
	return darwinBackend{scope: scopeOutput, label: "output", deviceSelector: selDefaultOutput}
}

func inputBackend() darwinBackend {
	return darwinBackend{scope: scopeInput, label: "input", deviceSelector: selDefaultInput}
}

// New opens the CoreAudio output backend. Fails when there is no default
// output device.
func New() (Backend, error) {
	b := outputBackend()
	if _, err := b.device(); err != nil {
		return nil, err
	}
	return b, nil
}

// NewInput opens the CoreAudio input backend (the default microphone).
// Fails when there is no default input device. A device with no volume knob
// still returns a backend; Level then reports Adjustable false.
func NewInput() (Backend, error) {
	b := inputBackend()
	if _, err := b.device(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b darwinBackend) Level() (Level, error) {
	device, err := b.device()
	if err != nil {
		return Level{}, err
	}
	address, err := b.volumeAddress(device)
	if err != nil {
		return Level{Percent: 0, Muted: false, Adjustable: false}, nil
	}
	if b.readMute(device) {
		return Level{Percent: 0, Muted: true, Adjustable: true}, nil
	}
	scalar, err := readF32(device, address)
	if err != nil {
		return Level{}, err
	}
	return Level{Percent: ScalarToPercent(scalar), Muted: false, Adjustable: true}, nil
}

func (b darwinBackend) Set(percent int) error {
	percent = ClampPercent(percent)
	device, err := b.device()
	if err != nil {
		return err
	}
	address, err := b.volumeAddress(device)
	if err != nil {
		return nil
	}
	if err := writeF32(device, address, PercentToScalar(percent)); err != nil {
		return err
	}
	if percent > 0 {
		return b.writeMute(device, false)
	}
	return nil
}

func (b darwinBackend) SetMute(muted bool) error {
	device, err := b.device()
	if err != nil {
		return err
	}
	return b.writeMute(device, muted)
}

func (b darwinBackend) DefaultKey() (*uint64, error) {
	device, err := b.device()
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

func (b darwinBackend) device() (C.AudioObjectID, error) {
	address := addr(b.deviceSelector, scopeGlobal)
	var device C.AudioObjectID
	size := C.UInt32(C.sizeof_AudioObjectID)
	status := C.vol_get(audioSystem, &address, &size, unsafe.Pointer(&device))
	if status != 0 {
		return 0, statusErr("default "+b.label, status)
	}
	if device == 0 {
		return 0, fmt.Errorf("no default %s device", b.label)
	}
	return device, nil
}

func (b darwinBackend) volumeAddress(device C.AudioObjectID) (C.AudioObjectPropertyAddress, error) {
	main := addr(selVolumeScalar, b.scope)
	if has(device, main) {
		return main, nil
	}
	virtual := addr(selVirtualVolume, b.scope)
	if has(device, virtual) {
		return virtual, nil
	}
	return C.AudioObjectPropertyAddress{}, fmt.Errorf("default %s device has no scalar volume", b.label)
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

func (b darwinBackend) readMute(device C.AudioObjectID) bool {
	address := addr(selMute, b.scope)
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

func (b darwinBackend) writeMute(device C.AudioObjectID, muted bool) error {
	address := addr(selMute, b.scope)
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
