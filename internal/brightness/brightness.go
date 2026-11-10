// Package brightness reads and writes built-in display brightness.
package brightness

// Level is the current brightness reading. Available is false when the
// platform has no usable API (external-only display, stub build, private
// symbols missing). Percent is then 0 and Set is a no-op.
type Level struct {
	Percent   int
	Available bool
}

// Backend talks to the platform brightness API.
type Backend interface {
	Level() (Level, error)
	Set(percent int) error
	Available() bool
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

// PercentToUnit maps 0–100 to a 0–1 brightness scalar.
func PercentToUnit(percent int) float32 {
	return float32(ClampPercent(percent)) / 100
}

// UnitToPercent maps a 0–1 scalar to 0–100, rounded.
func UnitToPercent(unit float32) int {
	if unit < 0 {
		unit = 0
	}
	if unit > 1 {
		unit = 1
	}
	return int(unit*100 + 0.5)
}
