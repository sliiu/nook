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

// Hardware is the set of platform backends a catalog is built from. A nil
// backend is listed and reads as unavailable.
type Hardware struct {
	Output     audio.Backend
	Input      audio.Backend
	Display    brightness.Backend
	Keyboard   keyboard.Backend
	Devices    devices.Backend
	Media      media.Backend
	Appearance system.Backend
	WiFi       power.Backend
}

// Desktop is the control catalog for one machine.
func Desktop(h Hardware) []Control {
	return []Control{
		FromAudio(h.Output),
		FromBrightness(h.Display),
		FromInput(h.Input),
		FromDevices(h.Devices),
		mediaAction(h.Media, specMediaPrevious),
		mediaAction(h.Media, specMediaToggle),
		mediaAction(h.Media, specMediaNext),
		FromKeyboard(h.Keyboard),
		FromSystem(h.Appearance, specAppearance),
		FromPower(h.WiFi, specWiFi),
	}
}

var (
	specOutput        = Spec{ID: OutputLevel, Kind: KindLevel, Mute: true}
	specDisplay       = Spec{ID: DisplayLevel, Kind: KindLevel}
	specInput         = Spec{ID: InputLevel, Kind: KindLevel, Mute: true}
	specDevice        = Spec{ID: OutputDevice, Kind: KindChoice}
	specMediaPrevious = Spec{ID: MediaPrevious, Kind: KindAction}
	specMediaToggle   = Spec{ID: MediaToggle, Kind: KindAction}
	specMediaNext     = Spec{ID: MediaNext, Kind: KindAction}
	specKeyboard      = Spec{ID: KeyboardLevel, Kind: KindLevel}
	specAppearance    = Spec{ID: Appearance, Kind: KindChoice}
	specWiFi          = Spec{ID: WiFiPower, Kind: KindSwitch}
)

// FromAudio adapts the default-output backend. A nil backend is unavailable.
func FromAudio(b audio.Backend) Control {
	return fromLevel(b, specOutput)
}

// FromInput adapts the default-input backend, the microphone. A nil backend
// is unavailable.
func FromInput(b audio.Backend) Control {
	return fromLevel(b, specInput)
}

func fromLevel(b audio.Backend, spec Spec) Control {
	if b == nil {
		return missing{spec}
	}
	return levelControl{b, spec}
}

// FromBrightness adapts the built-in display backend. A nil backend, or one
// that reports itself unavailable, reads as unavailable. A write in that
// state is a no-op, matching the brightness backend.
func FromBrightness(b brightness.Backend) Control {
	if b == nil {
		return missing{specDisplay}
	}
	return displayLevel{b}
}

// FromKeyboard adapts the keyboard backlight. Same shape as the display: a
// write while unavailable changes nothing.
func FromKeyboard(b keyboard.Backend) Control {
	if b == nil {
		return missing{specKeyboard}
	}
	return keyboardLevel{b}
}

// FromDevices adapts the output-device list.
func FromDevices(b devices.Backend) Control {
	if b == nil {
		return missing{specDevice}
	}
	return deviceChoice{b}
}

// FromPower adapts a switch such as Wi-Fi.
func FromPower(b power.Backend, spec Spec) Control {
	if b == nil {
		return missing{spec}
	}
	return powerSwitch{b, spec}
}

// FromSystem adapts a choice such as appearance.
func FromSystem(b system.Backend, spec Spec) Control {
	if b == nil {
		return missing{spec}
	}
	return systemChoice{b, spec}
}

func mediaAction(b media.Backend, spec Spec) Control {
	if b == nil {
		return missing{spec}
	}
	return mediaControl{b, spec}
}

type levelControl struct {
	b    audio.Backend
	spec Spec
}

func (o levelControl) Spec() Spec { return o.spec }

func (o levelControl) Available() bool { return true }

func (o levelControl) Read() (Value, error) {
	lv, err := o.b.Level()
	if err != nil {
		return Value{}, err
	}
	return Value{
		Available:  true,
		Percent:    lv.Percent,
		Muted:      lv.Muted,
		Adjustable: lv.Adjustable,
	}, nil
}

func (o levelControl) Apply(u Update) error {
	if u.Selected != nil || u.On != nil || (!o.spec.Mute && u.Muted != nil) {
		return ErrUnsupported
	}
	// Percent first, then mute, so an explicit mute survives a non-zero set
	// (a non-zero volume write unmutes on its own).
	if u.Percent != nil {
		if err := o.b.Set(audio.ClampPercent(*u.Percent)); err != nil {
			return err
		}
	}
	if u.Muted != nil {
		if err := o.b.SetMute(*u.Muted); err != nil {
			return err
		}
	}
	return nil
}

type displayLevel struct{ b brightness.Backend }

func (d displayLevel) Spec() Spec { return specDisplay }

func (d displayLevel) Available() bool { return d.b.Available() }

