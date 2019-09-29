GO  := go

# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on

PKGS = $(shell $(GO) list ./... | grep -v /vendor/)

all: test
.PHONY: all

test:
	$(GO) test $(PKGS)
.PHONY: test
