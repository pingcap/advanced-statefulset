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

ARCH ?= $(shell go env GOARCH)
OS ?= $(shell go env GOOS)

ALL_TARGETS := cmd/controller-manager
SRC_PREFIX := github.com/pingcap/advanced-statefulset
GIT_VERSION = $(shell ./hack/version.sh | awk -F': ' '/^GIT_VERSION:/ {print $$2}')

all: build
.PHONY: all

verify: verify-client
	./hack/verify-all.sh
.PHONY: verify

verify-client:
	make -C client verify
.PHONY: verify-client

build: $(ALL_TARGETS)
.PHONY: all

$(ALL_TARGETS):
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GO) build -ldflags "${LDFLAGS}" -o output/bin/$(OS)/$(ARCH)/$@ $(SRC_PREFIX)/$@
.PHONY: $(ALL_TARGETS)

test: test-client
	hack/make-rules/test.sh $(WHAT)
.PHONY: test

test-client:
	make -C client test
.PHONY: test-client

test-integration: openapi-spec
	@echo "skip integration tests now, ref https://github.com/kubernetes/kubernetes/issues/119220"
	# hack/make-rules/test-integration.sh $(WHAT)
.PHONY: test-integration

e2e:
	hack/e2e.sh
.PHONY: e2e

e2e-examples:
	hack/e2e-examples.sh
.PHONY: e2e-examples

openapi-spec:
	hack/generate-kube-openapi.sh
.PHONY: openapi-spec
