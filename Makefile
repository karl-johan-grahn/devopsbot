PREFIX := .
PKG_NAME := devopsbot
SHELL := /bin/bash
.SHELLFLAGS := -euxo pipefail -c

export GOPRIVATE=github.com/karl-johan-grahn/devopsbot

REVISION = $(shell git rev-parse --short HEAD)
VERSION = $(shell cat version.txt)

REVISION_FLAG = -X $(shell go list ./version).Revision=$(REVISION)
VERSION_FLAG = -X $(shell go list ./version).Version=$(VERSION)

# Get the string before and after the forward slash of go version arch
GOOS ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\1/')
GOARCH ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\2/')

version.txt:
	@./gen_version.sh > $@

image.iid: version.txt Dockerfile
	@docker build \
	--build-arg TOKEN=$(GITHUB_TOKEN) \
	--build-arg REVISION=$(REVISION) \
	--build-arg VERSION=$(VERSION) \
	--iidfile $@ \
	.

lint:
	@golangci-lint run --disable-all \
		--enable deadcode \
		--enable depguard \
		--enable dupl \
		--enable errcheck \
		--enable goconst \
		--enable gocritic \
		--enable gocyclo \
		--enable gofmt \
		--enable goimports \
		--enable gosec \
		--enable gosimple \
		--enable govet \
		--enable ineffassign \
		--enable misspell \
		--enable nakedret \
		--enable prealloc \
		--enable staticcheck \
		--enable structcheck \
		--enable stylecheck \
		--enable typecheck \
		--enable unconvert \
		--enable unused \
		--enable varcheck

$(PREFIX)/bin/$(PKG_NAME)_%: version.txt go.mod go.sum $(shell find $(PREFIX) -type f -name '*.go')
	GO111MODULE=on GOOS=$(shell echo $* | cut -f1 -d-) GOARCH=$(shell echo $* | cut -f2 -d- | cut -f1 -d.) CGO_ENABLED=0 \
		go build \
			-ldflags "$(REVISION_FLAG) $(VERSION_FLAG)" \
			-o $@ \
			./cmd/$(PKG_NAME)

$(PREFIX)/bin/$(PKG_NAME): $(PREFIX)/bin/$(PKG_NAME)_$(GOOS)-$(GOARCH)
	cp $< $@

build: $(PREFIX)/bin/$(PKG_NAME)

clean:
	@-rm -Rf $(PREFIX)/bin
	@-rm -f $(PREFIX)/*.[ci]id $(PREFIX)/*.tag $(PREFIX)/version.txt $(PREFIX)/c.out

test: build
	go test -v -coverprofile=c.out ./...

.PHONY: all build clean test lint
