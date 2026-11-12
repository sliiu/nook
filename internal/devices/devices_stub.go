//go:build !darwin

package devices

import "pixel-control-center/internal/system"

type stub struct{}

// New returns an unavailable backend outside macOS.
func New() (Backend, error) { return stub{}, nil }

func (stub) Available() bool { return false }
func (stub) State() (string, []system.Option, error) {
	return "", nil, nil
}
func (stub) Select(string) error { return nil }
