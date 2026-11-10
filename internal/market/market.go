// Package market fetches one scene from the public catalog and checks it
// against the commit the catalog names. A scene that fails any check is not
// installed.
package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"pixel-control-center/internal/scenes"
)

const (
	// DefaultRepo is the only catalog the app installs from.
	DefaultRepo = "sliiu/nook-scenes"
	// DefaultRef is the branch whose tip a link without a commit resolves to.
	DefaultRef = "main"

	maxMeta = 16 << 10

	fetchTimeout = 20 * time.Second
)

var (
	ErrSceneID  = errors.New("scene id is not valid")
	ErrCommit   = errors.New("catalog commit is not valid")
	ErrMeta     = errors.New("catalog metadata is not valid")
	ErrMismatch = errors.New("scene does not match the catalog")
	ErrNetwork  = errors.New("catalog could not be reached")
)

var (
	repoRe   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	idRe     = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)
	commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

	// Markup that reaches out of the file. A data: image is fine; a remote one
	// is not, because the review never saw those bytes.
	scriptSrc = regexp.MustCompile(`(?i)<script\b[^>]*\bsrc\s*=`)
	linkHref  = regexp.MustCompile(`(?i)<link\b[^>]*\bhref\s*=`)
	remoteImg = regexp.MustCompile(`(?i)<img\b[^>]*\bsrc\s*=\s*["']?https?:`)
)

// callNames are script calls a reviewed page has no reason to make. Matching
// is a reason to refuse, not a proof the page is safe: the sandbox is the
// boundary that holds when the text scan misses something.
var callNames = []string{"eval", "fetch", "import", "WebSocket", "XMLHttpRequest"}

// Listing is one scene's scene.json.
type Listing struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	HTMLSHA256  string `json:"htmlSha256"`
}

// Scene is a catalog page that passed every check.
type Scene struct {
	Listing Listing
	HTML    string
	Commit  string
}

// Client talks to one GitHub repository. Tests point Raw and Commits at a
// local server; both stay empty in the app.
type Client struct {
	Repo    string
	HTTP    *http.Client
	Raw     string // host serving /<repo>/<commit>/<id>/{scene.json,scene.html}
	Commits string // host serving /repos/<repo>/commits/<ref>
}

// ParseLink reads nook://scene/<id>[@<commit>] and the same id written bare.
// Anything else is refused, so a link cannot name a host or a path.
func ParseLink(link string) (id, commit string, err error) {
	s := strings.TrimSpace(link)
	s = strings.TrimPrefix(s, "nook://scene/")
	s = strings.TrimPrefix(s, "nook://")
	id, commit, _ = strings.Cut(s, "@")
	id = strings.Trim(id, "/")
	commit = strings.ToLower(commit)
	if !idRe.MatchString(id) {
		return "", "", ErrSceneID
	}
	if commit != "" && !commitRe.MatchString(commit) {
		return "", "", ErrCommit
	}
	return id, commit, nil
}

// Fetch loads one scene and checks the commit, the hash and the page. commit
// may be empty, which resolves to DefaultRef.
func (c *Client) Fetch(ctx context.Context, id, commit string) (Scene, error) {
	if !idRe.MatchString(id) {
		return Scene{}, ErrSceneID
	}
	if commit != "" && !commitRe.MatchString(commit) {
		return Scene{}, ErrCommit
	}
	repo := c.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	if !repoRe.MatchString(repo) {
		return Scene{}, ErrMeta
	}
	sha, err := c.resolve(ctx, repo, commit)
	if err != nil {
		return Scene{}, err
	}
	metaRaw, err := c.get(ctx, c.rawURL(repo, sha, id, "scene.json"), maxMeta)
	if err != nil {
		return Scene{}, err
	}
	htmlRaw, err := c.get(ctx, c.rawURL(repo, sha, id, "scene.html"), scenes.MaxHTMLSize)
	if err != nil {
		return Scene{}, err
	}
	scene, err := Check(id, sha, metaRaw, string(htmlRaw))
	if err != nil {
		return Scene{}, err
	}
	return scene, nil
}

