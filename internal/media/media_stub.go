//go:build !darwin

package media

type stub struct{}

// New returns an unavailable backend outside macOS.
func New() (Backend, error) { return stub{}, nil }

func (stub) Available() bool    { return false }
func (stub) Send(Command) error { return nil }
