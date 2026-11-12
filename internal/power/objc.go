//go:build darwin

package power

import "unsafe"

// cstring copies a C string into a Go string. The pointer comes from an
// Objective-C object, so it is read through an unsafe conversion.
func cstring(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	start := unsafe.Add(unsafe.Pointer(nil), ptr)
	length := 0
	for *(*byte)(unsafe.Add(start, length)) != 0 && length < 4096 {
		length++
	}
	return string(unsafe.Slice((*byte)(start), length))
}
