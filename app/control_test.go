package app

import (
	"testing"
	"time"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
)

type fakeAudio struct {
	level audio.Level
	sets  []int
	mutes []bool
}

func (f *fakeAudio) Level() (audio.Level, error) { return f.level, nil }
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
func (f *fakeAudio) DefaultKey() (*uint64, error) {
	key := uint64(1)
	return &key, nil
}

type fakeBright struct {
	level     brightness.Level
	sets      []int
	available bool
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

func TestSetVolumeClampsAndExpects(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 40, Adjustable: true}}
	c := NewControl(backend, nil, nil)
	if err := c.SetVolume(150); err != nil {
		t.Fatal(err)
	}
	if backend.sets[0] != 100 {
		t.Fatalf("set %d, want 100", backend.sets[0])
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.volume.expect == nil || *c.volume.expect != 100 || c.volume.seen != 100 {
		t.Fatalf("expect/seen not updated")
	}
}

func TestTakeExternalSuppressesExpect(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 55, Adjustable: true}}
	c := NewControl(backend, nil, nil)
	_ = c.SetVolume(55)
	c.mu.Lock()
	_, ok := c.takeExternal()
	c.mu.Unlock()
	if ok {
		t.Fatal("expected echo suppressed")
	}
}

func TestTakeExternalAnnouncesForeignChange(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 20, Adjustable: true}}
	c := NewControl(backend, nil, nil)
	backend.level.Percent = 80
	c.mu.Lock()
	change, ok := c.takeExternal()
	c.mu.Unlock()
	if !ok || change.volume == nil || *change.volume != 80 {
		t.Fatalf("got ok=%v change=%+v", ok, change)
	}
}

func TestBrightnessRoundTripAndExpect(t *testing.T) {
	audioBackend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	bright := &fakeBright{level: brightness.Level{Percent: 40, Available: true}, available: true}
	c := NewControl(audioBackend, bright, nil)
	if err := c.SetBrightness(70); err != nil {
		t.Fatal(err)
	}
	row, err := c.GetBrightness()
	if err != nil || row.Level != 70 || !row.Available {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	c.mu.Lock()
	_, ok := c.takeExternal()
	c.mu.Unlock()
	if ok {
		t.Fatal("brightness echo should be suppressed")
	}
	bright.level.Percent = 10
	c.mu.Lock()
	change, ok := c.takeExternal()
	c.mu.Unlock()
	if !ok || change.brightness == nil || *change.brightness != 10 {
		t.Fatalf("got ok=%v change=%+v", ok, change)
	}
}

func TestGetState(t *testing.T) {
	audioBackend := &fakeAudio{level: audio.Level{Percent: 33, Adjustable: true}}
	bright := &fakeBright{level: brightness.Level{Percent: 66, Available: true}, available: true}
	c := NewControl(audioBackend, bright, nil)
	state, err := c.GetState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Volume != 33 || state.Brightness != 66 || !state.Available {
		t.Fatalf("state=%+v", state)
	}
}

func TestUnavailableBrightnessNoop(t *testing.T) {
	audioBackend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	bright := &fakeBright{available: false}
	c := NewControl(audioBackend, bright, nil)
	if err := c.SetBrightness(80); err != nil {
		t.Fatal(err)
	}
	if len(bright.sets) != 0 {
		t.Fatal("Set should no-op when unavailable")
	}
	row, err := c.GetBrightness()
	if err != nil || row.Available || row.Level != 0 {
		t.Fatalf("row=%+v err=%v", row, err)
	}
}

func TestNoteIdleArmShortHide(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	c := NewControl(backend, nil, nil)
	c.NoteIdle()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hiding || c.hideAt.Before(time.Now()) {
		t.Fatal("hide timer not armed")
	}
}

func TestNoteIdleSkippedWhileDragging(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	c := NewControl(backend, nil, nil)
	c.NoteDrag(true)
	c.NoteIdle()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hiding {
		t.Fatal("hide should not arm while dragging")
	}
}
