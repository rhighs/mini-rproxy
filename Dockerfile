# syntax=docker/dockerfile:experimental
FROM --platform=$BUILDPLATFORM golang:1.24.4 AS build
ARG TARGET_ARCH
ARG BUILDPLATFORM

# cache go mod stuff
RUN --mount=type=cache,target=/go/pkg/mod \
    true

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN if [ -n "$TARGET_ARCH" ]; then \
      GOARCH="$TARGET_ARCH"; \
    else \
      case "$BUILDPLATFORM" in \
        *arm64*) GOARCH=arm64 ;; \
        *amd64*) GOARCH=amd64 ;; \
        *) GOARCH=amd64 ;; \
      esac ; \
    fi && \
    GOOS=linux GOARCH=$GOARCH CGO_ENABLED=1 make -C . && \
    GOOS=linux GOARCH=$GOARCH CGO_ENABLED=1 make -C plugins

# production target (distroless amd64)
FROM --platform=$BUILDPLATFORM gcr.io/distroless/base AS distroless
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /src/out/mini-rproxy /mini-rproxy
COPY --from=build /src/plugins/out /app/plugins
COPY config.example.yml /app/config.yml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]

# development target (use the same binary built earlier)
FROM --platform=$BUILDPLATFORM alpine:latest as dev
RUN apk add --no-cache bash ca-certificates curl gcompat
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /src/out/mini-rproxy /mini-rproxy
COPY --from=build /src/plugins/out /app/plugins
COPY config.example.yml /app/config.yml
EXPOSE 8080
ENTRYPOINT ["/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]
