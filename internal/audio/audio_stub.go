//go:build !darwin

package audio

import "errors"

type stub struct{}

// New returns a no-op backend outside macOS. Volume reads as 0 and is not
// adjustable; writes succeed without changing anything.
func New() (Backend, error) {
	return stub{}, nil
}

func (stub) Level() (Level, error) {
	return Level{Percent: 0, Muted: false, Adjustable: false}, nil
}

func (stub) Set(int) error { return nil }

func (stub) SetMute(bool) error { return nil }

func (stub) DefaultKey() (*uint64, error) {
	return nil, errors.New("audio: no default device on this platform")
}
