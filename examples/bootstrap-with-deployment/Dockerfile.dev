FROM golang:1.17 as builder

WORKDIR /vcluster

# Install Delve for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

ENV GO111MODULE on
ENV DEBUG true

# Symlink manifests folder to the expected path
RUN ln -s "$(pwd)/manifests" /manifests

ENTRYPOINT ["sleep", "999999999999"]