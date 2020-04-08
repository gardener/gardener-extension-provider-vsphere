############# builder
FROM golang:1.13.9 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-vsphere
COPY . .
RUN make install

############# base
FROM alpine:3.11.3 AS base

############# gardener-extension-provider-vsphere
FROM base AS gardener-extension-provider-vsphere

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-vsphere /gardener-extension-provider-vsphere
ENTRYPOINT ["/gardener-extension-provider-vsphere"]

############# gardener-extension-validator-vsphere
FROM base AS gardener-extension-validator-vsphere

COPY --from=builder /go/bin/gardener-extension-validator-vsphere /gardener-extension-validator-vsphere
ENTRYPOINT ["/gardener-extension-validator-vsphere"]
