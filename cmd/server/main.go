package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	needforheatmanualserver "github.com/energiegedrag-nl/needforheat-manual-server"
	custommiddleware "github.com/energiegedrag-nl/needforheat-manual-server/middleware"
	"github.com/energiegedrag-nl/needforheat-manual-server/parser"
	"github.com/energiegedrag-nl/needforheat-manual-server/wfs/dirfs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Every load (start-up and each reload) parses the manuals into a new folder ./parsed/<n>; the server switches to
	// it when it is complete, so a failed reload leaves the manuals that were served.
	reloader, servedFS, err := needforheatmanualserver.NewReloader(loader(conf), conf.AdminToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("generated folder structure to be served")

	server := needforheatmanualserver.NewServer(servedFS, needforheatmanualserver.ServerOptions{
		FallbackLanguage: conf.FallbackLanguage,
	})

	r := chi.NewRouter()

	//CleanPathRedirect is the router, it will make sure everything redirects to the right page
	r.Use(middleware.Timeout(time.Second * 30))
	r.Use(middleware.Heartbeat("/healthcheck"))
	r.Use(custommiddleware.CleanPathRedirect)
	r.Use(middleware.Logger)

	r.Mount("/", server)

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		BaseContext:       returnContextFn(ctx),
		ReadHeaderTimeout: 10 * time.Second, // Protects against slowloris (gosec G112).
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	servers := []*http.Server{httpServer}
	if conf.AdminAddr != "" {
		admin := chi.NewRouter()
		admin.Use(middleware.Timeout(time.Minute * 5)) // cloning the repositories again takes a while
		admin.Use(middleware.Logger)
		admin.Mount("/admin", reloader.Routes())
		servers = append(servers, &http.Server{
			Addr:              conf.AdminAddr,
			Handler:           admin,
			BaseContext:       returnContextFn(ctx),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      6 * time.Minute,
			IdleTimeout:       120 * time.Second,
		})
	}

	err = listenAndServe(ctx, servers...)
	if err != nil {
		log.Println(err)
	}
}

// loader returns the function that fetches and parses the manuals, each time into a new folder under ./parsed.
func loader(conf *Config) needforheatmanualserver.LoadFunc {
	generation := 0
	return func() (fs.FS, needforheatmanualserver.ManualsInfo, func(), error) {
		info := needforheatmanualserver.ManualsInfo{Source: conf.SourceName, Branch: conf.SourceBranch}

		source, err := conf.OpenSource()
		if err != nil {
			return nil, info, nil, err
		}
		defer parser.CloseSource(source) //nolint:errcheck // a leftover temporary directory is harmless
		info.Commit = parser.SourceCommit(source)

		generation++
		dir := path.Join("parsed", strconv.Itoa(generation))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, info, nil, err
		}
		cleanup := func() { _ = os.RemoveAll(dir) }

		parsedFS := dirfs.New(dir)
		if err := parser.New(parsedFS).Parse(source); err != nil {
			cleanup()
			return nil, info, nil, err
		}
		info.LoadedAt = time.Now()
		return parsedFS, info, cleanup, nil
	}
}

// Return a function for BaseContext that always returns context ctx.
func returnContextFn(ctx context.Context) func(net.Listener) context.Context {
	return func(_ net.Listener) context.Context {
		return ctx
	}
}

// Start the HTTP servers and gracefully shut them down if the context is cancelled.
func listenAndServe(ctx context.Context, servers ...*http.Server) error {
	g, gCtx := errgroup.WithContext(ctx)

	for i, server := range servers {
		g.Go(func() error {
			return server.ListenAndServe()
		})

		if i == 0 {
			log.Println("serving manuals on", server.Addr)
		} else {
			log.Println("serving admin routes on", server.Addr)
		}

		g.Go(func() error {
			<-gCtx.Done()
			return server.Shutdown(context.Background())
		})
	}

	return g.Wait()
}
