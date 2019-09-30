GO  := go

# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on

PKGS = $(shell $(GO) list ./... | grep -v /vendor/)

ARCH ?= $(shell go env GOARCH)
OS ?= $(shell go env GOOS)

ALL_TARGETS := cmd/controller-manager
SRC_PREFIX := github.com/cofyc/advanced-statefulset

all: build test
.PHONY: all

build: $(ALL_TARGETS)
.PHONY: all

$(ALL_TARGETS):
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GO) build -o output/bin/$(OS)/$(ARCH)/$@ $(SRC_PREFIX)/$@
.PHONY: $(ALL_TARGETS)

test:
	$(GO) test $(PKGS)
.PHONY: test
