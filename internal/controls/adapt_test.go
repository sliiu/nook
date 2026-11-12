package controls

import (
	"errors"
	"testing"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/brightness"
	"pixel-control-center/internal/keyboard"
	"pixel-control-center/internal/media"
	"pixel-control-center/internal/system"
)

func ptr[T any](v T) *T { return &v }

func TestDesktopCatalog(t *testing.T) {
	out := &fakeAudio{level: audio.Level{Percent: 40, Adjustable: true}}
	in := &fakeAudio{level: audio.Level{Percent: 70, Muted: true, Adjustable: true}}
	display := &fakeBright{level: brightness.Level{Percent: 60}, available: true}
	keys := &fakeKeyboard{level: keyboard.Level{Percent: 30}, available: true}
	devs := &fakeChoice{selected: "173", available: true, options: []system.Option{{ID: "173", Label: "Speakers"}}}
	look := &fakeChoice{selected: "auto", available: true, options: []system.Option{{ID: "auto", Label: "Auto"}}}
	wifi := &fakeSwitch{on: true, available: true}
	player := &fakeMedia{available: true}
	list := Desktop(Hardware{
		Output: out, Input: in, Display: display, Keyboard: keys,
		Devices: devs, Media: player, Appearance: look, WiFi: wifi,
	})

	want := []Spec{
		specOutput, specDisplay, specInput, specDevice,
		specMediaPrevious, specMediaToggle, specMediaNext,
		specKeyboard, specAppearance, specWiFi,
	}
	if len(list) != len(want) {
		t.Fatalf("catalog %d, want %d", len(list), len(want))
	}
	for i, spec := range want {
		if got := list[i].Spec(); got != spec {
			t.Fatalf("catalog[%d] = %+v, want %+v", i, got, spec)
		}
	}

	if !Find(list, OutputLevel).Available() || !Find(list, DisplayLevel).Available() || !Find(list, InputLevel).Available() {
		t.Fatal("live controls should be available")
	}
	got, err := Find(list, InputLevel).Read()
	if err != nil || got.Percent != 70 || !got.Muted {
		t.Fatalf("input read %+v, %v", got, err)
	}
	if !Find(list, OutputDevice).Available() || !Find(list, MediaToggle).Available() {
		t.Fatal("device and media should be available when a backend is wired")
	}
}

func TestFromAudioReadAndApply(t *testing.T) {
	b := &fakeAudio{level: audio.Level{Percent: 0, Muted: true, Adjustable: true}}
	c := FromAudio(b)

	v, err := c.Read()
	if err != nil {
		t.Fatal(err)
	}
	if v.Percent != 0 || !v.Muted || !v.Adjustable {
		t.Fatalf("read %+v", v)
	}

	if err := c.Apply(Update{Percent: ptr(150), Muted: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if len(b.sets) != 1 || b.sets[0] != 100 {
		t.Fatalf("set %v, want [100]", b.sets)
	}
	if len(b.mutes) != 1 || !b.mutes[0] {
		t.Fatalf("mute %v, want [true] after the set", b.mutes)
	}

	if err := c.Apply(Update{Selected: ptr("speakers")}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("selected on a level = %v", err)
	}
}

func TestFromBrightness(t *testing.T) {
	off := &fakeBright{}
	c := FromBrightness(off)
	if c.Available() {
		t.Fatal("external-only display is unavailable")
	}
	v, err := c.Read()
	if err != nil || v.Available || v.Percent != 0 {
		t.Fatalf("read %+v, %v", v, err)
	}
	if err := c.Apply(Update{Percent: ptr(40)}); err != nil {
		t.Fatal(err)
	}
	if len(off.sets) != 0 {
		t.Fatal("unavailable brightness write should be a no-op")
	}

	on := &fakeBright{level: brightness.Level{Percent: 10}, available: true}
	c = FromBrightness(on)
	if err := c.Apply(Update{Percent: ptr(-3)}); err != nil {
		t.Fatal(err)
	}
	if on.sets[0] != 0 {
		t.Fatalf("set %d, want 0", on.sets[0])
	}
	if err := c.Apply(Update{Muted: ptr(true)}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("mute on brightness = %v", err)
	}
}

func TestNilBackendsAreMissing(t *testing.T) {
	list := Desktop(Hardware{})
	for _, id := range []ID{OutputLevel, DisplayLevel, InputLevel} {
		c := Find(list, id)
		if c.Available() {
			t.Fatalf("%s nil backend should be unavailable", id)
		}
		if err := c.Apply(Update{Percent: ptr(10)}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("%s Apply = %v", id, err)
		}
	}
}

func TestSwitchChoiceAndAction(t *testing.T) {
	wifi := &fakeSwitch{on: true, available: true}
	switchControl := FromPower(wifi, specWiFi)
	got, err := switchControl.Read()
	if err != nil || !got.On || !got.Adjustable {
		t.Fatalf("wifi read %+v, %v", got, err)
	}
	if err := switchControl.Apply(Update{On: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if len(wifi.sets) != 1 || wifi.sets[0] {
		t.Fatalf("wifi set %v, want [false]", wifi.sets)
	}
	if err := switchControl.Apply(Update{Percent: ptr(1)}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("percent on a switch = %v", err)
	}

	devs := &fakeChoice{selected: "173", available: true, options: []system.Option{{ID: "173", Label: "Speakers"}}}
	choice := FromDevices(devs)
	got, err = choice.Read()
	if err != nil || got.Selected != "173" || len(got.Options) != 1 {
		t.Fatalf("device read %+v, %v", got, err)
	}
	if err := choice.Apply(Update{Selected: ptr("56")}); err != nil {
		t.Fatal(err)
	}
	if len(devs.picked) != 1 || devs.picked[0] != "56" {
		t.Fatalf("picked %v", devs.picked)
	}

	player := &fakeMedia{available: true}
	action := mediaAction(player, specMediaNext)
	if err := action.Apply(Update{}); err != nil {
		t.Fatal(err)
	}
	if len(player.sent) != 1 || player.sent[0] != media.Next {
		t.Fatalf("sent %v, want next", player.sent)
	}

	keys := &fakeKeyboard{available: true}
	backlight := FromKeyboard(keys)
	if err := backlight.Apply(Update{Percent: ptr(140)}); err != nil {
		t.Fatal(err)
	}
	if len(keys.sets) != 1 || keys.sets[0] != 100 {
		t.Fatalf("keyboard set %v, want [100]", keys.sets)
	}
	off := &fakeKeyboard{}
	if err := FromKeyboard(off).Apply(Update{Percent: ptr(10)}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable keyboard = %v", err)
	}
}
