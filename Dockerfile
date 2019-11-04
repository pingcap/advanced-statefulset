FROM golang:1.13-alpine as builder
WORKDIR /go/src/github.com/cofyc/advanced-statefulset
ADD . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=off go build -o output/bin/linux/amd64/cmd/controller-manager github.com/cofyc/advanced-statefulset/cmd/controller-manager

# For security, we use kubernetes community maintained debian base image.
# https://github.com/kubernetes/kubernetes/blob/master/build/debian-base/
FROM k8s.gcr.io/debian-base:v1.0.0

COPY --from=builder /go/src/github.com/cofyc/advanced-statefulset/output/bin/linux/amd64/cmd/controller-manager  /usr/local/bin/controller-manager
ENTRYPOINT ["/usr/local/bin/controller-manager"]
