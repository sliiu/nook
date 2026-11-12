// Package power is a switch control: on or off. Wi-Fi uses it.
package power

// Backend reads and writes one switch.
type Backend interface {
	// Available is false when this machine has no API for it. On then returns
	// false, and Set changes nothing.
	Available() bool
	On() (bool, error)
	Set(on bool) error
}
