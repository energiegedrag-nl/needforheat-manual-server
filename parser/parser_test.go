package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

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

// The manuals of a firmware repository get the main source's devices/template.html (the deployment's brand), not the
// repository's own template; without one in the main source, the repository's template is used as before.
func TestDeviceRepoManualsUseMainDeviceTemplate(t *testing.T) {
	repo := DeviceRepoSource{
		FS: fstest.MapFS{
			"docs/manuals/template.html":                   {Data: []byte("OLD {{.Title}}")},
			"docs/manuals/installation/languages/nl-NL.md": {Data: []byte("# Woonkamermodule\n\nTekst.")},
		},
		deviceName: "living-room",
	}
	page := "devices/living-room/installation/manufacturer/nl-NL/index.html"

	for _, tc := range []struct {
		name string
		main fstest.MapFS
		want string
	}{
		{"main template", fstest.MapFS{"devices/template.html": {Data: []byte("BRAND {{.Title}} {{.Language}}")}}, "BRAND Woonkamermodule nl-NL"},
		{"no main template", fstest.MapFS{"README.md": {Data: []byte("# readme")}}, "OLD Woonkamermodule"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := t.TempDir()
			p := New(dirfs.New(dest))
			if err := p.Parse(tc.main); err != nil {
				t.Fatalf("Parse main: %v", err)
			}
			if err := p.Parse(repo); err != nil {
				t.Fatalf("Parse repo: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(page)))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
