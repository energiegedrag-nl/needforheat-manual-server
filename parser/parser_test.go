package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/energiegedrag-nl/needforheat-manual-server/wfs/dirfs"
)

// Markdown outside a languages folder (TODO.md, CHANGELOG.md, docs/...) is not a manual and must not stop the parser.
func TestParseSkipsMarkdownOutsideLanguages(t *testing.T) {
	src := t.TempDir()
	files := map[string]string{
		"README.md":           "# readme",
		"TODO.md":             "# todo",
		"docs/notes/index.md": "# notes",
		"campaigns/generic/faq/languages/nl-NL.md": "# Veelgestelde vragen\n\nTekst.",
		"campaigns/generic/faq/assets/.gitkeep":    "",
		"campaigns/generic/faq/display_names.json": `{"nl-NL": "FAQ"}`,
	}
	for name, content := range files {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dest := t.TempDir()
	sourceFS, err := NewLabDirSource(src)
	if err != nil {
		t.Fatal(err)
	}

	if err := New(dirfs.New(dest)).Parse(sourceFS); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var pages []string
	err = filepath.WalkDir(dest, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "index.html" {
			rel, _ := filepath.Rel(dest, p)
			pages = append(pages, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("want exactly one manual page (the nl-NL faq), got %v", pages)
	}
}

func TestIsManualFile(t *testing.T) {
	cases := map[string]bool{
		".":                                   false,
		"docs/manuals/installation":           false,
		"docs/manuals/installation/languages": true,
		"campaigns/generic/faq/languages":     true,
	}
	for dir, want := range cases {
		if got := isManualFile(dir); got != want {
			t.Errorf("isManualFile(%q) = %v, want %v", dir, got, want)
		}
	}
}
