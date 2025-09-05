APP            ?= mini-rproxy
BIN_DIR        ?= bin
PLUGIN_SRC_DIR ?= plugins
PLUGIN_OUT_DIR ?= $(BIN_DIR)/plugins

GO_BUILD_FLAGS ?= -trimpath -ldflags="-s -w"

PLUGIN_DIRS := $(shell find $(PLUGIN_SRC_DIR) -type d -mindepth 1 -maxdepth 1)
PLUGIN_SOS  := $(patsubst $(PLUGIN_SRC_DIR)/%,$(PLUGIN_OUT_DIR)/%.so,$(PLUGIN_DIRS))

.PHONY: build plugins run clean fmt run-echo stop-echo

build:
	mkdir -p $(BIN_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(APP) ./cmd

plugins: $(PLUGIN_SOS)

$(PLUGIN_OUT_DIR)/%.so: 
	mkdir -p $(PLUGIN_OUT_DIR)
	go build $(GO_BUILD_FLAGS) -buildmode=plugin -o $@ $(wildcard $(PLUGIN_SRC_DIR)/$*/.go) $(wildcard $(PLUGIN_SRC_DIR)/$*/*.go)

run: build plugins
	./$(BIN_DIR)/$(APP) -config ./config.example.yml -plugindir $(PLUGIN_OUT_DIR) -verbose

fmt:
	go fmt ./...

run-echo:
	docker run -d --rm --name mini-rproxy-echo -p 9999:80 ealen/echo-server

stop-echo:
	docker stop mini-rproxy-echo || true

clean:
	rm -rf $(BIN_DIR)