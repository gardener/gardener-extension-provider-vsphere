############# builder
FROM golang:1.17.11 AS builder

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
FROM eu.gcr.io/gardener-project/gardener/testmachinery/testmachinery-run AS gardener-extension-gcve-tm-run

COPY --from=builder /go/bin/gcve-setup /gcve-setup

RUN  \
  apk update \
  && apk add openvpn bash
