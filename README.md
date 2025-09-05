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


## Dev Make Targets

```bash
make build   # bin/mini-rproxy
make docker  # build container
make fmt     # go fmt
```
