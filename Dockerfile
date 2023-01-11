############# builder
FROM golang:1.19.4 AS builder

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
FROM eu.gcr.io/gardener-project/gardener/testmachinery/testmachinery-run:stable AS tm-image
FROM eu.gcr.io/gardener-project/cc/job-image:latest AS gardener-extension-gcve-tm-run

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl; \
    chmod +x ./kubectl && mv ./kubectl /usr/local/bin; \
    export release=$(curl -s https://api.github.com/repos/hashicorp/terraform/releases/latest |  grep tag_name | cut -d: -f2 | tr -d \"\,\v | awk '{$1=$1};1'); \
    curl -LO https://releases.hashicorp.com/terraform/${release}/terraform_${release}_linux_amd64.zip; \
    unzip terraform_${release}_linux_amd64.zip ; \
    mv terraform /usr/local/bin/terraform; \
    curl -sSL https://sdk.cloud.google.com | bash ; \
    wget -qO /usr/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 && chmod a+x /usr/bin/yq

ENV PATH $PATH:/root/google-cloud-sdk/bin

COPY --from=builder /go/bin/gcve-setup /gcve-setup

COPY --from=tm-image /testrunner /testrunner
