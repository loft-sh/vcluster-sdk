# Build the manager binary
FROM golang:1.25 as builder

# Make sure we use go modules
WORKDIR /vcluster

# Copy the Go Modules manifests
COPY . .

# Build cmd
RUN CGO_ENABLED=0 GO111MODULE=on go build -mod vendor -o /plugin e2e/test_plugin/main.go

# we use alpine for easier debugging
FROM alpine

# Set root path as working directory
WORKDIR /

RUN mkdir -p /plugin

COPY --from=builder /plugin /plugin/plugin
COPY e2e/test_plugin/manifests/ /plugin/manifests
