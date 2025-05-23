# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# This Dockerfile contains multiple targets.
# Use 'docker build --target=<name> .' to build one.
#
# Every target has a BIN_NAME argument that must be provided via --build-arg=BIN_NAME=<name>
# when building.

ARG GO_VERSION=1.24.3

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

ENV BIN_NAME=$BIN_NAME

WORKDIR /build

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY api/ api/
COPY cmd/main.go cmd/main.go
COPY internal/ internal/
COPY version/ version/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -o $BIN_NAME cmd/main.go

# dev runs the binary from devbuild
# -----------------------------------
FROM gcr.io/distroless/static:nonroot as dev

ARG BIN_NAME
ARG PRODUCT_VERSION

ENV BIN_NAME=$BIN_NAME

LABEL version=$PRODUCT_VERSION

WORKDIR /
COPY --from=dev-builder /build/$BIN_NAME .
USER 65532:65532

ENTRYPOINT ["/bin/sh", "-c", "/$BIN_NAME"]

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
ARG TARGETOS
ARG TARGETARCH

ENV BIN_NAME=$BIN_NAME

LABEL maintainer="Terraform Ecosystem - Hybrid Cloud Team <hcp-tf-operator@hashicorp.com>"
LABEL version=$PRODUCT_VERSION
LABEL revision=$PRODUCT_REVISION

WORKDIR /
COPY LICENSE /licenses/copyright.txt
COPY dist/$TARGETOS/$TARGETARCH/$BIN_NAME .

USER 65532:65532

ENTRYPOINT ["/bin/sh", "-c", "/$BIN_NAME"]

# Red Hat UBI release image
# -----------------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5 AS release-ubi

ARG BIN_NAME
ARG PRODUCT_VERSION
ARG PRODUCT_REVISION
ARG TARGETOS
ARG TARGETARCH

ENV BIN_NAME=$BIN_NAME

LABEL name="HCP Terraform Operator"
LABEL vendor="HashiCorp"
LABEL release=$PRODUCT_REVISION
LABEL summary="HCP Terraform Operator for Kubernetes allows managing HCP Terraform / Terraform Enterprise resources via Kubernetes Custom Resources."
LABEL description="HCP Terraform Operator for Kubernetes allows managing HCP Terraform / Terraform Enterprise resources via Kubernetes Custom Resources."

LABEL maintainer="HashiCorp <hcp-tf-operator@hashicorp.com>"
LABEL version=$PRODUCT_VERSION
LABEL revision=$PRODUCT_REVISION

WORKDIR /
COPY LICENSE /licenses/copyright.txt
COPY dist/$TARGETOS/$TARGETARCH/$BIN_NAME .

USER 65532:65532

ENTRYPOINT ["/bin/sh", "-c", "/$BIN_NAME"]

# ===================================
#
#   Set default target to 'dev'.
#
# ===================================
FROM dev
