############# builder
FROM golang:1.19.1 AS builder

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
FROM eu.gcr.io/gardener-project/cc/job-image:1.1545.0 AS cli-job-image
FROM eu.gcr.io/gardener-project/gardener/testmachinery/testmachinery-run:stable AS tm-image
FROM debian:11 AS gardener-extension-gcve-tm-run

RUN apt-get update ; DEBIAN_FRONTEND=noninteractive apt-get install -yq bash wget curl gnupg2 python3 python3-pip lsb-release software-properties-common; \
    curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -; \
    apt-add-repository "deb [arch=$(dpkg --print-architecture)] https://apt.releases.hashicorp.com $(lsb_release -cs) main"; \
    apt-get update ; DEBIAN_FRONTEND=noninteractive apt-get install -yq kubernetes-client git openssh-client unzip software-properties-common terraform; \
    curl -sSL https://sdk.cloud.google.com | bash

COPY --from=builder /go/bin/gcve-setup /gcve-setup

COPY --from=cli-job-image /cc/utils /cc/utils
COPY --from=cli-job-image /bin/component-cli /bin/component-cli
COPY --from=cli-job-image /metadata/VERSION /metadata/VERSION
RUN pip3 install --upgrade --no-cache-dir \
  pip \
  wheel \
&& pip3 install --upgrade --no-cache-dir \
  --find-links /cc/utils/dist \
  gardener-cicd-libs==$(cat /metadata/VERSION) \
  gardener-cicd-cli==$(cat /metadata/VERSION) \
  gardener-cicd-dso==$(cat /metadata/VERSION) \
  pycryptodome

COPY --from=tm-image /testrunner /testrunner
