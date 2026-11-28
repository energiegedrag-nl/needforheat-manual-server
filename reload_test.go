package needforheatmanualserver

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/text/language"
)

// A loader that serves "v1", "v2", ... as the display name of one device, and fails when fail is set.
type fakeLoader struct {
	n    int
	fail bool
}

func (l *fakeLoader) load() (fs.FS, ManualsInfo, func(), error) {
	if l.fail {
		return nil, ManualsInfo{}, nil, errors.New("clone failed")
	}
	l.n++
	version := "v" + string(rune('0'+l.n))
	fsys := fstest.MapFS{"devices/lrm/display_names.json": {Data: []byte(version)}}
	return fsys, ManualsInfo{Source: "test", Commit: version, LoadedAt: time.Now()}, func() {}, nil
}

func do(t *testing.T, h http.Handler, method, target, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReloadServesNewManuals(t *testing.T) {
	loader := &fakeLoader{}
	reloader, swap, err := NewReloader(loader.load, "")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(swap, ServerOptions{FallbackLanguage: language.MustParse("nl-NL")})
	admin := reloader.Routes()

	if got := do(t, server, http.MethodGet, "/devices/lrm/", "").Body.String(); got != "v1" {
		t.Fatalf("before reload: got %q, want v1", got)
	}
	rec := do(t, admin, http.MethodPost, "/reload", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("reload: status %d", rec.Code)
	}
	if got := do(t, server, http.MethodGet, "/devices/lrm/", "").Body.String(); got != "v2" {
		t.Fatalf("after reload: got %q, want v2", got)
	}
	var info ManualsInfo
	if err := json.NewDecoder(do(t, admin, http.MethodGet, "/status", "").Body).Decode(&info); err != nil || info.Commit != "v2" {
		t.Fatalf("status: %+v, %v", info, err)
	}
}

func TestFailedReloadKeepsOldManuals(t *testing.T) {
	loader := &fakeLoader{}
	reloader, swap, err := NewReloader(loader.load, "")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(swap, ServerOptions{FallbackLanguage: language.MustParse("nl-NL")})

	loader.fail = true
	if rec := do(t, reloader.Routes(), http.MethodPost, "/reload", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("failed reload: status %d, want 502", rec.Code)
	}
	if got := do(t, server, http.MethodGet, "/devices/lrm/", "").Body.String(); got != "v1" {
		t.Fatalf("after failed reload: got %q, want v1", got)
	}
	if reloader.Info().Commit != "v1" {
		t.Fatalf("info after failed reload: %+v", reloader.Info())
	}
}

func TestAdminToken(t *testing.T) {
	loader := &fakeLoader{}
	reloader, _, err := NewReloader(loader.load, "secret")
	if err != nil {
		t.Fatal(err)
	}
	admin := reloader.Routes()
	for _, token := range []string{"", "wrong"} {
		if rec := do(t, admin, http.MethodPost, "/reload", token); rec.Code != http.StatusUnauthorized {
			t.Fatalf("token %q: status %d, want 401", token, rec.Code)
		}
	}
	if loader.n != 1 {
		t.Fatalf("refused requests loaded anyway: %d loads", loader.n)
	}
	if rec := do(t, admin, http.MethodPost, "/reload", "secret"); rec.Code != http.StatusOK {
		t.Fatalf("right token: status %d", rec.Code)
	}
	if rec := do(t, admin, http.MethodGet, "/reload", "secret"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /reload: status %d, want 405", rec.Code)
	}
}
