// Package prefs stores app preferences (settings.json) and resolves the UI
// language. "auto" follows the system language.
package prefs

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Supported UI languages. Anything else falls back to English.
var Supported = []string{"zh", "en", "ja"}

const Auto = "auto"

type data struct {
	Lang string `json:"lang"`
}

// Store is safe for concurrent use.
type Store struct {
	mu   sync.Mutex
	path string
	d    data
}

// Open loads path; a missing or corrupt file means defaults.
func Open(path string) *Store {
	s := &Store{path: path, d: data{Lang: Auto}}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s.d)
	}
	if !valid(s.d.Lang) {
		s.d.Lang = Auto
	}
	return s
}

func valid(l string) bool {
	if l == Auto {
		return true
	}
	for _, v := range Supported {
		if v == l {
			return true
		}
	}
	return false
}

// Setting returns the stored choice: "auto", "zh", "en" or "ja".
func (s *Store) Setting() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d.Lang
}

// Resolved returns the language actually in use.
func (s *Store) Resolved() string {
	if l := s.Setting(); l != Auto {
		return l
	}
	return SystemLang()
}

// SetLang stores the choice. Unknown values are ignored as "auto".
func (s *Store) SetLang(l string) error {
	if !valid(l) {
		l = Auto
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.Lang = l
	raw, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// SystemLang maps the system's preferred language to a supported one.
func SystemLang() string {
	tag := ""
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
			tag = string(out)
		}
	}
	if tag == "" {
		for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
			if v := os.Getenv(k); v != "" {
				tag = v
				break
			}
		}
	}
	return Match(tag)
}

// Match picks the first supported language mentioned in tag (e.g. the output
// of `defaults read -g AppleLanguages`, or a LANG value).
func Match(tag string) string {
	t := strings.ToLower(tag)
	best, at := "en", len(t)+1
	for _, l := range Supported {
		if i := strings.Index(t, l); i >= 0 && i < at {
			best, at = l, i
		}
	}
	return best
}
