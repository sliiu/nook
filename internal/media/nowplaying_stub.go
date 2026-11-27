//go:build !darwin

package media

// NowPlaying reads what is playing. Outside macOS there is no player to ask,
// so it reports stopped and no track.
func NowPlaying() (Playback, error) {
	return Playback{State: Stopped}, nil
}
