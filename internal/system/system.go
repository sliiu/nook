// Package system is a choice control: one selected id from a list.
// Appearance (light, dark, auto) uses it.
package system

// Option is one entry the user can pick.
type Option struct {
	ID    string
	Label string
}

// Backend reads and writes one choice.
type Backend interface {
	// Available is false when this machine has no API for it. State then
	// returns an empty selection, and Select changes nothing.
	Available() bool
	// State returns the selected id and the options, in display order.
	State() (selected string, options []Option, err error)
	// Select picks an id. An id that is not in the list is an error.
	Select(id string) error
}
