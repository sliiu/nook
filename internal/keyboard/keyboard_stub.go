//go:build !darwin

package keyboard

type stub struct{}

// New returns an unavailable backend outside macOS.
func New() (Backend, error) { return stub{}, nil }

func (stub) Level() (Level, error) { return Level{}, nil }
func (stub) Set(int) error         { return nil }
func (stub) Available() bool       { return false }
