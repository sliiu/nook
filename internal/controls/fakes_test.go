package controls

import (
	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
	"pixel-control-center/internal/devices"
	"pixel-control-center/internal/keyboard"
	"pixel-control-center/internal/media"
	"pixel-control-center/internal/power"
	"pixel-control-center/internal/system"
)

type fakeAudio struct {
	level audio.Level
	sets  []int
	mutes []bool
	err   error
}

func (f *fakeAudio) Level() (audio.Level, error) {
	if f.err != nil {
		return audio.Level{}, f.err
	}
	return f.level, nil
}
func (f *fakeAudio) Set(percent int) error {
	f.sets = append(f.sets, percent)
	f.level.Percent = percent
	if percent > 0 {
		f.level.Muted = false
	}
	return nil
}
func (f *fakeAudio) SetMute(muted bool) error {
	f.mutes = append(f.mutes, muted)
	f.level.Muted = muted
	if muted {
		f.level.Percent = 0
	}
	return nil
}
func (f *fakeAudio) DefaultKey() (*uint64, error) { return nil, nil }

type fakeBright struct {
	level     brightness.Level
	available bool
	sets      []int
}

func (f *fakeBright) Level() (brightness.Level, error) {
	f.level.Available = f.available
	return f.level, nil
}
func (f *fakeBright) Set(percent int) error {
	f.sets = append(f.sets, percent)
	f.level.Percent = percent
	return nil
}
func (f *fakeBright) Available() bool { return f.available }

type fakeKeyboard struct {
	level     keyboard.Level
	available bool
	sets      []int
}

func (f *fakeKeyboard) Level() (keyboard.Level, error) {
	f.level.Available = f.available
	return f.level, nil
}
func (f *fakeKeyboard) Set(percent int) error {
	f.sets = append(f.sets, percent)
	f.level.Percent = percent
	return nil
}
func (f *fakeKeyboard) Available() bool { return f.available }

type fakeChoice struct {
	selected  string
	options   []system.Option
	available bool
	picked    []string
}

func (f *fakeChoice) Available() bool { return f.available }
func (f *fakeChoice) State() (string, []system.Option, error) {
	return f.selected, f.options, nil
}
func (f *fakeChoice) Select(id string) error {
	f.picked = append(f.picked, id)
	f.selected = id
	return nil
}

type fakeSwitch struct {
	on        bool
	available bool
	sets      []bool
}

func (f *fakeSwitch) Available() bool   { return f.available }
func (f *fakeSwitch) On() (bool, error) { return f.on, nil }
func (f *fakeSwitch) Set(on bool) error {
	f.sets = append(f.sets, on)
	f.on = on
	return nil
}

type fakeMedia struct {
	available bool
	sent      []media.Command
}

func (f *fakeMedia) Available() bool { return f.available }
func (f *fakeMedia) Send(command media.Command) error {
	f.sent = append(f.sent, command)
	return nil
}

// The fakes satisfy the backend interfaces.
var (
	_ devices.Backend = (*fakeChoice)(nil)
	_ system.Backend  = (*fakeChoice)(nil)
	_ power.Backend   = (*fakeSwitch)(nil)
	_ media.Backend   = (*fakeMedia)(nil)
)
