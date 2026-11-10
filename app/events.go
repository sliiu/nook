package app

import "github.com/egoist/mygo"

// StateChanged is emitted when volume (and later brightness) moves outside a
// user drag. Watcher emissions use Source "external" so the page does not
// echo its own writes.
var StateChanged = mygo.NewEvent[StateChange]("state-changed")

// PillForm tells the page which flyout form is showing.
var PillForm = mygo.NewEvent[FormPayload]("pill-form")

// StateChange is a partial system state update.
type StateChange struct {
	Source     string `json:"source"` // "user" | "external"
	Volume     *int   `json:"volume,omitempty"`
	Brightness *int   `json:"brightness,omitempty"`
}

// FormPayload is the flyout form name.
type FormPayload struct {
	Form string `json:"form"` // "full" | "short" | "hidden"
}
