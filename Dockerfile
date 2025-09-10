# build
FROM --platform=linux/amd64 golang:1.24.4 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64 

RUN go build -trimpath -ldflags="-extldflags -s -w" -o /out/mini-rproxy ./cmd

# run (distroless)
FROM gcr.io/distroless/static:nonroot
ENV LISTEN_ADDR=:8080
WORKDIR /app
COPY --from=build /out/mini-rproxy /bin/mini-rproxy
COPY config.example.yml /app/config.yml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bin/mini-rproxy","-config","/app/config.yml","-plugindir","/app/plugins"]
