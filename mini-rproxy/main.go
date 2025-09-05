package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Prefix   string `yaml:"prefix"`
	Upstream string `yaml:"upstream"`
}

type Config struct {
	ListenAddr string  `yaml:"listen_addr"`
	Routes     []Route `yaml:"routes"`
}

type EnvConfig struct {
	LogMode string
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) ||
			strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`) {
			val = val[1 : len(val)-1]
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}

func loadConfig(path string) Config {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		log.Fatalf("parse yaml: %v", err)
	}
	if c.ListenAddr == "" {
		c.ListenAddr = ":8080"
	}
	return c
}

func loadEnv(path string) EnvConfig {
	if err := loadDotEnv(path); err != nil {
		log.Printf("failed loading %s env file, skipping...\n", path)
	}
	var e EnvConfig
	if v := os.Getenv("LOG_MODE"); v != "" {
		if v == "text" || v == "json" {
			e.LogMode = v
		} else {
			e.LogMode = "text"
		}
	}
	return e
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

func main() {
	cfgPath := flag.String("config", "config.yaml", "config file")
	verbose := flag.Bool("verbose", false, "verbose logging")
	flag.Parse()

	c := loadConfig(*cfgPath)
	e := loadEnv(".env")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if e.LogMode == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"message\":\"OK\"}"))
	})

	mux.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			route, hit := findRoute(c.Routes, r.URL.Path)
			if !hit {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			url, err := url.Parse(route.Upstream)
			if err != nil {
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(url)
			if *verbose {
				logger.Info("proxying incoming request",
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
					"matched_prefix", route.Prefix,
					"upstream_url", url.String(),
				)
				logger.Info("proxying headers",
					"headers", r.Header,
				)
			}
			{
				r.URL.Host = url.Host
				r.URL.Scheme = url.Scheme
				r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
				r.Host = url.Host
				path := r.URL.Path
				r.URL.Path = strings.TrimLeft(path, route.Prefix)
			}
			if *verbose {
				logger.Info("[req->upstream]",
					"method", r.Method,
					"url", r.URL.String(),
					"host", r.Host,
					"headers", r.Header,
				)
			}
			proxy.ServeHTTP(w, r)
		},
	)

	srv := &http.Server{
		Addr:              c.ListenAddr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", c.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
