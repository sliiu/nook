//go:build darwin

package media

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Now playing state via the private MediaRemote framework.
//
// MRMediaRemoteGetNowPlayingApplicationPlaybackState and
// MRMediaRemoteGetNowPlayingInfo both take a dispatch queue and a block, and
// the block runs on that queue. Passing nil, or an NSOperationQueue, delivers
// nothing; a queue from dispatch_queue_create does. The state comes back as
// one of the values MediaRemote has used since macOS 10.12: 0 stopped,
// 1 playing, 2 paused.
//
// The info call returns the track dictionary on older systems and nil on
// macOS 26, which withholds the title, artist and app from a process that
// does not hold the now-playing entitlement. Nil is reported as no track
// rather than a failure, so the state still comes through.

const (
	stateStopped = 0
	statePlaying = 1
	statePaused  = 2
)

var (
	nowOnce sync.Once
	nowOK   bool

	stateFn func(queue unsafe.Pointer, block unsafe.Pointer)
	infoFn  func(queue unsafe.Pointer, block unsafe.Pointer)

	makeQueue func(allocator, attr unsafe.Pointer) unsafe.Pointer

	keyTitle  objc.ID
	keyArtist objc.ID
	keyRate   objc.ID
)

func loadNowPlaying() {
	lib, err := purego.Dlopen(mediaRemotePath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	core, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&stateFn, lib, "MRMediaRemoteGetNowPlayingApplicationPlaybackState")
	purego.RegisterLibFunc(&infoFn, lib, "MRMediaRemoteGetNowPlayingInfo")
	purego.RegisterLibFunc(&makeQueue, core, "dispatch_queue_create")
	keyTitle = keySymbol(lib, "kMRMediaRemoteNowPlayingInfoTitle")
	keyArtist = keySymbol(lib, "kMRMediaRemoteNowPlayingInfoArtist")
	keyRate = keySymbol(lib, "kMRMediaRemoteNowPlayingInfoPlaybackRate")
	nowOK = stateFn != nil && infoFn != nil && makeQueue != nil
}

// keySymbol reads an exported NSString constant. A missing one reads back
// empty, which drops that field rather than failing the read.
func keySymbol(lib uintptr, name string) objc.ID {
	sym, err := purego.Dlsym(lib, name)
	if err != nil {
		return 0
	}
	return *(*objc.ID)(unsafe.Add(unsafe.Pointer(nil), sym))
}

// NowPlaying reads the current player. The first call resolves the symbols;
// when they are gone it reports stopped, the same answer as no player.
func NowPlaying() (Playback, error) {
	nowOnce.Do(loadNowPlaying)
	if !nowOK {
		return Playback{State: Stopped}, nil
	}
	queue := makeQueue(nil, nil)
	if queue == nil {
		return Playback{}, fmt.Errorf("media: no dispatch queue")
	}

	state, err := callState(queue)
	if err != nil {
		return Playback{}, err
	}
	out := Playback{State: state}

	info, err := callInfo(queue)
	if err != nil {
		return Playback{}, err
	}
	if info != 0 {
		out.Title = textFor(info, keyTitle)
		out.Artist = textFor(info, keyArtist)
		out.Track = out.Title != "" || out.Artist != ""
		// A rate above zero is playing even when the state call said otherwise,
		// which happens in the moment between a command and the state catching up.
		if rateFor(info, keyRate) > 0 && out.State != Playing {
			out.State = Playing
		}
	}
	return out, nil
}

func callState(queue unsafe.Pointer) (State, error) {
	done := make(chan int32, 1)
	block := objc.NewBlock(func(_ objc.Block, state int32) {
		done <- state
	})
	defer block.Release()
	stateFn(queue, unsafe.Add(unsafe.Pointer(nil), block))
	select {
	case state := <-done:
		return decodeState(state), nil
	case <-time.After(2 * time.Second):
		return "", fmt.Errorf("media: playback state was not answered")
	}
}

func callInfo(queue unsafe.Pointer) (objc.ID, error) {
	done := make(chan objc.ID, 1)
	block := objc.NewBlock(func(_ objc.Block, info objc.ID) {
		if info != 0 {
			info.Send(selRetain)
		}
		done <- info
	})
	defer block.Release()
	infoFn(queue, unsafe.Add(unsafe.Pointer(nil), block))
	select {
	case info := <-done:
		return info, nil
	case <-time.After(2 * time.Second):
		return 0, fmt.Errorf("media: now playing info was not answered")
	}
}

func decodeState(state int32) State {
	switch state {
	case statePlaying:
		return Playing
	case statePaused:
		return Paused
	default:
		return Stopped
	}
}

var (
	selObject = objc.RegisterName("objectForKey:")
	selUTF8   = objc.RegisterName("UTF8String")
	selRetain = objc.RegisterName("retain")
	selDouble = objc.RegisterName("doubleValue")
)

func textFor(info, key objc.ID) string {
	if info == 0 || key == 0 {
		return ""
	}
	value := info.Send(selObject, key)
	if value == 0 {
		return ""
	}
	return cstring(value.Send(selUTF8))
}

func rateFor(info, key objc.ID) float64 {
	if info == 0 || key == 0 {
		return 0
	}
	value := info.Send(selObject, key)
	if value == 0 {
		return 0
	}
	return objc.Send[float64](value, selDouble)
}

func cstring(ptr objc.ID) string {
	if ptr == 0 {
		return ""
	}
	start := unsafe.Add(unsafe.Pointer(nil), uintptr(ptr))
	length := 0
	for *(*byte)(unsafe.Add(start, length)) != 0 && length < 1024 {
		length++
	}
	return string(unsafe.Slice((*byte)(start), length))
}
