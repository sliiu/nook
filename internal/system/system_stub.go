//go:build !darwin

package system

type stub struct{}

// NewAppearance returns an unavailable backend outside macOS.
func NewAppearance() (Backend, error) { return stub{}, nil }

func (stub) Available() bool { return false }
func (stub) State() (string, []Option, error) {
	return "", nil, nil
}
func (stub) Select(string) error { return nil }
