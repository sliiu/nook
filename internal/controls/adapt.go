package controls

import (
	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
)

// Desktop is the control catalog. Output volume, built-in brightness and the
// microphone use the backends when they are non-nil. The output device and
// the media keys are listed so callers can see them, and Apply returns
// ErrUnavailable until a platform backend exists.
func Desktop(output, input audio.Backend, display brightness.Backend) []Control {
	return []Control{
		FromAudio(output),
		FromBrightness(display),
		FromInput(input),
		missing{specDevice},
		missing{specMediaPrevious},
		missing{specMediaToggle},
		missing{specMediaNext},
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
	if u.Selected != nil || (!o.spec.Mute && u.Muted != nil) {
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
	if u.Muted != nil || u.Selected != nil {
		return ErrUnsupported
	}
	if u.Percent == nil || !d.b.Available() {
		return nil
	}
	return d.b.Set(brightness.ClampPercent(*u.Percent))
}

// missing is a catalog entry with no platform backend yet.
type missing struct{ spec Spec }

func (m missing) Spec() Spec      { return m.spec }
func (m missing) Available() bool { return false }
func (m missing) Read() (Value, error) {
	return Value{}, nil
}
func (m missing) Apply(Update) error { return ErrUnavailable }
