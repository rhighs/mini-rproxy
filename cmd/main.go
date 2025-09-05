package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
	"github.com/tgym-digital/mini-rproxy/core/pluginmgr"
	"github.com/tgym-digital/mini-rproxy/core/rproxy"
)

type Config struct {
	ListenAddr string         `yaml:"listen_addr"`
	Routes     []rproxy.Route `yaml:"routes"`
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
		if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
			(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
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
	} else {
		e.LogMode = "text"
	}
	return e
}

func loadPlugins(dir string) []pluginapi.Plugin {
	if dir == "" {
		return nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("resolve plugin dir failed: %v\n", err)
		return nil
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := &pluginmgr.Loader{Logger: logger}
	plugins, err := loader.LoadAll(abs)
	if err != nil {
		log.Printf("plugin walk error: %v\n", err)
	}
	log.Printf("plugin scan dir=%s loaded=%d\n", abs, len(plugins))
	return plugins
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "config file")
	verbose := flag.Bool("verbose", false, "verbose logging")
	pluginDir := flag.String("plugindir", "", "directory containing .so plugins")
	flag.Parse()

	c := loadConfig(*cfgPath)
	e := loadEnv(".env")

	plugins := loadPlugins(*pluginDir)

	proxy := rproxy.NewRProxy(c.Routes, *verbose, e.LogMode, plugins)
	server := proxy.Start(c.ListenAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