func (d displayLevel) Read() (Value, error) {
	if !d.b.Available() {
		return Value{}, nil
	}
	lv, err := d.b.Level()
	if err != nil {
		return Value{}, err
	}
	return Value{
		Available:  lv.Available,
		Percent:    lv.Percent,
		Adjustable: lv.Available,
	}, nil
}

func (d displayLevel) Apply(u Update) error {
	if u.Muted != nil || u.Selected != nil || u.On != nil {
		return ErrUnsupported
	}
	if u.Percent == nil || !d.b.Available() {
		return nil
	}
	return d.b.Set(brightness.ClampPercent(*u.Percent))
}

type keyboardLevel struct{ b keyboard.Backend }

func (k keyboardLevel) Spec() Spec { return specKeyboard }

func (k keyboardLevel) Available() bool { return k.b.Available() }

func (k keyboardLevel) Read() (Value, error) {
	if !k.b.Available() {
		return Value{}, nil
	}
	lv, err := k.b.Level()
	if err != nil {
		return Value{}, err
	}
	return Value{
		Available:  lv.Available,
		Percent:    lv.Percent,
		Adjustable: lv.Available,
	}, nil
}

func (k keyboardLevel) Apply(u Update) error {
	if !k.b.Available() {
		return ErrUnavailable
	}
	if u.Muted != nil || u.Selected != nil || u.On != nil {
		return ErrUnsupported
	}
	if u.Percent == nil {
		return nil
	}
	return k.b.Set(keyboard.ClampPercent(*u.Percent))
}

type deviceChoice struct{ b devices.Backend }

func (d deviceChoice) Spec() Spec { return specDevice }

func (d deviceChoice) Available() bool { return d.b.Available() }

func (d deviceChoice) Read() (Value, error) {
	if !d.b.Available() {
		return Value{}, nil
	}
	selected, options, err := d.b.State()
	if err != nil {
		return Value{}, err
	}
	return Value{
		Available:  true,
		Adjustable: len(options) > 0,
		Selected:   selected,
		Options:    toOptions(options),
	}, nil
}

func (d deviceChoice) Apply(u Update) error {
	if !d.b.Available() {
		return ErrUnavailable
	}
	if u.Percent != nil || u.Muted != nil || u.On != nil {
		return ErrUnsupported
	}
	if u.Selected == nil {
		return nil
	}
	return d.b.Select(*u.Selected)
}

type powerSwitch struct {
	b    power.Backend
	spec Spec
}

func (p powerSwitch) Spec() Spec { return p.spec }

func (p powerSwitch) Available() bool { return p.b.Available() }

func (p powerSwitch) Read() (Value, error) {
	if !p.b.Available() {
		return Value{}, nil
	}
	on, err := p.b.On()
	if err != nil {
		return Value{}, err
	}
	return Value{Available: true, Adjustable: true, On: on}, nil
}

func (p powerSwitch) Apply(u Update) error {
	if !p.b.Available() {
		return ErrUnavailable
	}
	if u.Percent != nil || u.Muted != nil || u.Selected != nil {
		return ErrUnsupported
	}
	if u.On == nil {
		return nil
	}
	return p.b.Set(*u.On)
}

type systemChoice struct {
	b    system.Backend
	spec Spec
}

func (s systemChoice) Spec() Spec { return s.spec }

func (s systemChoice) Available() bool { return s.b.Available() }

func (s systemChoice) Read() (Value, error) {
	if !s.b.Available() {
		return Value{}, nil
	}
	selected, options, err := s.b.State()
	if err != nil {
		return Value{}, err
	}
	return Value{
		Available:  true,
		Adjustable: len(options) > 0,
		Selected:   selected,
		Options:    toOptions(options),
	}, nil
}

func (s systemChoice) Apply(u Update) error {
	if !s.b.Available() {
		return ErrUnavailable
	}
	if u.Percent != nil || u.Muted != nil || u.On != nil {
		return ErrUnsupported
	}
	if u.Selected == nil {
		return nil
	}
	return s.b.Select(*u.Selected)
}

type mediaControl struct {
	b    media.Backend
	spec Spec
}

func (m mediaControl) Spec() Spec { return m.spec }

func (m mediaControl) Available() bool { return m.b.Available() }

func (m mediaControl) Read() (Value, error) {
	return Value{Available: m.b.Available()}, nil
}

func (m mediaControl) Apply(Update) error {
	if !m.b.Available() {
		return ErrUnavailable
	}
	return m.b.Send(media.Command(m.spec.ID))
}

func toOptions(in []system.Option) []Option {
	out := make([]Option, len(in))
	for i, option := range in {
		out[i] = Option{ID: option.ID, Label: option.Label}
	}
	return out
}

// missing is a catalog entry with no platform backend yet.
type missing struct{ spec Spec }

func (m missing) Spec() Spec      { return m.spec }
func (m missing) Available() bool { return false }
func (m missing) Read() (Value, error) {
	return Value{}, nil
}
func (m missing) Apply(Update) error { return ErrUnavailable }
