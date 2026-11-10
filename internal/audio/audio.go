// Package audio reads and writes the default output volume.
package audio

// Level is the default output reading. A muted device reads as Percent 0
// with Muted true. Adjustable is false when the device has no volume knob;
// Set and SetMute then change nothing.
type Level struct {
	Percent    int
	Muted      bool
	Adjustable bool
}

// Backend talks to the platform volume API. Implementations are
// constructed with New.
type Backend interface {
	Level() (Level, error)
	Set(percent int) error
	SetMute(muted bool) error
	// DefaultKey is a stable-enough id for the current default device so the
	// watcher can tell a device swap from a level change. Nil when unknown.
	DefaultKey() (*uint64, error)
}

// ClampPercent keeps a level in 0–100.
func ClampPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// PercentToScalar maps 0–100 to a CoreAudio-style 0–1 scalar.
func PercentToScalar(percent int) float32 {
	return float32(ClampPercent(percent)) / 100
}

// ScalarToPercent maps a 0–1 scalar to 0–100, rounded.
func ScalarToPercent(scalar float32) int {
	if scalar < 0 {
		scalar = 0
	}
	if scalar > 1 {
		scalar = 1
	}
	return int(scalar*100 + 0.5)
}
