# mini-rproxy

A tiny reverse proxy with support for simple request processing plugins.

---

## What it does

- Reverse proxies requests based on path prefix → configured upstream base URL.
- Health endpoint at `/health`.
- Loads routing from a simple `config.yaml` (or a custom path via `-config`).
- Normalizes `X-Forwarded-Host`, upstream host, and trims configured prefix.

## Quick Start (Local)

### Copy examples
```bash
cp config.example.yml config.yaml
```

### Run it
```bash
make run
```

### Run a health check
```
curl -i http://localhost:8080/health
```

### Reach one of the example upstream

> req -> localhost:8080/workout/say33 --[proxy pass]--> https://api-beta.example.net/say33

```
curl -X GET http://localhost:8080/workout/say33 | jq
```

## Docker

```bash
docker build -t mini-rproxy:latest .
docker run --rm \
  -p 8080:8080 \
  -v "$(pwd)/config.yaml:/app/config.yml:ro" \
  mini-rproxy:latest
```

Or via compose:

```bash
docker compose up --build
```

## Configuration

`config.yaml` (example):

```yaml
listen_addr: ":8080"
routes:
  - prefix: /workout
    upstream: https://api-beta.example.net
  - prefix: /core
    upstream: https://api-gamma.example.net
  - prefix: /link
    upstream: https://api-zeta.example.net
```

Routing rule: longest matching `prefix` wins. That prefix is trimmed before forwarding.

## CLI Flags

| Flag      | Default        | Description                  |
|-----------|----------------|------------------------------|
| `-config` | `config.yaml`  | Path to YAML config file     |
| `-verbose`| `false`        | Extra request logging        |
| `-plugindir` | (empty)     | Directory with `.so` plugins |

## Dev Make Targets

```bash
make build   # bin/mini-rproxy
make docker  # build container
make fmt     # go fmt
```

---

## Plugin Support

mini-rproxy supports runtime plugins for request/response processing.  
Plugins are Go shared objects (`.so` files) loaded dynamically at startup.

### How plugin loading works

1. **Build your plugin** as a `.so` using Go's plugin system (see example below).
2. **Place the plugin file** in a directory (e.g. `bin/plugins/`).
3. **Start mini-rproxy** with the `-plugindir` flag pointing to your plugin directory:
   ```bash
   ./bin/mini-rproxy -config ./config.yaml -plugindir ./bin/plugins
   ```
4. mini-rproxy will scan the directory and load all `.so` plugins, logging the loaded plugin count.

### Example: Simple Plugin

A minimal plugin must implement the `pluginapi.Plugin` interface  
and export a symbol named `MiniRProxyPluginInstance`.  
Here is an example that adds a header to requests and responses:

```go
// plugins/headerdemo/headerdemo.go
package main

import (
    "github.com/tgym-digital/mini-rproxy/core/pluginapi"
)

type HeaderDemo struct{}

func (h *HeaderDemo) Name() string { return "header-demo" }

func (h *HeaderDemo) Handle(ctx *pluginapi.Context) error {
    switch ctx.Phase {
    case pluginapi.PhaseRequest:
        ctx.Request.Header.Set("X-Demo-Request", "hello")
    case pluginapi.PhaseResponse:
        if ctx.Response != nil {
            ctx.Response.Header.Set("X-Demo-Response", "hello")
        }
    }
    return nil
}

// This symbol must be exported for plugin loading:
var MiniRProxyPluginInstance pluginapi.Plugin = &HeaderDemo{}
```

### Build the plugin

```bash
go build -buildmode=plugin -o bin/plugins/headerdemo.so ./plugins/headerdemo
```

### Plugin loading notes

- The main binary and all plugins **must be built with the exact same Go version**.
- If you change the `pluginapi` interface, rebuild both main and plugins.
- On startup, mini-rproxy logs a count of loaded plugins.  
  If you see `plugins_loaded: 0`, check your build and plugin directory.

---

### Writing your own plugin

1. Implement the `pluginapi.Plugin` interface (see above).
2. Export the `MiniRProxyPluginInstance` symbol.
3. Build with `go build -buildmode=plugin ...`.
4. Place the `.so` in your plugin directory, and restart mini-rproxy.

---

Let us know if you want more plugin examples!