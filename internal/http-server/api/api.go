package api

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-chi/render"

	"humi/internal/config"
	"humi/internal/http-server/handlers/nodes"
	"humi/internal/http-server/handlers/readings"
	"humi/internal/http-server/handlers/service"
	"humi/internal/http-server/handlers/stats"
	"humi/internal/http-server/middleware/nodeauth"
	"humi/internal/http-server/middleware/timeout"
	"humi/internal/lib/sl"
)

// Handler is everything the API needs from the business layer.
type Handler interface {
	nodeauth.Authenticator
	readings.Handler
	nodes.Handler
	stats.Handler
	service.Handler
}

// New builds the router and blocks serving it.
func New(conf *config.Config, log *slog.Logger, handler Handler) error {
	srvLog := log.With(sl.Module("api.server"))

	router := chi.NewRouter()
	router.Use(timeout.Timeout(10))
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			next.ServeHTTP(w, r)
		})
	})

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", service.Health(log, handler))

		// Ingest is the only endpoint exposed to the internet: token + rate limit.
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(conf.Ingest.RatePerMin, time.Minute))
			r.Use(nodeauth.New(log, handler))
			r.Post("/readings", readings.Ingest(log, handler))
		})

		// TODO(phase5): put the read API behind the user JWT before exposing it.
		r.Get("/nodes", nodes.List(log, handler))
		r.Get("/nodes/{slug}/series", stats.Series(log, handler))
	})

	if conf.Web.Dir != "" {
		mountStatic(router, conf.Web.Dir)
		srvLog.Info("serving web ui", slog.String("dir", conf.Web.Dir))
	}

	httpLog := slog.NewLogLogger(log.Handler(), slog.LevelError)
	httpServer := &http.Server{
		Handler:           router,
		ErrorLog:          httpLog,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	address := fmt.Sprintf("%s:%s", conf.Listen.BindIP, conf.Listen.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	srvLog.Info("starting api server", slog.String("address", address))

	return httpServer.Serve(listener)
}

// mountStatic serves the Angular build, falling back to index.html so client
// side routes survive a reload.
func mountStatic(router chi.Router, dir string) {
	files := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")

	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
