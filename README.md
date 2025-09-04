# Relay Proxy (JWT → Legacy Token Bridge)

A tiny, purpose-built reverse proxy that sits in front of MyWellness legacy services.  
It accepts modern JWT-based Authorization headers, peeks inside and inspects claims so that we can transparently evolve toward **issuing or injecting the legacy token fields** required by older upstreams—without forcing clients to know or even care!

> Chill summary: point your SDK / client at this instead of the zoo of `*.example.com` hosts, ship a Bearer JWT, and let the proxy handle routing + (soon) legacy auth shape.

---

## What it does (today)

- Reverse proxies requests based on path prefix → configured upstream base URL.
- Health endpoint at `/health`.
- Loads routing from a simple `config.yaml` (or a custom path via `-config`).
- Reads runtime secrets / keys from environment (or `.env`) for future token translation.
- Decodes HS256 JWTs (if `Authorization: Bearer <token>`) and logs the claims (debug/trace step before full translation logic).
- Normalizes `X-Forwarded-Host`, upstream host, and trims configured prefix.
- Inject legacy token headers derived from JWT claims + configured secrets.
- Optionally cache / memoize legacy token derivations per JWT.

## Quick Start (Local)

```bash
# 1. Copy examples
cp config.example.yml config.yaml
cp .env.example .env   # edit values

# 2. Run (Go)
go run ./relay-proxy

# or build & run
go build -o relay ./relay-proxy && JWT_SECRET=dev ./relay -config config.yaml

# 3. Hit it
curl -i http://localhost:8080/health
```

Proxying a request (example path mapped to `api-beta.example.net`):
```bash
curl -H "Authorization: Bearer $JWT" http://localhost:8080/workout/say33
```
## Docker

```bash
docker build -t relay-proxy:latest .
docker run --rm \
  -p 8080:8080 \
  -e JWT_SECRET=devsecret \
  -v "$(pwd)/config.yaml:/app/config.yml:ro" \
  relay-proxy:latest
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

## Environment / Secrets

| Variable          | Purpose                                      | Required |
|-------------------|----------------------------------------------|----------|
| JWT_SECRET        | HS256 secret for validating client JWTs      | Yes      |
| TG_HASH_KEY       | Legacy auth hash key (future translation)    | No       |
| TG_SIGN_SALT      | Legacy signing salt (future translation)     | No       |
| TG_LEGACY_PKEY    | Legacy private key / compat material         | No       |
| (future) *_ARN    | Fetch from AWS Secrets Manager               | Planned  |

`.env` file is auto-read if present (simple KEY=VALUE lines).

## CLI Flags

| Flag      | Default     | Description                  |
|-----------|-------------|------------------------------|
| `-config` | `config.yaml` | Path to YAML config file     |
| `-verbose` | `false`    | Extra request + claim logging |


## Dev Make Targets

```bash
make build   # bin/relay-proxy
make docker  # build container
make fmt     # go fmt
```
