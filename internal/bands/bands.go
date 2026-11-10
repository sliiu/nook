// Package bands maps 0–100 levels onto the frozen 4-step Home axes.
package bands

import "fmt"

// Of returns the 0–3 band for a 0–100 level.
// 0–24 → 0, 25–49 → 1, 50–74 → 2, 75–100 → 3.
func Of(level int) int {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	switch {
	case level < 25:
		return 0
	case level < 50:
		return 1
	case level < 75:
		return 2
	default:
		return 3
	}
}

// ID formats the frozen state id V{vx}B{by}.
func ID(volume, brightness int) string {
	return fmt.Sprintf("V%dB%d", Of(volume), Of(brightness))
}
