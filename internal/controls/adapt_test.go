package controls

import (
	"errors"
	"testing"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
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

func ptr[T any](v T) *T { return &v }

func TestDesktopCatalog(t *testing.T) {
	out := &fakeAudio{level: audio.Level{Percent: 40, Adjustable: true}}
	in := &fakeAudio{level: audio.Level{Percent: 70, Muted: true, Adjustable: true}}
	display := &fakeBright{level: brightness.Level{Percent: 60}, available: true}
	list := Desktop(out, in, display)

	want := []Spec{
		specOutput, specDisplay, specInput, specDevice,
		specMediaPrevious, specMediaToggle, specMediaNext,
	}
	if len(list) != len(want) {
		t.Fatalf("catalog %d, want %d", len(list), len(want))
	}
	for i, spec := range want {
		if got := list[i].Spec(); got != spec {
			t.Fatalf("catalog[%d] = %+v, want %+v", i, got, spec)
		}
	}

	if !Find(list, OutputLevel).Available() || !Find(list, DisplayLevel).Available() || !Find(list, InputLevel).Available() {
		t.Fatal("live controls should be available")
	}
	got, err := Find(list, InputLevel).Read()
	if err != nil || got.Percent != 70 || !got.Muted {
		t.Fatalf("input read %+v, %v", got, err)
	}
	for _, id := range []ID{OutputDevice, MediaToggle} {
		c := Find(list, id)
		if c.Available() {
			t.Fatalf("%s should be unavailable", id)
		}
		if err := c.Apply(Update{}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("%s Apply = %v, want ErrUnavailable", id, err)
		}
	}
}

func TestFromAudioReadAndApply(t *testing.T) {
	b := &fakeAudio{level: audio.Level{Percent: 0, Muted: true, Adjustable: true}}
	c := FromAudio(b)

	v, err := c.Read()
	if err != nil {
		t.Fatal(err)
	}
	if v.Percent != 0 || !v.Muted || !v.Adjustable {
		t.Fatalf("read %+v", v)
	}

	if err := c.Apply(Update{Percent: ptr(150), Muted: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if len(b.sets) != 1 || b.sets[0] != 100 {
		t.Fatalf("set %v, want [100]", b.sets)
	}
	if len(b.mutes) != 1 || !b.mutes[0] {
		t.Fatalf("mute %v, want [true] after the set", b.mutes)
	}

	if err := c.Apply(Update{Selected: ptr("speakers")}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("selected on a level = %v", err)
	}
}

func TestFromBrightness(t *testing.T) {
	off := &fakeBright{}
	c := FromBrightness(off)
	if c.Available() {
		t.Fatal("external-only display is unavailable")
	}
	v, err := c.Read()
	if err != nil || v.Available || v.Percent != 0 {
		t.Fatalf("read %+v, %v", v, err)
	}
	if err := c.Apply(Update{Percent: ptr(40)}); err != nil {
		t.Fatal(err)
	}
	if len(off.sets) != 0 {
		t.Fatal("unavailable brightness write should be a no-op")
	}

	on := &fakeBright{level: brightness.Level{Percent: 10}, available: true}
	c = FromBrightness(on)
	if err := c.Apply(Update{Percent: ptr(-3)}); err != nil {
		t.Fatal(err)
	}
	if on.sets[0] != 0 {
		t.Fatalf("set %d, want 0", on.sets[0])
	}
	if err := c.Apply(Update{Muted: ptr(true)}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("mute on brightness = %v", err)
	}
}

func TestNilBackendsAreMissing(t *testing.T) {
	list := Desktop(nil, nil, nil)
	for _, id := range []ID{OutputLevel, DisplayLevel, InputLevel} {
		c := Find(list, id)
		if c.Available() {
			t.Fatalf("%s nil backend should be unavailable", id)
		}
		if err := c.Apply(Update{Percent: ptr(10)}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("%s Apply = %v", id, err)
		}
	}
}
