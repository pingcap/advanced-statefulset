# Copyright 2019 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

GO  := go

# Enable GO111MODULE=off explicitly, enable it with GO111MODULE=on when necessary.
export GO111MODULE := off

ARCH ?= $(shell go env GOARCH)
OS ?= $(shell go env GOOS)

ALL_TARGETS := cmd/controller-manager
SRC_PREFIX := github.com/pingcap/advanced-statefulset
GIT_VERSION = $(shell ./hack/version.sh | awk -F': ' '/^GIT_VERSION:/ {print $$2}')

# in GOPATH mode, we must use the full path name related to $GOPATH
# https://github.com/golang/go/issues/19000
ifneq ($(VERSION),)
    LDFLAGS += -X $(SRC_PREFIX)/vendor/k8s.io/component-base/version.gitVersion=${VERSION}
else
    LDFLAGS += -X $(SRC_PREFIX)/vendor/k8s.io/component-base/version.gitVersion=${GIT_VERSION}
endif

all: build
.PHONY: all

verify:
	./hack/verify-all.sh 
.PHONY: verify

build: $(ALL_TARGETS)
.PHONY: all

$(ALL_TARGETS):
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GO) build -ldflags "${LDFLAGS}" -o output/bin/$(OS)/$(ARCH)/$@ $(SRC_PREFIX)/$@
.PHONY: $(ALL_TARGETS)

test:
	hack/make-rules/test.sh $(WHAT)
.PHONY: test

test-integration: vendor/k8s.io/kubernetes/pkg/generated/openapi/zz_generated.openapi.go
	hack/make-rules/test-integration.sh $(WHAT)
.PHONY: test-integration

e2e:
	hack/e2e.sh
.PHONY: e2e

e2e-examples:
	hack/e2e-examples.sh
.PHONY: e2e-examples

vendor/k8s.io/kubernetes/pkg/generated/openapi/zz_generated.openapi.go:
	hack/generate-kube-openapi.sh
.PHONY: vendor/k8s.io/kubernetes/pkg/generated/openapi/zz_generated.openapi.go
