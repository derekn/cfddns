APP_NAME := cfddns
CURRENT_PLATFORM := $(shell printf '%s-%s' $$(go env GOOS GOARCH))
VERSION := $(shell date '+%Y.%-m.%-d')
GIT_COMMIT := $(shell git rev-parse --short HEAD)
PLATFORMS := $(sort darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-arm windows-amd64 windows-arm64 $(CURRENT_PLATFORM))

MAKEFLAGS += -j
.DEFAULT_GOAL := build
.PHONY: clean update build build-all release lint test $(PLATFORMS)

clean:
	@rm -rf dist/

update:
	@go get -u ./cmd
	@go mod tidy

$(PLATFORMS): OUTPUT=$(APP_NAME)-$@$(if $(findstring windows,$@),.exe,)
$(PLATFORMS): export GOOS=$(word 1,$(subst -, ,$@))
$(PLATFORMS): export GOARCH=$(word 2,$(subst -, ,$@))
$(PLATFORMS):
	@$(if $(filter linux-arm,$@),GOARM=5,) \
	go build \
		-C cmd \
		-trimpath \
		-buildvcs=false \
		-ldflags '-s -w -X main.version=$(VERSION)+$(GIT_COMMIT)' \
		-o '../dist/$(OUTPUT)'
	@echo $(OUTPUT)

build: $(CURRENT_PLATFORM)

build-all: $(PLATFORMS)

release: lint clean build-all
	@find dist -type f ! -name '*.exe' | parallel 'bzip2 -z9v {}'
	@find dist -type f -name '*.exe' | parallel 'zip -m9 {.}.zip {}'
	@rhash -r --printf '%{sha-256}  %f\n' dist > dist/SHA256SUMS
	@git tag -f 'v$(VERSION)'

lint:
	@go vet ./cmd
	@-golangci-lint run
	@gofmt -d ./cmd

test:
	@go test ./cmd --cover -v
