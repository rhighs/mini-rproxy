package rproxy

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
)

type Route struct {
	Prefix   string
	Upstream string
}

func findRoute(routes []Route, p string) (Route, bool) {
	var best Route
	hit := false
	bestLen := -1
	for _, r := range routes {
		if strings.HasPrefix(p, r.Prefix) && len(r.Prefix) > bestLen {
			best = r
			hit = true
			bestLen = len(r.Prefix)
		}
	}
	return best, hit
}

type RProxy struct {
	verbose bool
	logger  *slog.Logger
	routes  []Route
	plugins []pluginapi.Plugin
}

func NewRProxy(routes []Route, verbose bool, logMode string, plugins []pluginapi.Plugin) *RProxy {
	var logger *slog.Logger
	if logMode == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return &RProxy{
		verbose: verbose,
		logger:  logger,
		routes:  routes,
		plugins: plugins,
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

const abortHeader = "X-MiniRProxy-Plugin-Abort"

type pluginAbortTransport struct {
	base http.RoundTripper
}

func (t *pluginAbortTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if msg := req.Header.Get(abortHeader); msg != "" {
		return nil, errors.New(msg)
	}
	return t.base.RoundTrip(req)
}

func (R *RProxy) Start(addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		w = rec
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"OK"}`))
		R.logger.Info("request",
			"path", r.URL.Path,
			"method", r.Method,
			"status", rec.status,
		)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		w = rec

		route, hit := findRoute(R.routes, r.URL.Path)
		if !hit {
			w.WriteHeader(http.StatusNotFound)
			R.logger.Info("request",
				"path", r.URL.Path,
				"method", r.Method,
				"status", rec.status,
			)
			return
		}

		up, err := url.Parse(route.Upstream)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			R.logger.Error("bad upstream", "upstream", route.Upstream, "error", err)
			R.logger.Info("request",
				"path", r.URL.Path,
				"method", r.Method,
				"status", rec.status,
			)
			return
		}

		if R.verbose {
			R.logger.Info("proxying",
				"method", r.Method,
				"path", r.URL.Path,
				"matched_prefix", route.Prefix,
				"upstream", up.String(),
			)
		}

		values := make(map[string]any)

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = up.Scheme
				req.URL.Host = up.Host
				req.Host = up.Host
				req.Header.Set("X-Forwarded-Host", r.Host)

				if strings.HasPrefix(req.URL.Path, route.Prefix) {
					req.URL.Path = strings.TrimPrefix(req.URL.Path, route.Prefix)
					if req.URL.Path == "" {
						req.URL.Path = "/"
					}
				}

				for _, p := range R.plugins {
					ctx := &pluginapi.Context{
						Phase:   pluginapi.PhaseRequest,
						Request: req,
						Values:  values,
					}
					if err := p.Handle(ctx); err != nil {
						req.Header.Set(abortHeader, p.Name()+": "+err.Error())
						break
					}
				}

				if R.verbose {
					R.logger.Info("upstream_request_ready",
						"method", req.Method,
						"url", req.URL.String(),
						"host", req.Host,
						"plugins", len(R.plugins),
					)
				}
			},
			Transport: &pluginAbortTransport{base: http.DefaultTransport},
			ModifyResponse: func(resp *http.Response) error {
				for _, p := range R.plugins {
					ctx := &pluginapi.Context{
						Phase:    pluginapi.PhaseResponse,
						Request:  resp.Request,
						Response: resp,
						Values:   values,
					}
					if err := p.Handle(ctx); err != nil {
						return err
					}
				}
				if R.verbose {
					R.logger.Info("response_from_upstream",
						"status", resp.StatusCode,
						"upstream", route.Upstream,
					)
				}
				return nil
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				status := http.StatusBadGateway
				if strings.Contains(err.Error(), "plugin") {
					status = http.StatusBadGateway
				}
				R.logger.Error("proxy_error", "error", err)
				http.Error(rw, http.StatusText(status), status)
			},
		}

		proxy.ServeHTTP(w, r)

		R.logger.Info("request",
			"path", r.URL.Path,
			"method", r.Method,
			"status", rec.status,
		)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		R.logger.Info("listening",
			"addr", addr,
			"plugins_loaded", len(R.plugins),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			R.logger.Error("server error", "error", err)
		}
	}()

	return srv
}
