package controls

import "pixel-control-center/internal/power"

// FakeSwitch is an exported power backend for the app package tests.
type FakeSwitch struct {
	OnValue     bool
	IsAvailable bool
	Sets        []bool
}

func (f *FakeSwitch) Available() bool   { return f.IsAvailable }
func (f *FakeSwitch) On() (bool, error) { return f.OnValue, nil }
func (f *FakeSwitch) Set(on bool) error {
	f.Sets = append(f.Sets, on)
	f.OnValue = on
	return nil
}

var _ power.Backend = (*FakeSwitch)(nil)
