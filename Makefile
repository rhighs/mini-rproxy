APP            ?= mini-rproxy
PLUGIN_OUT_DIR ?= plugins/out
OUT_DIR        ?= out

GO_BUILD_FLAGS ?= -trimpath -ldflags="-s -w"

.PHONY: build run clean fmt
.PHONY: docker-build docker-run docker-stop docker-run-echo docker-stop-echo

all: build
build:
	mkdir -p $(OUT_DIR)
	go build $(GO_BUILD_FLAGS) -o $(OUT_DIR)/$(APP) ./cmd
run: build
	./$(OUT_DIR)/$(APP) -config ./config.example.yml -verbose -plugindir $(PLUGIN_OUT_DIR) 

fmt:
	go fmt ./...

clean:
	rm -rf $(OUT_DIR)

DOCKER_PLATFORM ?= linux/arm64
DOCKER_TARGET   ?= dev

docker-build: plugins
	docker buildx build --platform=$(DOCKER_PLATFORM) --target=$(DOCKER_TARGET) -t local/$(APP):latest --no-cache .
docker-run:
	docker run --platform=$(DOCKER_PLATFORM) --rm -p 8080:8080 --name $(APP) local/$(APP):latest
docker-stop:
	docker stop $(APP)
docker-run-echo:
	docker run -d --rm --name  $(APP)-echo -p 9999:80 ealen/echo-server
docker-stop-echo:
	docker stop $(APP)-echo || true
