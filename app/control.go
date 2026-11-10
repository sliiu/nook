// Package app is the Control bridge the flyout and settings pages call.
package app

import (
	"sync"
	"time"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
	"pixel-control-center/internal/prefs"
	"pixel-control-center/internal/scenes"
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

// BrightnessRow is what GetBrightness hands back.
type BrightnessRow struct {
	Level     int  `json:"level"`
	Available bool `json:"available"`
}

// SystemState is the dual-axis snapshot for scene boot.
type SystemState struct {
	Volume     int  `json:"volume"`
	Brightness int  `json:"brightness"`
	Available  bool `json:"brightnessAvailable"`
}

type axisSlot struct {
	seen   int
	expect *int
	ready  bool
}

// Control is bound as window.mygo "Control". Methods run on their own
// goroutines; fields shared with the watcher take mu.
type Control struct {
	mu         sync.Mutex
	audio      audio.Backend
	brightness brightness.Backend
	shell      Shell
	volume     axisSlot
	bright     axisSlot
	device     *uint64
	dragging   bool
	hideAt     time.Time
	hiding     bool
	scenes     *scenes.Store
	onScenes   func()
	onSize     func(w, h int)
	prefs      *prefs.Store
	onLang     func()
}

// NewControl wires backends and the tray shell. brightness may be nil; then
// brightness reads as unavailable.
func NewControl(audioBackend audio.Backend, brightBackend brightness.Backend, shell Shell) *Control {
	c := &Control{audio: audioBackend, brightness: brightBackend, shell: shell}
	if level, err := audioBackend.Level(); err == nil {
		c.volume.seen = level.Percent
		c.volume.ready = true
	}
	if key, err := audioBackend.DefaultKey(); err == nil {
		c.device = key
	}
	if brightBackend != nil && brightBackend.Available() {
		if level, err := brightBackend.Level(); err == nil {
			c.bright.seen = level.Percent
			c.bright.ready = true
		}
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
	c.volume.seen = level.Percent
	c.volume.ready = true
	if c.device == nil {
		if key, keyErr := c.audio.DefaultKey(); keyErr == nil {
			c.device = key
		}
	}
	c.mu.Unlock()
	return LevelRow{Level: level.Percent, Muted: level.Muted, Adjustable: level.Adjustable}, nil
}

// SetVolume sets the default output. A non-zero write unmutes.
func (c *Control) SetVolume(level int) error {
	level = audio.ClampPercent(level)
	if err := c.audio.Set(level); err != nil {
		return err
	}
	c.mu.Lock()
	expected := level
	c.volume.expect = &expected
	c.volume.seen = level
	c.volume.ready = true
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
		c.volume.expect = &zero
		c.volume.seen = 0
		c.volume.ready = true
		c.mu.Unlock()
	}
	return nil
}

// GetBrightness returns built-in display brightness. Available is false when
// the backend cannot drive the panel.
func (c *Control) GetBrightness() (BrightnessRow, error) {
	if c.brightness == nil || !c.brightness.Available() {
		return BrightnessRow{Level: 0, Available: false}, nil
	}
	level, err := c.brightness.Level()
	if err != nil {
		return BrightnessRow{}, err
	}
	c.mu.Lock()
	c.bright.seen = level.Percent
	c.bright.ready = true
	c.mu.Unlock()
	return BrightnessRow{Level: level.Percent, Available: level.Available}, nil
}

// SetBrightness sets built-in brightness. No-op when unavailable.
func (c *Control) SetBrightness(level int) error {
	if c.brightness == nil || !c.brightness.Available() {
		return nil
	}
	level = brightness.ClampPercent(level)
	if err := c.brightness.Set(level); err != nil {
		return err
	}
	c.mu.Lock()
	expected := level
	c.bright.expect = &expected
	c.bright.seen = level
	c.bright.ready = true
	c.mu.Unlock()
	return nil
}

// GetState returns volume and brightness together for scene start.
func (c *Control) GetState() (SystemState, error) {
	vol, err := c.GetVolume()
	if err != nil {
		return SystemState{}, err
	}
	bright, err := c.GetBrightness()
	if err != nil {
		return SystemState{}, err
	}
	return SystemState{
		Volume:     vol.Level,
		Brightness: bright.Level,
		Available:  bright.Available,
	}, nil
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

// Watch polls volume and brightness until stop is closed.
// Package-level so mygo.Bind does not expose a channel-typed method.
func Watch(c *Control, stop <-chan struct{}) {
	if c == nil {
		return
	}
	c.watch(stop)
}

func (c *Control) watch(stop <-chan struct{}) {
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
			c.volume.seen = level.Percent
			c.volume.ready = true
			if c.device == nil {
				if key, keyErr := c.audio.DefaultKey(); keyErr == nil {
					c.device = key
				}
			}
		}
		if c.brightness != nil && c.brightness.Available() {
			if level, err := c.brightness.Level(); err == nil {
				c.bright.seen = level.Percent
				c.bright.ready = true
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
	volume     *int
	brightness *int
}

func (c *Control) takeExternal() (external, bool) {
	var out external
	changed := false

	if vol, ok := c.takeVolumeExternal(); ok {
		v := vol
		out.volume = &v
		changed = true
	}
	if bright, ok := c.takeBrightnessExternal(); ok {
		b := bright
		out.brightness = &b
		changed = true
	}
	return out, changed
}

func (c *Control) takeVolumeExternal() (int, bool) {
	level, err := c.audio.Level()
	if err != nil {
		return 0, false
	}
	device, _ := c.audio.DefaultKey()
	if !c.volume.ready {
		c.volume.seen = level.Percent
		c.device = device
		c.volume.ready = true
		return 0, false
	}
	levelChanged := level.Percent != c.volume.seen
	deviceChanged := !sameKey(device, c.device)
	if !levelChanged && !deviceChanged {
		return 0, false
	}
	c.device = device
	if levelChanged {
		if c.volume.expect != nil && *c.volume.expect == level.Percent {
			c.volume.seen = level.Percent
			if !deviceChanged {
				return 0, false
			}
		} else {
			c.volume.expect = nil
			c.volume.seen = level.Percent
			return level.Percent, true
		}
	}
	if deviceChanged && c.device != nil {
		return level.Percent, true
	}
	return 0, false
}

func (c *Control) takeBrightnessExternal() (int, bool) {
	if c.brightness == nil || !c.brightness.Available() {
		return 0, false
	}
	level, err := c.brightness.Level()
	if err != nil || !level.Available {
		return 0, false
	}
	if !c.bright.ready {
		c.bright.seen = level.Percent
		c.bright.ready = true
		return 0, false
	}
	if level.Percent == c.bright.seen {
		return 0, false
	}
	if c.bright.expect != nil && *c.bright.expect == level.Percent {
		c.bright.seen = level.Percent
		return 0, false
	}
	c.bright.expect = nil
	c.bright.seen = level.Percent
	return level.Percent, true
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
			StateChanged.Emit(panel, StateChange{
				Source:     "external",
				Volume:     change.volume,
				Brightness: change.brightness,
			})
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
