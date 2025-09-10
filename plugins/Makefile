OUT_DIR        ?= out
PLUGIN_SRC_DIR ?= plugins
PLUGIN_OUT_DIR ?= $(OUT_DIR)



PLUGIN_DIRS := $(shell find $(PLUGIN_SRC_DIR) -type d -mindepth 1 -maxdepth 1)
PLUGIN_SOS  := $(patsubst $(PLUGIN_SRC_DIR)/%,$(PLUGIN_OUT_DIR)/%.so,$(PLUGIN_DIRS))

.PHONY: test clean fmt

all: $(PLUGIN_SOS)
$(PLUGIN_OUT_DIR)/%.so: 
	mkdir -p $(PLUGIN_OUT_DIR)
	go build -trimpath -ldflags="-s -w" -buildmode=plugin -o $@ $(wildcard $(PLUGIN_SRC_DIR)/$*/*.go)

test:
	go test ./...
fmt:
	go fmt ./...
clean:
	rm -rf $(OUT_DIR)
