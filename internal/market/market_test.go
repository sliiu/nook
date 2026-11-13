package market

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pixel-control-center/internal/scenes"
)

const (
	commit = "0123456789abcdef0123456789abcdef01234567"
	page   = "<div style=\"width:360px;height:360px\"></div>\n<script>\nControl.getState().then((s) => { desktop.textContent = s.volume })\n</script>\n"
)

func meta() string {
	return `{"id":"night","name":"Night","version":"1.0.0","description":"dark","htmlSha256":"` +
		scenes.HTMLHash(page) + `"}`
}

func TestCheckAcceptsAPlainPage(t *testing.T) {
	scene, err := Check("night", commit, []byte(meta()), page)
	if err != nil {
		t.Fatal(err)
	}
	if scene.Listing.Name != "Night" || scene.Commit != commit {
		t.Fatalf("%+v", scene.Listing)
	}
}

func TestCheckRefuses(t *testing.T) {
	cases := map[string]string{
		"remote script":  "<script src=\"https://evil/x.js\"></script>",
		"stylesheet":     "<link rel=\"stylesheet\" href=\"x.css\">",
		"remote image":   "<img src=\"https://evil/a.png\">",
		"eval":           "<script>eval('1')</script>",
		"function ctor":  "<script>new Function('return 1')()</script>",
		"document write": "<script>document.write('x')</script>",
		"dynamic import": "<script>import('./x.js')</script>",
		"fetch":          "<script>fetch('/x')</script>",
		"websocket":      "<script>new WebSocket('wss://x')</script>",
		"xhr":            "<script>new XMLHttpRequest()</script>",
		"parent frame":   "<script>parent.Control</script>",
		"parent index":   "<script>parent['Control']</script>",
		"window top":     "<script>window.top.location</script>",
		"top frame":      "<script>top.location</script>",
	}
	for name, html := range cases {
		raw := strings.Replace(meta(), scenes.HTMLHash(page), scenes.HTMLHash(html), 1)
		if _, err := Check("night", commit, []byte(raw), html); !errors.Is(err, ErrMismatch) {
			t.Errorf("%s: got %v, want ErrMismatch", name, err)
		}
	}

	wrong := strings.Replace(meta(), scenes.HTMLHash(page), strings.Repeat("ab", 32), 1)
	if _, err := Check("night", commit, []byte(wrong), page); !errors.Is(err, ErrMismatch) {
		t.Errorf("hash mismatch: %v", err)
	}
	other := strings.Replace(meta(), `"id":"night"`, `"id":"other"`, 1)
	if _, err := Check("night", commit, []byte(other), page); !errors.Is(err, ErrMismatch) {
		t.Errorf("id mismatch: %v", err)
	}

	with := strings.TrimRight(meta(), "}") + `,"preview":"preview.png","author":"Ada"}`
	if _, err := Check("night", commit, []byte(with), page); err != nil {
		t.Errorf("preview and author: %v", err)
	}
	remote := strings.TrimRight(meta(), "}") + `,"preview":"https://evil/a.png"}`
	if _, err := Check("night", commit, []byte(remote), page); !errors.Is(err, ErrMeta) {
		t.Errorf("remote preview: %v", err)
	}
}

func TestCheckAllowsWordsInsideIdentifiers(t *testing.T) {
	html := page + "<style>body{top:0}</style><script>const top = rect.top; self.desktop = 1</script>" // contains "desktop" and "textContent"
	raw := strings.Replace(meta(), scenes.HTMLHash(page), scenes.HTMLHash(html), 1)
	if _, err := Check("night", commit, []byte(raw), html); err != nil {
		t.Fatal(err)
	}
}

func TestParseLink(t *testing.T) {
	id, got, err := ParseLink("nook://scene/night@" + commit)
	if err != nil || id != "night" || got != commit {
		t.Fatalf("%s %s %v", id, got, err)
	}
	if _, _, err := ParseLink("night"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"nook://scene/Night", "https://evil/night", "night@abc", "../night"} {
		if _, _, err := ParseLink(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestFetchResolvesTheBranchTip(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/sliiu/nook-scenes/commits/main":
			w.Write([]byte(`{"sha":"` + commit + `"}`))
		case "/sliiu/nook-scenes/" + commit + "/scenes/night/scene.json":
			w.Write([]byte(meta()))
		case "/sliiu/nook-scenes/" + commit + "/scenes/night/scene.html":
			w.Write([]byte(page))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	scene, err := (&Client{Raw: api.URL, Commits: api.URL}).Fetch(context.Background(), "night", "")
	if err != nil {
		t.Fatal(err)
	}
	if scene.Commit != commit || scene.HTML != page {
		t.Fatalf("%+v", scene)
	}
}

func TestFetchRefusesATamperedPage(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "scene.html") {
			w.Write([]byte(page + "<script>fetch('https://evil')</script>"))
			return
		}
		w.Write([]byte(meta()))
	}))
	defer api.Close()

	_, err := (&Client{Raw: api.URL}).Fetch(context.Background(), "night", commit)
	if !errors.Is(err, ErrMismatch) {
		t.Fatalf("got %v", err)
	}
}
