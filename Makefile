APP_NAME := cfddns
CURRENT_PLATFORM := $(shell printf '%s-%s' $$(go env GOOS GOARCH))
VERSION := $(shell date '+%Y.%-m.%-d')-$(shell git rev-parse --short HEAD)
PLATFORMS := $(sort darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-arm windows-amd64 windows-arm64 $(CURRENT_PLATFORM))

MAKEFLAGS += -j
.DEFAULT_GOAL := build-current
.PHONY: clean update build build-current release lint test $(PLATFORMS)

clean:
	@rm -rf dist/

update:
	@go get -u ./cmd
	@go mod tidy

$(PLATFORMS): OUTPUT=$(APP_NAME)-$@-$(VERSION)$(if $(findstring windows,$@),.exe,)
$(PLATFORMS): export GOOS=$(word 1,$(subst -, ,$@))
$(PLATFORMS): export GOARCH=$(word 2,$(subst -, ,$@))
$(PLATFORMS):
	@echo $(OUTPUT)
	@$(if $(filter linux-arm,$@),export GOARM=5,)
	@go build \
		-C cmd \
		-trimpath \
		-buildvcs=false \
		-ldflags '-s -w -X main.version=$(VERSION)' \
		-o '../dist/$(OUTPUT)'

build: $(PLATFORMS)

build-current: $(CURRENT_PLATFORM)

release: lint clean build
	@find dist -type f ! -name '*.exe' | parallel 'xz -zv {}'
	@find dist -type f -name '*.exe' | parallel 'zip -m {}.zip {}'
	@git tag -f 'v$(VERSION)'

lint:
	@go vet ./cmd
	@-golangci-lint run
	@gofmt -d ./cmd

test:
	@go test ./cmd --cover -v