// Check reports whether meta and html are the scene id at commit. It does not
// touch the network, so the review action and the app share one verdict.
func Check(id, commit string, metaRaw []byte, html string) (Scene, error) {
	if !idRe.MatchString(id) || !commitRe.MatchString(commit) {
		return Scene{}, ErrMeta
	}
	var meta Listing
	dec := json.NewDecoder(strings.NewReader(string(metaRaw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&meta); err != nil {
		return Scene{}, ErrMeta
	}
	if meta.ID != id || meta.HTMLSHA256 != scenes.HTMLHash(html) {
		return Scene{}, ErrMismatch
	}
	if strings.TrimSpace(meta.Name) == "" || utf8Len(meta.Name) > scenes.MaxNameLen {
		return Scene{}, ErrMeta
	}
	if meta.Version == "" || len(meta.Version) > 32 || strings.TrimSpace(html) == "" {
		return Scene{}, ErrMeta
	}
	if scriptSrc.MatchString(html) || linkHref.MatchString(html) || remoteImg.MatchString(html) || unsafeScript(html) {
		return Scene{}, ErrMismatch
	}
	return Scene{Listing: meta, HTML: html, Commit: commit}, nil
}

// unsafeScript reports calls and frame walks the catalog refuses. Names are
// matched as whole identifiers, so "top" inside "desktop" is not a hit.
func unsafeScript(html string) bool {
	low := strings.ToLower(html)
	if strings.Contains(low, "new function") && hasCall(html, "Function") {
		return true
	}
	if strings.Contains(low, "document.write") {
		return true
	}
	for _, name := range callNames {
		if strings.Contains(html, name) && hasCall(html, name) {
			return true
		}
	}
	return identUsed(html, "parent") || identUsed(html, "top")
}

// hasCall reports name followed by an opening paren, ignoring whitespace.
func hasCall(s, name string) bool {
	for from := 0; ; {
		at := strings.Index(s[from:], name)
		if at < 0 {
			return false
		}
		at += from
		if bounded(s, at, at+len(name)) {
			rest := strings.TrimLeftFunc(s[at+len(name):], unicode.IsSpace)
			if strings.HasPrefix(rest, "(") {
				return true
			}
		}
		from = at + len(name)
	}
}

func identUsed(s, name string) bool {
	for from := 0; ; {
		at := strings.Index(s[from:], name)
		if at < 0 {
			return false
		}
		at += from
		if bounded(s, at, at+len(name)) {
			return true
		}
		from = at + len(name)
	}
}

func bounded(s string, start, end int) bool {
	return (start == 0 || !isIdent(s[start-1])) && (end == len(s) || !isIdent(s[end]))
}

func isIdent(b byte) bool {
	return b == '_' || b == '$' || (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func (c *Client) resolve(ctx context.Context, repo, commit string) (string, error) {
	if commit != "" {
		return commit, nil
	}
	host := c.Commits
	if host == "" {
		host = "https://api.github.com"
	}
	url := strings.TrimRight(host, "/") + "/repos/" + repo + "/commits/" + DefaultRef
	raw, err := c.get(ctx, url, maxMeta)
	if err != nil {
		return "", err
	}
	var body struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || !commitRe.MatchString(body.SHA) {
		return "", ErrCommit
	}
	return body.SHA, nil
}

func (c *Client) rawURL(repo, commit, id, file string) string {
	host := c.Raw
	if host == "" {
		host = "https://raw.githubusercontent.com"
	}
	return strings.TrimRight(host, "/") + "/" + repo + "/" + commit + "/" + id + "/" + file
}

func (c *Client) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrNetwork
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "nook")
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNetwork, err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrNetwork, res.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNetwork, err.Error())
	}
	if int64(len(raw)) > limit {
		return nil, scenes.ErrHTMLLarge
	}
	return raw, nil
}

func utf8Len(s string) int { return utf8.RuneCountInString(s) }
