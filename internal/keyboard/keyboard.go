// Package keyboard reads and writes the keyboard backlight.
package keyboard

// Level is the current backlight. Available is false when the machine has no
// backlight or the platform has no API. Percent is then 0 and Set is a no-op.
type Level struct {
	Percent   int
	Available bool
}

// Backend talks to the platform keyboard backlight.
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
