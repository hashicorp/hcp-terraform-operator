# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# This Dockerfile contains multiple targets.
# Use 'docker build --target=<name> .' to build one.
#
# Every target has a BIN_NAME argument that must be provided via --build-arg=BIN_NAME=<name>
# when building.

ARG GO_VERSION=1.20

# ===================================
#
#   Non-release images.
#
# ===================================


# dev-builder compiles the binary
# -----------------------------------
FROM golang:$GO_VERSION as dev-builder

ARG BIN_NAME
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -o $BIN_NAME main.go

# dev runs the binary from devbuild
# -----------------------------------
FROM gcr.io/distroless/static:nonroot as dev

ARG BIN_NAME
ARG PRODUCT_VERSION

LABEL version=$PRODUCT_VERSION

WORKDIR /
COPY --from=dev-builder /build/$BIN_NAME .
USER 65532:65532

ENTRYPOINT ["/$BIN_NAME"]

# ===================================
#
#   Release images.
#
# ===================================


# default release image
# -----------------------------------
FROM gcr.io/distroless/static:nonroot AS release-default

ARG BIN_NAME
ARG PRODUCT_VERSION
ARG PRODUCT_REVISION
ARG PRODUCT_NAME=$BIN_NAME
ARG TARGETOS
ARG TARGETARCH

ENV BIN_NAME=$BIN_NAME

LABEL maintainer="Team Terraform Ecosystem - Kuberhentes <team-tf-k8s@hashicorp.com>"
LABEL version=$PRODUCT_VERSION
LABEL revision=$PRODUCT_REVISION

COPY LICENSE /licenses/copyright.txt
COPY dist/$TARGETOS/$TARGETARCH/$BIN_NAME /bin/

USER 65532:65532

ENTRYPOINT ["/$BIN_NAME"]

# ===================================
#
#   Set default target to 'dev'.
#
# ===================================
FROM dev
