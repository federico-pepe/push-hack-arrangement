BINARY    := arrangement
SRC_DIR   := src
BUILD_DIR := build
GOOS      := linux
GOARCH    := amd64
LDFLAGS   := -s -w

.PHONY: pytest all build build-local preview test vet fmt clean

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
	@test -f testdata/p3.als || { echo "put a Live Set at testdata/p3.als"; exit 1; }
	$(BUILD_DIR)/$(BINARY)-local -set testdata/p3.als -preview $(BUILD_DIR)/preview.png

pytest:
	python3 -m unittest discover -s tests

test:
	cd $(SRC_DIR) && go test ./...

vet:
	cd $(SRC_DIR) && go vet ./...

fmt:
	cd $(SRC_DIR) && go fmt ./...

clean:
	rm -rf $(BUILD_DIR) $(BINARY)
