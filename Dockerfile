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

FROM golang:1.13 as builder
WORKDIR /go/src/github.com/pingcap/advanced-statefulset
ADD . .
RUN make cmd/controller-manager

# For security, we use kubernetes community maintained debian base image.
# https://github.com/kubernetes/kubernetes/blob/master/build/debian-base/
FROM k8s.gcr.io/debian-base:v1.0.0

COPY --from=builder /go/src/github.com/pingcap/advanced-statefulset/output/bin/linux/amd64/cmd/controller-manager  /usr/local/bin/advanced-statefulset-controller-manager
ENTRYPOINT ["/usr/local/bin/advanced-statefulset-controller-manager"]
