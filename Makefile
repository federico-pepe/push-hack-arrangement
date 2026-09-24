BINARY    := arrangement
SRC_DIR   := src
BUILD_DIR := build
GOOS      := linux
GOARCH    := amd64
LDFLAGS   := -s -w

.PHONY: all build build-local preview test vet fmt clean

all: build

build:
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
	  go build -ldflags "$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY) .
	@cp $(BUILD_DIR)/$(BINARY) $(BINARY)

build-local:
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && go build -o ../$(BUILD_DIR)/$(BINARY)-local .

preview: build-local
	$(BUILD_DIR)/$(BINARY)-local -preview $(BUILD_DIR)/preview.png

test:
	cd $(SRC_DIR) && go test ./...

vet:
	cd $(SRC_DIR) && go vet ./...

fmt:
	cd $(SRC_DIR) && go fmt ./...

clean:
	rm -rf $(BUILD_DIR) $(BINARY)
