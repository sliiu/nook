//go:build darwin

// Media keys via the private MediaRemote framework.
//
// MRMediaRemoteSendCommand posts one command to the current player. The
// command numbers are the ones MediaRemote has used since macOS 10.12:
// 4 play/pause, 5 next, 6 previous. The symbol is resolved at runtime, so a
// system update that drops it reports the control unavailable instead of
// failing to launch.
package media

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const mediaRemotePath = "/System/Library/PrivateFrameworks/MediaRemote.framework/MediaRemote"

const (
	commandToggle   uint32 = 4
	commandNext     uint32 = 5
	commandPrevious uint32 = 6
)

var (
	loadOnce sync.Once
	send     func(command uint32, userInfo unsafe.Pointer, queue unsafe.Pointer, callback unsafe.Pointer) bool
)

type darwinBackend struct{ ok bool }

// New resolves the send symbol. It always succeeds; Available is false when
// the symbol is gone.
func New() (Backend, error) {
	loadOnce.Do(load)
	return darwinBackend{ok: send != nil}, nil
}

func load() {
	lib, err := purego.Dlopen(mediaRemotePath, purego.RTLD_LAZY)
	if err != nil {
		return
	}
	var fn func(command uint32, userInfo unsafe.Pointer, queue unsafe.Pointer, callback unsafe.Pointer) bool
	purego.RegisterLibFunc(&fn, lib, "MRMediaRemoteSendCommand")
	send = fn
}

func (b darwinBackend) Available() bool { return b.ok }

func (b darwinBackend) Send(command Command) error {
	if !b.ok {
		return fmt.Errorf("media: not available on this system")
	}
	code, ok := codes[command]
	if !ok {
		return fmt.Errorf("media: unknown command %q", command)
	}
	if !send(code, nil, nil, nil) {
		return fmt.Errorf("media: command %s was not delivered", command)
	}
	return nil
}

var codes = map[Command]uint32{
	Toggle:   commandToggle,
	Next:     commandNext,
	Previous: commandPrevious,
}
