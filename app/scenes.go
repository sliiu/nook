package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"

	"pixel-control-center/internal/market"
	"pixel-control-center/internal/scenes"
)

// SceneSummary is one row of the scene list (no HTML body). MarketID is
// empty for a scene the user wrote or imported.
type SceneSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsHome   bool   `json:"isHome"`
	Current  bool   `json:"current"`
	MarketID string `json:"marketId"`
	Version  string `json:"version"`
}

// SceneView is a scene with its HTML.
type SceneView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HTML     string `json:"html"`
	IsHome   bool   `json:"isHome"`
	MarketID string `json:"marketId"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
}

// ImportResult is what the file dialog picked. Nothing is saved yet.
type ImportResult struct {
	Name      string `json:"name"`
	HTML      string `json:"html"`
	Cancelled bool   `json:"cancelled"`
}

// ScenesChanged tells windows to refresh the list and the flyout to remount.
var ScenesChanged = mygo.NewEvent[ScenesPayload]("scenes-changed")

// ScenesPayload carries the selected scene id.
type ScenesPayload struct {
	Current string `json:"current"`
}

// AttachScenes wires the scene store. onChange runs after any mutation (tray
// menu rebuild, event fan-out). Package-level so mygo.Bind does not expose a
// method with non-JSON parameters.
func AttachScenes(c *Control, store *scenes.Store, onChange func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scenes = store
	c.onScenes = onChange
}

func (c *Control) store() (*scenes.Store, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scenes == nil {
		return nil, errors.New("scenes are not available")
	}
	return c.scenes, nil
}

func (c *Control) changed() {
	c.mu.Lock()
	fn := c.onScenes
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func view(sc scenes.Scene) SceneView {
	v := SceneView{ID: sc.ID, Name: sc.Name, HTML: sc.HTML, IsHome: sc.ID == scenes.HomeID}
	if sc.Market != nil {
		v.MarketID, v.Version, v.Commit = sc.Market.ID, sc.Market.Version, sc.Market.Commit
	}
	return v
}

func summary(sc scenes.Scene, current string) SceneSummary {
	row := SceneSummary{ID: sc.ID, Name: sc.Name, IsHome: sc.ID == scenes.HomeID, Current: sc.ID == current}
	if sc.Market != nil {
		row.MarketID, row.Version = sc.Market.ID, sc.Market.Version
	}
	return row
}

// catalog is the only source InstallFromLink reads. Tests replace it.
var catalog = &market.Client{}

// ListScenes returns every scene, home first.
func (c *Control) ListScenes() ([]SceneSummary, error) {
	s, err := c.store()
	if err != nil {
		return nil, err
	}
	current := s.Current().ID
	list := s.List()
	out := make([]SceneSummary, 0, len(list))
	for _, sc := range list {
		out = append(out, summary(sc, current))
	}
	return out, nil
}

// GetScene returns one scene with its HTML.
func (c *Control) GetScene(id string) (SceneView, error) {
	s, err := c.store()
	if err != nil {
		return SceneView{}, err
	}
	for _, sc := range s.List() {
		if sc.ID == id {
			return view(sc), nil
		}
	}
	return SceneView{}, scenes.ErrNotFound
}

// CurrentScene returns the scene the flyout should show.
func (c *Control) CurrentScene() (SceneView, error) {
	s, err := c.store()
	if err != nil {
		return SceneView{}, err
	}
	return view(s.Current()), nil
}

// SelectScene switches the flyout scene.
func (c *Control) SelectScene(id string) error {
	s, err := c.store()
	if err != nil {
		return err
	}
	if err := s.Select(id); err != nil {
		return err
	}
	c.changed()
	return nil
}

// SaveScene creates (id == "") or updates a scene.
func (c *Control) SaveScene(id, name, html string) (SceneView, error) {
	s, err := c.store()
	if err != nil {
		return SceneView{}, err
	}
	sc, err := s.Save(id, name, html)
	if err != nil {
		return SceneView{}, err
	}
	c.changed()
	return view(sc), nil
}

// DeleteScene removes a scene; home cannot be removed.
func (c *Control) DeleteScene(id string) error {
	s, err := c.store()
	if err != nil {
		return err
	}
	if err := s.Delete(id); err != nil {
		return err
	}
	c.changed()
	return nil
}

// RestoreHome writes the factory Home back and selects it.
func (c *Control) RestoreHome() (SceneView, error) {
	s, err := c.store()
	if err != nil {
		return SceneView{}, err
	}
	sc, err := s.RestoreHome()
	if err != nil {
		return SceneView{}, err
	}
	c.changed()
	return view(sc), nil
}

// InstallFromLink installs one catalog scene. link is nook://scene/<id>,
// optionally @<commit>, or the bare id. An installed scene with the same
// catalog id is refreshed instead of duplicated.
func (c *Control) InstallFromLink(link string) (SceneView, error) {
	id, commit, err := market.ParseLink(link)
	if err != nil {
		return SceneView{}, err
	}
	fetched, err := catalog.Fetch(context.Background(), id, commit)
	if err != nil {
		return SceneView{}, err
	}
	s, err := c.store()
	if err != nil {
		return SceneView{}, err
	}
	sc, err := s.Install(scenes.Market{
		ID:      fetched.Listing.ID,
		Version: fetched.Listing.Version,
		Commit:  fetched.Commit,
	}, fetched.Listing.Name, fetched.HTML)
	if err != nil {
		return SceneView{}, err
	}
	c.changed()
	return view(sc), nil
}

// ImportHTML opens a file dialog and returns the picked page. It does not
// save: the settings page shows it in the editor first.
func (c *Control) ImportHTML() (ImportResult, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:   "导入场景 HTML",
		Filters: []mygo.FileFilter{{Name: "HTML", Extensions: []string{"html", "htm"}}},
	})
	if err != nil {
		return ImportResult{}, err
	}
	if len(paths) == 0 {
		return ImportResult{Cancelled: true}, nil
	}
	info, err := os.Stat(paths[0])
	if err != nil {
		return ImportResult{}, err
	}
	if info.Size() > scenes.MaxHTMLSize {
		return ImportResult{}, scenes.ErrHTMLLarge
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		return ImportResult{}, err
	}
	name := strings.TrimSuffix(filepath.Base(paths[0]), filepath.Ext(paths[0]))
	return ImportResult{Name: name, HTML: string(raw)}, nil
}

// AttachFlyoutSize wires the callback that resizes the flyout window.
func AttachFlyoutSize(c *Control, fn func(w, h int)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onSize = fn
}

// SetFlyoutSize lets the mounted scene's measured size drive the flyout.
func (c *Control) SetFlyoutSize(w, h int) {
	if w < 1 || h < 1 {
		return
	}
	c.mu.Lock()
	fn := c.onSize
	c.mu.Unlock()
	if fn != nil {
		fn(w, h)
	}
}
