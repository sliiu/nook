package prefs

import (
	"path/filepath"
	"testing"
)

func TestMatch(t *testing.T) {
	cases := map[string]string{
		"(\n    \"zh-Hans-CN\",\n    \"en-US\"\n)": "zh",
		"(\"ja-JP\", \"en-US\")":                 "ja",
		"en_US.UTF-8":                           "en",
		"fr_FR":                                 "en",
		"":                                      "en",
	}
	for in, want := range cases {
		if got := Match(in); got != want {
			t.Errorf("Match(%q)=%s want %s", in, got, want)
		}
	}
}

func TestPersist(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	s := Open(p)
	if s.Setting() != Auto {
		t.Fatal("default should be auto")
	}
	if err := s.SetLang("ja"); err != nil {
		t.Fatal(err)
	}
	if Open(p).Resolved() != "ja" {
		t.Fatal("not persisted")
	}
	_ = s.SetLang("xx")
	if s.Setting() != Auto {
		t.Fatal("unknown should become auto")
	}
}
