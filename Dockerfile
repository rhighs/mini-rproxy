# syntax=docker/dockerfile:experimental
FROM --platform=$BUILDPLATFORM golang:1.24.4 AS build
ARG TARGET_ARCH
ARG BUILDPLATFORM

# cache go mod stuff
RUN --mount=type=cache,target=/go/pkg/mod \
    true

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

# default to building for the platform the builder is running on
ARG GOARCH
RUN if [ -z "$TARGET_ARCH" ]; then \
      case "$BUILDPLATFORM" in \
        *arm64*) export GOARCH=arm64 ;; \
        *amd64*) export GOARCH=amd64 ;; \
        *) export GOARCH=amd64 ;; \
      esac ; \
    else export GOARCH=$TARGET_ARCH ; \
    fi && \
    echo "GOARCH=$GOARCH" > /tmp/target_arch

ARG CC
RUN if [ -n "$TARGET_ARCH" ]; then \
      GOARCH="$TARGET_ARCH"; \
    else \
      case "$BUILDPLATFORM" in \
        *arm64*) GOARCH=arm64 ;; \
        *amd64*) GOARCH=amd64 ;; \
        *) GOARCH=amd64 ;; \
      esac ; \
    fi && \
    echo "GOARCH=$GOARCH" > /tmp/target_arch && \
    GOOS=linux GOARCH=$GOARCH CGO_ENABLED=1 \
      go build -trimpath -ldflags "-s -w" -o /out/mini-rproxy ./cmd

# production target (distroless amd64)
FROM gcr.io/distroless/base AS distroless
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /out/mini-rproxy /mini-rproxy
COPY config.example.yml /app/config.yml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]

# development target (use the same binary built earlier)
FROM alpine:latest as dev
RUN apk add --no-cache bash ca-certificates curl gcompat
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /out/mini-rproxy /mini-rproxy
COPY config.example.yml /app/config.yml
EXPOSE 8080
ENTRYPOINT ["/mini-rproxy", "-config", "/app/config.yml", "-plugindir", "/app/plugins"]
