package app

import "github.com/egoist/mygo"

// AppInfo is the running build, for the About page.
type AppInfo struct {
	Version string `json:"version"`
}

// UpdateInfo is what the last update check found. Available is empty when
// the app is current, before a check, or after one that failed.
type UpdateInfo struct {
	Current   string `json:"current"`
	Available string `json:"available"`
}

// UpdateChanged tells windows to redraw the update row.
var UpdateChanged = mygo.NewEvent[UpdateInfo]("update-changed")

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
