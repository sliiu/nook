// Package app is the Control bridge the flyout and settings pages call.
package app

import (
	"sync"
	"time"

	"pixel-control-center/internal/audio"
)

const (
	hideAfter = 1600 * time.Millisecond
	pollEvery = 250 * time.Millisecond
)

// LevelRow is what GetVolume hands back.
type LevelRow struct {
	Level      int  `json:"level"`
	Muted      bool `json:"muted"`
	Adjustable bool `json:"adjustable"`
}

// Control is bound as window.mygo "Control". Methods run on their own
// goroutines; fields shared with the watcher take mu.
type Control struct {
	mu       sync.Mutex
	audio    audio.Backend
	shell    Shell
	seen     int
	expect   *int
	device   *uint64
	ready    bool
	dragging bool
	hideAt   time.Time
	hiding   bool
}

// NewControl wires an audio backend and the tray shell.
func NewControl(backend audio.Backend, shell Shell) *Control {
	c := &Control{audio: backend, shell: shell}
	if level, err := backend.Level(); err == nil {
		c.seen = level.Percent
		c.ready = true
	}
	if key, err := backend.DefaultKey(); err == nil {
		c.device = key
	}
	return c
}

// GetVolume returns the default output percent, 0–100. A muted device reads
// as 0.
func (c *Control) GetVolume() (LevelRow, error) {
	level, err := c.audio.Level()
	if err != nil {
		return LevelRow{}, err
	}
	c.mu.Lock()
	c.absorb(level.Percent)
	c.mu.Unlock()
	return LevelRow{Level: level.Percent, Muted: level.Muted, Adjustable: level.Adjustable}, nil
}

func (c *Control) absorb(percent int) {
	c.seen = percent
	c.ready = true
	if c.device == nil {
		if key, err := c.audio.DefaultKey(); err == nil {
			c.device = key
		}
	}
}

// SetVolume sets the default output. A non-zero write unmutes.
func (c *Control) SetVolume(level int) error {
	level = audio.ClampPercent(level)
	if err := c.audio.Set(level); err != nil {
		return err
	}
	c.mu.Lock()
	expected := level
	c.expect = &expected
	c.seen = level
	c.ready = true
	c.mu.Unlock()
	return nil
}

// SetMute mutes or unmutes without writing a new level.
func (c *Control) SetMute(muted bool) error {
	if err := c.audio.SetMute(muted); err != nil {
		return err
	}
	if muted {
		c.mu.Lock()
		zero := 0
		c.expect = &zero
		c.seen = 0
		c.ready = true
		c.mu.Unlock()
	}
	return nil
}

// NoteDrag is the pointer on the pad/slider. While down, own writes are the
// volume and the flyout stays up.
func (c *Control) NoteDrag(dragging bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dragging = dragging
	if dragging {
		c.hiding = false
	}
}

// NoteIdle asks the short form to hide once the page has been still. The
// full form stays until the next left click.
func (c *Control) NoteIdle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dragging {
		return
	}
	if c.shell != nil && c.shell.Form() == "full" {
		return
	}
	c.hideAt = time.Now().Add(hideAfter)
	c.hiding = true
}

// Watch polls output volume until stop is closed.
func (c *Control) Watch(stop <-chan struct{}) {
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.poll()
		}
	}
}

func (c *Control) poll() {
	c.mu.Lock()
	if c.dragging {
		if level, err := c.audio.Level(); err == nil {
			c.seen = level.Percent
			c.ready = true
			if c.device == nil {
				if key, keyErr := c.audio.DefaultKey(); keyErr == nil {
					c.device = key
				}
			}
		}
		c.mu.Unlock()
		return
	}
	change, ok := c.takeExternal()
	hide := c.hiding && !c.hideAt.IsZero() && !time.Now().Before(c.hideAt)
	form := ""
	if c.shell != nil {
		form = c.shell.Form()
	}
	c.mu.Unlock()

	if ok {
		c.showExternal(change, form)
	}
	if hide {
		c.mu.Lock()
		still := c.hiding && !c.dragging
		c.mu.Unlock()
		if still && c.shell != nil && c.shell.Form() != "full" {
			c.shell.Hide()
		}
	}
}

type external struct {
	level int
}

func (c *Control) takeExternal() (external, bool) {
	level, err := c.audio.Level()
	if err != nil {
		return external{}, false
	}
	device, _ := c.audio.DefaultKey()
	if !c.ready {
		c.seen = level.Percent
		c.device = device
		c.ready = true
		return external{}, false
	}
	levelChanged := level.Percent != c.seen
	deviceChanged := !sameKey(device, c.device)
	if !levelChanged && !deviceChanged {
		return external{}, false
	}
	c.device = device
	if levelChanged {
		if c.expect != nil && *c.expect == level.Percent {
			c.seen = level.Percent
			if !deviceChanged {
				return external{}, false
			}
		} else {
			c.expect = nil
			c.seen = level.Percent
			return external{level: level.Percent}, true
		}
	}
	// Device-only change: still announce so the page can refresh.
	if deviceChanged && c.device != nil {
		return external{level: level.Percent}, true
	}
	return external{}, false
}

func sameKey(a, b *uint64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (c *Control) showExternal(change external, form string) {
	if c.shell != nil {
		if panel := c.shell.Panel(); panel != nil {
			vol := change.level
			StateChanged.Emit(panel, StateChange{Source: "external", Volume: &vol})
		}
	}
	if form == "full" {
		return
	}
	c.mu.Lock()
	c.hideAt = time.Now().Add(hideAfter)
	c.hiding = true
	c.mu.Unlock()
	if c.shell != nil {
		c.shell.ShowShort()
	}
}
