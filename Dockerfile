FROM golang:1.12-alpine as builder
WORKDIR /go/src/github.com/cofyc/advanced-statefulset
ADD . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o output/bin/linux/amd64/cmd/controller-manager github.com/cofyc/advanced-statefulset/cmd/controller-manager

FROM alpine:latest

COPY --from=builder /go/src/github.com/cofyc/advanced-statefulset/output/bin/linux/amd64/cmd/controller-manager  /usr/local/bin/controller-manager
ENTRYPOINT ["/usr/local/bin/controller-manager"]
