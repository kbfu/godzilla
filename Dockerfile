FROM golang:1.21.4 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

ADD . /godzilla
WORKDIR /godzilla

RUN export GOOS=${TARGETOS} && \
    export GOARCH=${TARGETARCH}

RUN CGO_ENABLED=0 go build

FROM alpine:3.18

COPY --from=builder godzilla/godzilla /godzilla/godzilla
WORKDIR /godzilla
RUN apk add git