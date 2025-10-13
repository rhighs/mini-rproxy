package pluginmgr

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"plugin"

	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
)

type Loader struct {
	Logger *slog.Logger
}

func (l *Loader) log(level string, msg string, args ...any) {
	if l.Logger == nil {
		return
	}
	switch level {
	case "info":
		l.Logger.Info(msg, args...)
	case "error":
		l.Logger.Error(msg, args...)
	}
}

func (l *Loader) LoadAll(dir string) ([]pluginapi.Plugin, error) {
	var plugins []pluginapi.Plugin
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, wErr error) error {
		if wErr != nil {
			return wErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".so" {
			return nil
		}
		p, err := l.loadOne(path)
		if err != nil {
			l.log("error", "failed loading plugin", "path", path, "error", err)
			return nil
		}

		p.Init()
		l.log("info", "loaded plugin", "name", p.Name(), "path", path)
		plugins = append(plugins, p)
		return nil
	})
	return plugins, err
}

func (l *Loader) loadOne(path string) (pluginapi.Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	sym, err := p.Lookup("MiniRProxyPluginInstance")
	if err != nil {
		return nil, fmt.Errorf("lookup MiniRProxyPluginInstance: %w", err)
	}
	plug, ok := sym.(pluginapi.Plugin)
	if !ok {
		if ptr, ok2 := sym.(*pluginapi.Plugin); ok2 && ptr != nil {
			if *ptr == nil {
				return nil, errors.New("plugin symbol pointer is nil")
			}
			return *ptr, nil
		}
		return nil, fmt.Errorf("symbol does not implement pluginapi.Plugin (type=%T)", sym)
	}
	return plug, nil
}
