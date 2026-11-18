package needforheatmanualserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golang.org/x/text/language"
)

func testServer() *Server {
	page := &fstest.MapFile{Data: []byte("<p>x</p>")}
	fsys := fstest.MapFS{
		"campaigns/generic/privacy/nl-NL/index.html":      page,
		"campaigns/generic/privacy/en-US/index.html":      page,
		"campaigns/generic/privacy-full/nl-NL/index.html": page,
		"campaigns/generic/privacy-full/en-US/index.html": page,
		"devices/lrm/display_names.json":                  {Data: []byte(`{}`)},
	}
	return NewServer(fsys, ServerOptions{FallbackLanguage: language.MustParse("nl-NL")})
}

func get(t *testing.T, s *Server, target, acceptLang, referer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil) // Host: example.com
	if acceptLang != "" {
		req.Header.Set("Accept-Language", acceptLang)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

// A relative link (../../privacy-full/) from a Dutch page stays Dutch, even if the browser prefers English.
func TestLanguageFromReferer(t *testing.T) {
	s := testServer()
	cases := []struct{ name, referer, want string }{
		{"same host keeps page language", "http://example.com/campaigns/generic/privacy/nl-NL/", "/campaigns/generic/privacy-full/nl-NL/"},
		{"no referer uses Accept-Language", "", "/campaigns/generic/privacy-full/en-US/"},
		{"other host is ignored", "http://elsewhere.org/campaigns/generic/privacy/nl-NL/", "/campaigns/generic/privacy-full/en-US/"},
		{"unavailable language is ignored", "http://example.com/campaigns/generic/privacy/de-DE/", "/campaigns/generic/privacy-full/en-US/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := get(t, s, "/campaigns/generic/privacy-full/", "en-US", c.referer)
			if rec.Code != http.StatusFound || rec.Header().Get("Location") != c.want {
				t.Fatalf("got %d %q, want 302 %q", rec.Code, rec.Header().Get("Location"), c.want)
			}
		})
	}
}

// Without a generic and without a manufacturer manual: one redirect to manufacturer, then 404 (no redirect loop).
func TestNoManufacturerRedirectLoop(t *testing.T) {
	s := testServer()
	rec := get(t, s, "/devices/lrm/installation/generic/", "", "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/devices/lrm/installation/manufacturer/" {
		t.Fatalf("generic: got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = get(t, s, "/devices/lrm/installation/manufacturer/", "", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("manufacturer: got %d %q, want 404", rec.Code, rec.Header().Get("Location"))
	}
}
