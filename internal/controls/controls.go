// Package controls is the cross-platform control surface.
//
// Volume, display brightness, the microphone and the keyboard backlight are
// levels. Wi-Fi is a switch. The output device and the appearance are choices.
// Media keys are actions. Platform backends adapt into Control; a control with
// no backend on this machine is still listed and reports itself unavailable.
package controls

import "errors"

// Kind is the shape of a control. It does not change per platform.
type Kind string

const (
	// KindLevel is a 0–100 scalar. Spec.Mute marks an independent mute bit
	// (output volume, microphone). Display brightness is a level without one.
	KindLevel Kind = "level"
	// KindSwitch is on or off (Wi-Fi).
	KindSwitch Kind = "switch"
	// KindChoice is one selected id from a list (the default output device).
	KindChoice Kind = "choice"
	// KindAction is a one-shot. Apply itself is the impulse (media keys).
	KindAction Kind = "action"
)

// ID is a stable control name, the same on every platform.
type ID string

const (
	OutputLevel   ID = "output.level"
	DisplayLevel  ID = "display.level"
	InputLevel    ID = "input.level"
	OutputDevice  ID = "output.device"
	MediaPrevious ID = "media.previous"
	MediaToggle   ID = "media.toggle"
	MediaNext     ID = "media.next"
	KeyboardLevel ID = "keyboard.level"
	Appearance    ID = "appearance"
	WiFiPower     ID = "wifi.power"
)

var (
	// ErrUnavailable is Apply on a control this machine cannot drive.
	ErrUnavailable = errors.New("controls: not available on this system")
	// ErrUnsupported is a field that does not belong to this control's kind.
	ErrUnsupported = errors.New("controls: field does not apply to this control")
)

// Spec is the static shape. Whether the machine can drive it is Available,
// which changes with hardware.
type Spec struct {
	ID   ID
	Kind Kind
	// Mute is true when a level also has a mute bit that is not the level.
	Mute bool
}

// Option is one entry of a choice control.
type Option struct {
	ID    string
	Label string
}

// Value is a reading. Fields that do not belong to the kind stay zero.
type Value struct {
	Available bool
	Percent   int
	Muted     bool
	// Adjustable is false when a level can be read but has no knob
	// (no default output device). Writes then change nothing.
	Adjustable bool
	On         bool
	Selected   string
	Options    []Option
}

// Update is a partial write. Nil fields are left unchanged.
// KindAction ignores it; Apply itself is the impulse.
type Update struct {
	Percent  *int
	Muted    *bool
	On       *bool
	Selected *string
}

// Control is the one surface volume, brightness and the later controls share.
type Control interface {
	Spec() Spec
	Available() bool
	Read() (Value, error)
	Apply(Update) error
}

// Find returns the control with id, or nil.
func Find(list []Control, id ID) Control {
	for _, c := range list {
		if c != nil && c.Spec().ID == id {
			return c
		}
	}
	return nil
}
