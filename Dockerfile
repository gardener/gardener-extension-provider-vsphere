############# builder
FROM golang:1.18.5 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-vsphere
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

############# base
FROM gcr.io/distroless/static-debian11:nonroot AS base

############# gardener-extension-provider-vsphere
FROM base AS gardener-extension-provider-vsphere
WORKDIR /

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-vsphere /gardener-extension-provider-vsphere
ENTRYPOINT ["/gardener-extension-provider-vsphere"]

############# gardener-extension-validator-vsphere
FROM base AS gardener-extension-validator-vsphere
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-validator-vsphere /gardener-extension-validator-vsphere
ENTRYPOINT ["/gardener-extension-validator-vsphere"]

############# gcve-tm-run
FROM debian:11 AS gardener-extension-gcve-tm-run

RUN apt-get update && apt-get install bash wget curl unzip software-properties-common gnupg2 -y; \
    curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -; \
    apt-add-repository "deb [arch=$(dpkg --print-architecture)] https://apt.releases.hashicorp.com $(lsb_release -cs) main"; \
    apt-get update; apt-get install terraform -y; \
    curl -sSL https://sdk.cloud.google.com | bash

COPY --from=builder /go/bin/gcve-setup /gcve-setup
