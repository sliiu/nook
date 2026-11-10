package app

import (
	"github.com/egoist/mygo"

	"pixel-control-center/internal/prefs"
)

// LangInfo is the language choice and what it resolves to right now.
type LangInfo struct {
	Setting  string `json:"setting"`  // auto | zh | en | ja
	Resolved string `json:"resolved"` // zh | en | ja
}

// LangChanged tells every window to re-render in the new language.
var LangChanged = mygo.NewEvent[LangInfo]("lang-changed")

// AttachPrefs wires preferences and the callback run after a language change.
func AttachPrefs(c *Control, p *prefs.Store, onChange func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prefs = p
	c.onLang = onChange
}

// GetLang returns the stored language choice and the language in use.
func (c *Control) GetLang() LangInfo {
	c.mu.Lock()
	p := c.prefs
	c.mu.Unlock()
	if p == nil {
		return LangInfo{Setting: prefs.Auto, Resolved: prefs.SystemLang()}
	}
	return LangInfo{Setting: p.Setting(), Resolved: p.Resolved()}
}

// SetLang stores the language ("auto", "zh", "en", "ja").
func (c *Control) SetLang(lang string) (LangInfo, error) {
	c.mu.Lock()
	p, fn := c.prefs, c.onLang
	c.mu.Unlock()
	if p != nil {
		if err := p.SetLang(lang); err != nil {
			return LangInfo{}, err
		}
	}
	if fn != nil {
		fn()
	}
	return c.GetLang(), nil
}
