// controlsmoke reads every control and changes nothing.
//
//	go run ./cmd/controlsmoke
package main

import (
	"fmt"
	"os"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
	"pixel-control-center/internal/controls"
	"pixel-control-center/internal/devices"
	"pixel-control-center/internal/keyboard"
	"pixel-control-center/internal/media"
	"pixel-control-center/internal/power"
	"pixel-control-center/internal/system"
)

func main() {
	open := func(name string, fn func() (any, error)) any {
		v, err := fn()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
			return nil
		}
		return v
	}
	h := controls.Hardware{
		Output:     open("output", func() (any, error) { return audio.New() }).(audio.Backend),
		Input:      open("input", func() (any, error) { return audio.NewInput() }).(audio.Backend),
		Display:    open("display", func() (any, error) { return brightness.New() }).(brightness.Backend),
		Keyboard:   open("keyboard", func() (any, error) { return keyboard.New() }).(keyboard.Backend),
		Devices:    open("devices", func() (any, error) { return devices.New() }).(devices.Backend),
		Media:      open("media", func() (any, error) { return media.New() }).(media.Backend),
		Appearance: open("appearance", func() (any, error) { return system.NewAppearance() }).(system.Backend),
		WiFi:       open("wifi", func() (any, error) { return power.NewWiFi() }).(power.Backend),
	}
	failed := false
	for _, control := range controls.Desktop(h) {
		spec := control.Spec()
		value, err := control.Read()
		if err != nil {
			failed = true
			fmt.Printf("%-16s ERROR %v\n", spec.ID, err)
			continue
		}
		fmt.Printf("%-16s kind=%-7s available=%-5v adjustable=%-5v", spec.ID, spec.Kind, value.Available, value.Adjustable)
		switch spec.Kind {
		case controls.KindLevel:
			fmt.Printf(" percent=%d muted=%v", value.Percent, value.Muted)
		case controls.KindSwitch:
			fmt.Printf(" on=%v", value.On)
		case controls.KindChoice:
			fmt.Printf(" selected=%q options=%d", value.Selected, len(value.Options))
			for _, option := range value.Options {
				fmt.Printf("\n%-16s   %s  %s", "", option.ID, option.Label)
			}
		}
		fmt.Println()
	}
	if failed {
		os.Exit(1)
	}
}
