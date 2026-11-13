package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pixel-control-center/internal/audio"
	"pixel-control-center/internal/market"
	"pixel-control-center/internal/scenes"
)

func TestInstallFromLinkRefreshesTheSameScene(t *testing.T) {
	page := "<p>market</p>"
	commit := strings.Repeat("ab", 20)
	meta := `{"id":"night","name":"Night","version":"1.0.0","description":"","htmlSha256":"` +
		scenes.HTMLHash(page) + `"}`
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs"):
			line := commit + " refs/heads/main\n"
			fmt.Fprintf(w, "%04x%s", len(line)+4, line)
		case strings.HasSuffix(r.URL.Path, "/scene.json"):
			w.Write([]byte(meta))
		case strings.HasSuffix(r.URL.Path, "/scene.html"):
			w.Write([]byte(page))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)

	prev := catalog
	catalog = &market.Client{Raw: api.URL, Refs: api.URL}
	t.Cleanup(func() { catalog = prev })

	store, err := scenes.Open(t.TempDir()+"/scenes.json", "<p>factory</p>")
	if err != nil {
		t.Fatal(err)
	}
	c := NewControl(&fakeAudio{level: audio.Level{Adjustable: true}}, nil, nil)
	AttachScenes(c, store, nil)

	first, err := c.InstallFromLink("nook://scene/night")
	if err != nil || first.MarketID != "night" || first.Commit != commit {
		t.Fatalf("install: %v %+v", err, first)
	}
	second, err := c.InstallFromLink("night")
	if err != nil || second.ID != first.ID {
		t.Fatalf("refresh should keep the scene, got %v %+v", err, second)
	}
	if _, err := c.SaveScene(first.ID, "Night", "<p>nope</p>"); err == nil {
		t.Fatal("a catalog scene must not be editable")
	}
	if _, err := c.InstallFromLink("https://evil.example/night"); err == nil {
		t.Fatal("a link must not be able to name a host")
	}
}
