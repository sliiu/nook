package app

import (
	"testing"
	"time"

	"pixel-control-center/internal/audio"
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

func TestSetVolumeClampsAndExpects(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 40, Adjustable: true}}
	c := NewControl(backend, nil)
	if err := c.SetVolume(150); err != nil {
		t.Fatal(err)
	}
	if backend.sets[0] != 100 {
		t.Fatalf("set %d, want 100", backend.sets[0])
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.expect == nil || *c.expect != 100 || c.seen != 100 {
		t.Fatalf("expect/seen not updated")
	}
}

func TestTakeExternalSuppressesExpect(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 55, Adjustable: true}}
	c := NewControl(backend, nil)
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
	c := NewControl(backend, nil)
	backend.level.Percent = 80
	c.mu.Lock()
	change, ok := c.takeExternal()
	c.mu.Unlock()
	if !ok || change.level != 80 {
		t.Fatalf("got ok=%v change=%+v", ok, change)
	}
}

func TestNoteIdleArmShortHide(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	c := NewControl(backend, nil)
	c.NoteIdle()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hiding || c.hideAt.Before(time.Now()) {
		t.Fatal("hide timer not armed")
	}
}

func TestNoteIdleSkippedWhileDragging(t *testing.T) {
	backend := &fakeAudio{level: audio.Level{Percent: 10, Adjustable: true}}
	c := NewControl(backend, nil)
	c.NoteDrag(true)
	c.NoteIdle()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hiding {
		t.Fatal("hide should not arm while dragging")
	}
}
