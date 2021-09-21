module github.com/gardener/gardener-extension-provider-vsphere

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/gardener/etcd-druid v0.5.0
	github.com/gardener/gardener v1.32.0
	github.com/gardener/machine-controller-manager v0.37.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.6.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.1.2
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmware/go-vmware-nsxt v0.0.0-20200114231430-33a5af043f2e
	github.com/vmware/vsphere-automation-sdk-go/lib v0.3.1
	github.com/vmware/vsphere-automation-sdk-go/runtime v0.3.1
	github.com/vmware/vsphere-automation-sdk-go/services/nsxt v0.4.0
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/apiserver v0.22.1
	k8s.io/autoscaler v0.0.0-20190805135949-100e91ba756e
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider-vsphere v1.1.0
	k8s.io/code-generator v0.22.1
	k8s.io/component-base v0.22.1
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.21.2
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gardener/gardener-resource-manager/api => github.com/gardener/gardener-resource-manager/api v0.25.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.11.0 // keep this value in sync with k8s.io/client-go
	k8s.io/api => k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.2
	k8s.io/apiserver => k8s.io/apiserver v0.21.2
	k8s.io/client-go => k8s.io/client-go v0.21.2
	k8s.io/code-generator => k8s.io/code-generator v0.21.2
	k8s.io/component-base => k8s.io/component-base v0.21.2
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.2
)

// needed for infra-cli and load balancer cleanup
replace k8s.io/cloud-provider-vsphere => github.com/MartinWeindel/cloud-provider-vsphere v1.0.1-0.20210910074917-6559ac3f3bcf
