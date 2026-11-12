// Package devices lists output devices and picks the default one.
package devices

import "pixel-control-center/internal/system"

// Backend reads and writes the default output device.
type Backend interface {
	Available() bool
	// State returns the selected device id and every output device.
	State() (selected string, options []system.Option, err error)
	Select(id string) error
}
