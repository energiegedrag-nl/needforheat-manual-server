package needforheatmanualserver

import (
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// SwapFS is a filesystem whose contents can be replaced at once, while requests keep being served.
// A request that already opened a file keeps reading it from the old filesystem.
type SwapFS struct {
	mu   sync.RWMutex
	fsys fs.FS
}

// NewSwapFS returns a SwapFS that serves fsys until the first Swap.
func NewSwapFS(fsys fs.FS) *SwapFS {
	return &SwapFS{fsys: fsys}
}

// Open opens the named file in the current filesystem.
func (s *SwapFS) Open(name string) (fs.File, error) {
	s.mu.RLock()
	fsys := s.fsys
	s.mu.RUnlock()
	return fsys.Open(name)
}

// Swap makes fsys the current filesystem.
func (s *SwapFS) Swap(fsys fs.FS) {
	s.mu.Lock()
	s.fsys = fsys
	s.mu.Unlock()
}

// ManualsInfo tells which version of the manuals is served.
type ManualsInfo struct {
	// Source is the git repository or directory the manuals come from.
	Source string `json:"source"`
	// Branch is the branch of a git source; empty for the default branch or a directory.
	Branch string `json:"branch,omitempty"`
	// Commit is the commit of a git source; empty for a directory.
	Commit string `json:"commit,omitempty"`
	// LoadedAt is when this version was fetched and parsed.
	LoadedAt time.Time `json:"loaded_at"`
}

// A LoadFunc fetches and parses the manuals. It returns the parsed filesystem, what was loaded and a function that
// removes the parsed files once they are no longer served.
type LoadFunc func() (fs.FS, ManualsInfo, func(), error)

// Reloader loads the manuals into a SwapFS and loads them again on request, so that a change in the manuals
// repository is served without restarting the server.
type Reloader struct {
	mu      sync.Mutex // one load at a time
	load    LoadFunc
	swap    *SwapFS
	info    ManualsInfo
	cleanup func()
	token   string

	infoMu sync.RWMutex
}

// NewReloader loads the manuals once and returns the Reloader and the filesystem to serve them from.
// token, when not empty, must be sent as "Authorization: Bearer <token>" to the admin routes.
func NewReloader(load LoadFunc, token string) (*Reloader, *SwapFS, error) {
	fsys, info, cleanup, err := load()
	if err != nil {
		return nil, nil, err
	}
	swap := NewSwapFS(fsys)
	return &Reloader{load: load, swap: swap, info: info, cleanup: cleanup, token: token}, swap, nil
}

// Info returns what is served now.
func (r *Reloader) Info() ManualsInfo {
	r.infoMu.RLock()
	defer r.infoMu.RUnlock()
	return r.info
}

// Reload fetches and parses the manuals again and serves them. On an error the old manuals stay.
func (r *Reloader) Reload() (ManualsInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fsys, info, cleanup, err := r.load()
	if err != nil {
		return r.Info(), err
	}
	r.swap.Swap(fsys)

	r.infoMu.Lock()
	old := r.cleanup
	r.info, r.cleanup = info, cleanup
	r.infoMu.Unlock()

	if old != nil {
		// Give requests that are still reading old files a moment before removing them.
		time.AfterFunc(time.Minute, old)
	}
	log.Println("reloaded manuals", info.Commit)
	return info, nil
}

// Routes returns the admin routes: GET /status and POST /reload. Serve them on a port that is only reachable from
// inside the server (NFH_ADMIN_ADDR), not on the public one.
func (r *Reloader) Routes() http.Handler {
	router := chi.NewRouter()
	router.Use(r.authorize)
	router.Get("/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, r.Info())
	})
	router.Post("/reload", func(w http.ResponseWriter, _ *http.Request) {
		info, err := r.Reload()
		if err != nil {
			log.Println("reloading manuals failed:", err)
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "served": info})
			return
		}
		writeJSON(w, http.StatusOK, info)
	})
	return router
}

func (r *Reloader) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.token != "" {
			sent, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(sent), []byte(r.token)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
