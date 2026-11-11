package scenes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func open(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenes.json")
	s, err := Open(path, "<p>factory</p>")
	if err != nil {
		t.Fatal(err)
	}
	return s, path
}

func TestSeedsHomeAndSelectsIt(t *testing.T) {
	s, _ := open(t)
	list := s.List()
	if len(list) != 1 || list[0].ID != HomeID || list[0].HTML != "<p>factory</p>" {
		t.Fatalf("unexpected seed: %+v", list)
	}
	if s.Current().ID != HomeID {
		t.Fatal("home should be current")
	}
}

func TestCorruptFileIsReplacedByFactory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenes.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path, "<p>factory</p>")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 || s.Current().ID != HomeID {
		t.Fatalf("corrupt file not repaired: %+v", s.List())
	}
}

func TestSaveCreateUpdatePersist(t *testing.T) {
	s, path := open(t)
	sc, err := s.Save("", "Night", "<b>x</b>")
	if err != nil || sc.ID == "" {
		t.Fatalf("create: %v %+v", err, sc)
	}
	if _, err := s.Save(sc.ID, "Night 2", "<b>y</b>"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, "<p>factory</p>")
	if err != nil {
		t.Fatal(err)
	}
	list := reopened.List()
	if len(list) != 2 || list[1].Name != "Night 2" || list[1].HTML != "<b>y</b>" {
		t.Fatalf("not persisted: %+v", list)
	}
}

func TestNameRules(t *testing.T) {
	s, _ := open(t)
	cases := []struct {
		name, html string
		want       error
	}{
		{"  ", "<p>", ErrNameEmpty},
		{strings.Repeat("长", MaxNameLen+1), "<p>", ErrNameLong},
		{"ok", "   ", ErrHTMLEmpty},
		{"ok", strings.Repeat("a", MaxHTMLSize+1), ErrHTMLLarge},
		{"home", "<p>", ErrNameTaken},
	}
	for _, c := range cases {
		if _, err := s.Save("", c.name, c.html); !errors.Is(err, c.want) {
			t.Errorf("Save(%q) = %v, want %v", c.name, err, c.want)
		}
	}
	if _, err := s.Save("", "A", "<p>"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save("", "a", "<p>"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("names must be case-insensitive unique, got %v", err)
	}
}

func TestHomeCannotBeDeletedOrRenamed(t *testing.T) {
	s, _ := open(t)
	if err := s.Delete(HomeID); !errors.Is(err, ErrHomeLocked) {
		t.Fatalf("delete home: %v", err)
	}
	sc, err := s.Save(HomeID, "Renamed", "<p>custom</p>")
	if err != nil || sc.Name != HomeName || sc.HTML != "<p>custom</p>" {
		t.Fatalf("home save: %v %+v", err, sc)
	}
}

func TestDeleteCurrentFallsBackToHome(t *testing.T) {
	s, _ := open(t)
	sc, _ := s.Save("", "Extra", "<p>")
	if err := s.Select(sc.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(sc.ID); err != nil {
		t.Fatal(err)
	}
	if s.Current().ID != HomeID {
		t.Fatal("should fall back to home")
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestInstallAndRefresh(t *testing.T) {
	s, path := open(t)
	m := Market{ID: "night", Version: "1.0.0", Commit: strings.Repeat("ab", 20), HTMLHash: strings.Repeat("cd", 32)}
	sc, err := s.Install(m, "Night", "<p>v1</p>")
	if err != nil || sc.Market == nil || sc.Market.HTMLHash != HTMLHash("<p>v1</p>") {
		t.Fatalf("install: %v %+v", err, sc)
	}
	if _, err := s.Save(sc.ID, "Night", "<p>edited</p>"); !errors.Is(err, ErrMarket) {
		t.Fatalf("catalog scene must not be editable, got %v", err)
	}
	m.Version = "1.1.0"
	updated, err := s.Install(m, "Night", "<p>v2</p>")
	if err != nil || updated.ID != sc.ID || updated.HTML != "<p>v2</p>" || updated.Market.Version != "1.1.0" {
		t.Fatalf("refresh: %v %+v", err, updated)
	}
	if len(s.List()) != 2 {
		t.Fatalf("refresh created a second scene: %+v", s.List())
	}
	reopened, err := Open(path, "<p>factory</p>")
	if err != nil || reopened.List()[1].Market.Commit != m.Commit {
		t.Fatalf("market record not persisted: %v", err)
	}
}

func TestInstallRejectsBadRecords(t *testing.T) {
	s, _ := open(t)
	ok := Market{ID: "night", Version: "1", Commit: strings.Repeat("ab", 20), HTMLHash: strings.Repeat("cd", 32)}
	bad := []Market{
		{ID: "Night", Version: "1", Commit: ok.Commit, HTMLHash: ok.HTMLHash},
		{ID: "night", Version: "", Commit: ok.Commit, HTMLHash: ok.HTMLHash},
		{ID: "night", Version: "1", Commit: "abc", HTMLHash: ok.HTMLHash},
	}
	for _, m := range bad {
		if _, err := s.Install(m, "Night", "<p>"); !errors.Is(err, ErrMarket) {
			t.Errorf("Install(%+v) = %v", m, err)
		}
	}
}

func TestRepairDropsBrokenMarketRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenes.json")
	raw := `{"current":"home","scenes":[{"id":"s_1","name":"Night","html":"<p>","market":{"id":"BAD","version":"1","commit":"abc","htmlHash":"abc"}}]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path, "<p>factory</p>")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 {
		t.Fatalf("broken market record kept: %+v", s.List())
	}
}

func TestRestoreHome(t *testing.T) {
	s, _ := open(t)
	_, _ = s.Save(HomeID, "x", "<p>custom</p>")
	extra, _ := s.Save("", "Extra", "<p>")
	_ = s.Select(extra.ID)
	home, err := s.RestoreHome()
	if err != nil || home.HTML != "<p>factory</p>" || s.Current().ID != HomeID {
		t.Fatalf("restore: %v %+v", err, home)
	}
}
