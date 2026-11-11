// Package scenes persists the user's scene list (scenes.json). The "home"
// scene always exists, sits first, and cannot be deleted; RestoreHome writes
// the factory page back.
package scenes

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	HomeID      = "home"
	HomeName    = "Home"
	MaxNameLen  = 40
	MaxHTMLSize = 1 << 20 // 1 MiB per scene
)

var (
	ErrNotFound   = errors.New("scene not found")
	ErrNameEmpty  = errors.New("scene name is empty")
	ErrNameLong   = errors.New("scene name is too long")
	ErrNameTaken  = errors.New("scene name already exists")
	ErrHTMLEmpty  = errors.New("scene html is empty")
	ErrHTMLLarge  = errors.New("scene html is too large")
	ErrHomeLocked = errors.New("the home scene cannot be deleted or renamed")
	ErrMarket     = errors.New("a catalog scene cannot be edited")
)

// Scene is one single-file HTML scene. Market is set only for scenes
// installed from the catalog; a local scene leaves it empty.
type Scene struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	HTML   string  `json:"html"`
	Market *Market `json:"market,omitempty"`
}

// Market records where an installed catalog scene came from. HTMLHash is the
// SHA-256 of HTML, lowercase hex, checked again on every update.
type Market struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	HTMLHash string `json:"htmlHash"`
}

type file struct {
	Current string  `json:"current"`
	Scenes  []Scene `json:"scenes"`
}

// Store is safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	path    string
	factory string
	data    file
}

// Open loads path, seeding or repairing it so home always exists. A missing
// or corrupt file is replaced by a factory-only list.
func Open(path, factoryHTML string) (*Store, error) {
	s := &Store{path: path, factory: factoryHTML}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s.data) // corrupt file: fall through to repair
	}
	s.repair()
	if err := s.save(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) repair() {
	kept := make([]Scene, 0, len(s.data.Scenes)+1)
	home := Scene{ID: HomeID, Name: HomeName, HTML: s.factory}
	seen := map[string]bool{HomeID: true}
	for _, sc := range s.data.Scenes {
		if sc.ID == HomeID {
			if strings.TrimSpace(sc.HTML) != "" {
				home.HTML = sc.HTML
			}
			continue
		}
		if sc.ID == "" || seen[sc.ID] || validate(sc.Name, sc.HTML) != nil {
			continue
		}
		if sc.Market != nil && !validMarket(sc.Market) {
			continue
		}
		seen[sc.ID] = true
		kept = append(kept, sc)
	}
	s.data.Scenes = append([]Scene{home}, kept...)
	if !seen[s.data.Current] {
		s.data.Current = HomeID
	}
}

func validate(name, html string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameEmpty
	}
	if utf8.RuneCountInString(name) > MaxNameLen {
		return ErrNameLong
	}
	if strings.TrimSpace(html) == "" {
		return ErrHTMLEmpty
	}
	if len(html) > MaxHTMLSize {
		return ErrHTMLLarge
	}
	return nil
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
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

func (s *Store) index(id string) int {
	for i, sc := range s.data.Scenes {
		if sc.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) nameTaken(name, exceptID string) bool {
	for _, sc := range s.data.Scenes {
		if sc.ID != exceptID && strings.EqualFold(sc.Name, name) {
			return true
		}
	}
	return false
}

// freeName keeps a catalog scene's name when it is free, and appends " 2", " 3"
// and so on when a local scene already has it. The catalog id is not a name
// the user chose, so a clash must not stop the install. exceptMarket is the
// catalog id being refreshed, whose own name stays available.
func (s *Store) freeName(name, exceptMarket string) string {
	exceptID := ""
	for _, sc := range s.data.Scenes {
		if sc.Market != nil && sc.Market.ID == exceptMarket {
			exceptID = sc.ID
			break
		}
	}
	if !s.nameTaken(name, exceptID) {
		return name
	}
	for n := 2; n < 1000; n++ {
		candidate := fmt.Sprintf("%s %d", name, n)
		if utf8.RuneCountInString(candidate) > MaxNameLen {
			break
		}
		if !s.nameTaken(candidate, exceptID) {
			return candidate
		}
	}
	return name
}

// List returns every scene, home first.
func (s *Store) List() []Scene {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Scene(nil), s.data.Scenes...)
}

// Current returns the selected scene.
func (s *Store) Current() Scene {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Scenes[s.index(s.data.Current)]
}

// Select makes id the scene the flyout shows.
func (s *Store) Select(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index(id) < 0 {
		return ErrNotFound
	}
	s.data.Current = id
	return s.save()
}

// Save creates a scene when id is empty, otherwise updates it. Home keeps its
// name; only its HTML can change.
func (s *Store) Save(id, name, html string) (Scene, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if id == HomeID {
		name = HomeName
	}
	if err := validate(name, html); err != nil {
		return Scene{}, err
	}
	if s.nameTaken(name, id) {
		return Scene{}, ErrNameTaken
	}
	if id == "" {
		sc := Scene{ID: newID(), Name: name, HTML: html}
		s.data.Scenes = append(s.data.Scenes, sc)
		return sc, s.save()
	}
	i := s.index(id)
	if i < 0 {
		return Scene{}, ErrNotFound
	}
	if s.data.Scenes[i].Market != nil {
		return Scene{}, ErrMarket
	}
	s.data.Scenes[i].Name = name
	s.data.Scenes[i].HTML = html
	return s.data.Scenes[i], s.save()
}

// Delete removes a scene; the selection falls back to home.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == HomeID {
		return ErrHomeLocked
	}
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	s.data.Scenes = append(s.data.Scenes[:i], s.data.Scenes[i+1:]...)
	if s.data.Current == id {
		s.data.Current = HomeID
	}
	return s.save()
}

// Install records a catalog scene, or refreshes the one already installed
// for the same market id. The caller has already checked the catalog.
func (s *Store) Install(m Market, name, html string) (Scene, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = s.freeName(strings.TrimSpace(name), m.ID)
	if err := validate(name, html); err != nil {
		return Scene{}, err
	}
	m.HTMLHash = HTMLHash(html)
	if !validMarket(&m) {
		return Scene{}, ErrMarket
	}
	for i, sc := range s.data.Scenes {
		if sc.Market != nil && sc.Market.ID == m.ID {
			s.data.Scenes[i].Name = name
			s.data.Scenes[i].HTML = html
			s.data.Scenes[i].Market = &m
			return s.data.Scenes[i], s.save()
		}
	}
	sc := Scene{ID: newID(), Name: name, HTML: html, Market: &m}
	s.data.Scenes = append(s.data.Scenes, sc)
	return sc, s.save()
}

// RestoreHome writes the factory page back into home and selects it.
func (s *Store) RestoreHome() (Scene, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(HomeID)
	s.data.Scenes[i].HTML = s.factory
	s.data.Current = HomeID
	return s.data.Scenes[i], s.save()
}

var marketIDRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)

func validMarket(m *Market) bool {
	if m == nil || !marketIDRe.MatchString(m.ID) {
		return false
	}
	if len(m.Commit) != 40 || len(m.HTMLHash) != 64 {
		return false
	}
	for _, s := range []string{m.Commit, m.HTMLHash} {
		for _, r := range s {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return false
			}
		}
	}
	if m.Version == "" || len(m.Version) > 32 {
		return false
	}
	return true
}

// HTMLHash is the lowercase hex SHA-256 of a scene page.
func HTMLHash(html string) string {
	sum := sha256.Sum256([]byte(html))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "s_" + hex.EncodeToString(b[:])
}
