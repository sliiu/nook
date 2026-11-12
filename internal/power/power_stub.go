//go:build !darwin

package power

type stub struct{}

// NewWiFi returns an unavailable backend outside macOS.
func NewWiFi() (Backend, error) { return stub{}, nil }

func (stub) Available() bool   { return false }
func (stub) On() (bool, error) { return false, nil }
func (stub) Set(bool) error    { return nil }
