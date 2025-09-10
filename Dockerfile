ARG BUILDPLATFORM

# platform variant build
FROM --platform=$BUILDPLATFORM golang:1.24.4 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=1

RUN go build -trimpath -ldflags="-s -w" -o /out/mini-rproxy ./cmd

# production target
FROM gcr.io/distroless/static:nonroot AS distroless
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /out/mini-rproxy /bin/mini-rproxy
COPY config.example.yml /app/config.yml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bin/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]

# development target
FROM scratch AS dev
WORKDIR /app
COPY --from=build /out/mini-rproxy /mini-rproxy
COPY config.example.yml /app/config.yml
CMD ["/mini-rproxy", "-config", "/app/config.yml", "-plugindir", "/app/plugins"]
