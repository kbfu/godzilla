# Multi-stage docker build
# Build stage
FROM golang:1.21.4 AS builder

ARG TARGETOS=linux
ARG TARGETARCH

ADD . /godzilla
WORKDIR /godzilla

RUN export GOOS=${TARGETOS} && \
    export GOARCH=${TARGETARCH}

RUN CGO_ENABLED=0 go build

FROM alpine:3.18

# Install generally useful things
COPY --from=builder godzilla /godzilla
WORKDIR /godzilla