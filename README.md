# [Gardener Extension for vSphere provider](https://gardener.cloud)

[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/gardener-tests/pipelines/gardener-extension-provider-vsphere-main/jobs/main-head-update-job/badge)](https://concourse.ci.gardener.cloud/teams/gardener-tests/pipelines/gardener-extension-provider-vsphere-main/jobs/main-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardener-extension-provider-vsphere)](https://goreportcard.com/report/github.com/gardener/gardener-extension-provider-vsphere)

## Overview 
The Gardener Extension for vSphere is a [GEP-1](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md) provider implementation that allows Gardener to leverage vSphere clusters for machine provisioning. 

vSphere is an undeniable class leader for commercially supported virtual machine orchestration. The Gardener extension for vSphere provider compliments this leadership by allowing Gardener to create Kubernetes nodes within vSphere.  

Like other Gardener provider extensions, the vSphere provider pairs with a provider-specific Machine Controller Manager providing node services to Kubernetes clusters. This extension provides complimentary APIs to Gardener. A Gardener-hosted Kubernetes
cluster does not know anything about it's environment (such as bare metal vs. public cloud or within a hyperscaler vs. standalone), only that the MCM abstraction can manage requests such as cluster autoscaling. 

An example for a `ControllerRegistration` resource that can be used to register this controller to Gardener can be found [here](example/controller-registration.yaml).

Please find more information regarding the extensibility concepts and the architecture details in the GEP-1 proposal. 

## Use Cases
The primary use case for this extension is organizations who wish to deploy a substantial Gardener landscape and use vSphere for data center fleet management. We intentionally sidestep prescribing any particular extension as this is
an intimately local determination and the benefits of different solutions are more than adequately debated in industry literature.

While we may inadvertently duplicate some documentation in the mainline Gardener documentation, it is only to reduce tedium as new evaluators and developers come up-to-speed with the concepts relevant to successful deployment.
We refer directly to the mainline Gardener documentation for the most up-to-date information. 

## Supported Kubernetes versions

This extension controller supports the following Kubernetes versions:

| Version         | Support | Conformance test results                                                                                                                                                                                                   |
|-----------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Kubernetes 1.28 | 1.28.0+ | N/A                                                                                                                                                                                                                        |
| Kubernetes 1.27 | 1.27.0+ | N/A                                                                                                                                                                                                                        |
| Kubernetes 1.26 | 1.26.0+ | [![Gardener v1.26 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.26%20vSphere/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.26%20vSphere) |
| Kubernetes 1.25 | 1.25.0+ | [![Gardener v1.25 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.25%20vSphere/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.25%20vSphere) |
| Kubernetes 1.24 | 1.24.0+ | [![Gardener v1.24 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.24%20vSphere/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.24%20vSphere) |

Older versions of the extension [(`v0.16.0` and earlier)](https://github.com/gardener/gardener-extension-provider-vsphere/releases/tag/v0.16.0) are supported prior to current releases.

Please take a look [here](https://github.com/gardener/gardener/blob/master/docs/usage/supported_k8s_versions.md) to see which versions are supported by Gardener in general.

----
## Deployment patterns
As with any production software, deployment of Gardener and this extension should be considered in the context of both lifecycle and automation. Orgs should aspire to have apply 

## How to start using or developing this extension controller locally

You can run the controller locally on your machine by executing `make start`.

Static code checks and tests can be executed by running `make verify`. We are using Go modules for Golang package dependency management and [Ginkgo](https://github.com/onsi/ginkgo)/[Gomega](https://github.com/onsi/gomega) for testing.

## Feedback and Support

Feedback and contributions are always welcome. Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extension-provider-vsphere/issues) or join our [Slack channel #gardener](https://kubernetes.slack.com/messages/gardener) (please invite yourself to the Kubernetes workspace [here](http://slack.k8s.io)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* ["Gardener Project Update" blog on kubernetes.io](https://kubernetes.io/blog/2019/12/02/gardener-project-update/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)
* [GEP-4 (New `core.gardener.cloud/v1beta1` API)](https://github.com/gardener/gardener/blob/master/docs/proposals/04-new-core-gardener-cloud-apis.md)
* [Extensibility API documentation](https://github.com/gardener/gardener/tree/master/docs/extensions)
* [Gardener Extensions Golang library](https://godoc.org/github.com/gardener/gardener/extensions/pkg)
* [Gardener API Reference](https://gardener.cloud/api-reference/)
