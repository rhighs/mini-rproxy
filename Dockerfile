# build
FROM golang:1.24.4 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mini-rproxy ./cmd

# run (distroless)
FROM gcr.io/distroless/static:nonroot
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /out/mini-rproxy /bin/mini-rproxy
COPY config.example.yml /app/config.yml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bin/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]
