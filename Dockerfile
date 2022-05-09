############# builder
FROM golang:1.17.9 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-vsphere
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

############# base
FROM alpine:3.15.4 AS base

############# gardener-extension-provider-vsphere
FROM base AS gardener-extension-provider-vsphere

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-vsphere /gardener-extension-provider-vsphere
ENTRYPOINT ["/gardener-extension-provider-vsphere"]

############# gardener-extension-validator-vsphere
FROM base AS gardener-extension-validator-vsphere

COPY --from=builder /go/bin/gardener-extension-validator-vsphere /gardener-extension-validator-vsphere
ENTRYPOINT ["/gardener-extension-validator-vsphere"]
