package media

// Playback is what the system reports for the current player. It is a
// snapshot, not a subscription: the caller polls.
//
// macOS 26 answers the playback state and withholds the track. Title, Artist
// and App stay empty there, and Track is false so a caller can tell "nothing
// is known" from a player that really has no title. A later system that
// answers fills them in without a change to this shape.
type Playback struct {
	// State is playing, paused or stopped. A machine with no player reports
	// stopped.
	State State
	// Track is true when the system named what is playing. The strings are
	// empty when it did not, even if something is playing.
	Track bool
	Title string
	// Artist is the performer. It is empty when the track has none.
	Artist string
	// App is the player, such as "Music". It is empty when the system does
	// not say which one.
	App string
}

// State is the player's transport state.
type State string

const (
	// Playing is a player currently producing sound.
	Playing State = "playing"
	// Paused is a player stopped on a track it can resume.
	Paused State = "paused"
	// Stopped is no current player, or one with nothing queued.
	Stopped State = "stopped"
)

// Playing reports whether something is currently playing. It is the one fact
// every platform that has a player can answer.
func (p Playback) Playing() bool { return p.State == Playing }
