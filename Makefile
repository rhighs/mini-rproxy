APP=mini-rproxy
# GOOS=linux GOARCH=amd64 CGO_ENABLED=0 
build:
	go build -trimpath -ldflags="-s -w" -o bin/$(APP) ./mini-rproxy

docker:
	docker build -t $(APP):latest .

run: build
	./bin/$(APP) -config ./config.example.yml

fmt:
	go fmt ./...
