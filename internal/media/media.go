// Package media sends media-key commands. It does not read what is playing.
package media

// Command is one media key. The values match the control ids.
type Command string

const (
	Previous Command = "media.previous"
	Toggle   Command = "media.toggle"
	Next     Command = "media.next"
)

// Backend sends one command. Available is false when the platform has no API.
type Backend interface {
	Available() bool
	Send(Command) error
}
