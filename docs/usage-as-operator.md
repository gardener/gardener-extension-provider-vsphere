# Using the vSphere provider extension with Gardener as operator

The [`core.gardener.cloud/v1alpha1.CloudProfile` resource](https://github.com/gardener/gardener/blob/master/example/30-cloudprofile.yaml) declares a `providerConfig` field that is meant to contain provider-specific configuration.

In this document we are describing how this configuration looks like for VMware vSphere and provide an example `CloudProfile` manifest with minimal configuration that you can use to allow creating vSphere shoot clusters.

## `CloudProfileConfig`

The cloud profile configuration contains information about the real machine image paths in the vSphere environment (image names).
You have to map every version that you specify in `.spec.machineImages[].versions` here such that the vSphere extension knows the image ID for every version you want to offer.

It also contains optional default values for DNS servers that shall be used for shoots.
In the `dnsServers[]` list you can specify IP addresses that are used as DNS configuration for created shoot subnets.

The `dhcpOptions` list allows to specify DHCP options. See [BOOTP Vendor Extensions and DHCP Options](https://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml)
for valid codes (tags) and details about values. The code `15` (domain name) is only allowed for
when using NSX-T 2.5. For NSX-T >= 3.0 use `119` (search domain).

The `dockerDaemonOptions` allow to adjust the docker daemon configuration.
- with `dockerDaemonOptions.httpProxyConf` the content of the proxy configuration file can be set.
See [Docker HTTP/HTTPS proxy](https://docs.docker.com/config/daemon/systemd/#httphttps-proxy) for more details
- with `dockerDaemonOptions.insecureRegistries` insecure registries can be specified. This
should only be used for development or evaluation purposes.


Also, you have to specify several name of NSX-T objects in the constraints.

An example `CloudProfileConfig` for the vSphere extension looks as follows:

```yaml
apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
kind: CloudProfileConfig
namePrefix: my_gardener
defaultClassStoragePolicyName: "vSAN Default Storage Policy"
folder: my-vsphere-vm-folder
regions:
- name: region1
  vsphereHost: my.vsphere.host
  vsphereInsecureSSL: true
  nsxtHost: my.vsphere.host
  nsxtInsecureSSL: true
  transportZone: "my-tz"
  logicalTier0Router: "my-tier0router"
  edgeCluster: "my-edgecluster"
  snatIpPool: "my-snat-ip-pool"
  datacenter: my-vsphere-dc
  zones:
  - name: zone1
    computeCluster: my-vsphere-computecluster1
    # resourcePool: my-resource-pool1 # provide either computeCluster or resourcePool or hostSystem
    # hostSystem: my-host1 # provide either computeCluster or resourcePool or hostSystem
    datastore: my-vsphere-datastore1
    #datastoreCluster: my-vsphere-datastore-cluster # provide either datastore or datastoreCluster
  - name: zone2
    computeCluster: my-vsphere-computecluster2
    # resourcePool: my-resource-pool2 # provide either computeCluster or resourcePool or hostSystem
    # hostSystem: my-host2 # provide either computeCluster or resourcePool or hostSystem
    datastore: my-vsphere-datastore2
    #datastoreCluster: my-vsphere-datastore-cluster # provide either datastore or datastoreCluster
constraints:
  loadBalancerConfig:
    size: MEDIUM
    classes:
    - name: default
      ipPoolName: gardener_lb_vip
# optional DHCP options like 119 (search domain), 42 (NTP), 15 (domain name (only NSX-T 2.5))
#dhcpOptions:
#- code: 15
#  values:
#  - foo.bar.com
#- code: 42
#  values:
#  - 136.243.202.118
#  - 80.240.29.124
#  - 78.46.53.8
#  - 162.159.200.123
dnsServers:
- 10.10.10.11
- 10.10.10.12
machineImages:
- name: flatcar
  versions:
  - version: 3139.2.3
    path: gardener/templates/flatcar-3139.2.3
    guestId: other4xLinux64Guest
#dockerDaemonOptions:
#  httpProxyConf: |
#    [Service]
#    Environment="HTTPS_PROXY=https://proxy.example.com:443"
#  insecureRegistries:
#  - myregistrydomain.com:5000
#  - blabla.mycompany.local
```

## Example `CloudProfile` manifest

Please find below an example `CloudProfile` manifest:

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: CloudProfile
metadata:
  name: vsphere
spec:
  type: vsphere
  providerConfig:
    apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
    kind: CloudProfileConfig
    namePrefix: my_gardener
    defaultClassStoragePolicyName: "vSAN Default Storage Policy"
    folder: my-vsphere-vm-folder
    regions:
    - name: region1
      vsphereHost: my.vsphere.host
      vsphereInsecureSSL: true
      nsxtHost: my.vsphere.host
      nsxtInsecureSSL: true
      transportZone: "my-tz"
      logicalTier0Router: "my-tier0router"
      edgeCluster: "my-edgecluster"
      snatIpPool: "my-snat-ip-pool"
      datacenter: my-vsphere-dc
      zones:
      - name: zone1
        computeCluster: my-vsphere-computecluster1
        # resourcePool: my-resource-pool1 # provide either computeCluster or resourcePool or hostSystem
        # hostSystem: my-host1 # provide either computeCluster or resourcePool or hostSystem
        datastore: my-vsphere-datastore1
        #datastoreCluster: my-vsphere-datastore-cluster # provide either datastore or datastoreCluster
      - name: zone2
        computeCluster: my-vsphere-computecluster2
        # resourcePool: my-resource-pool2 # provide either computeCluster or resourcePool or hostSystem
        # hostSystem: my-host2 # provide either computeCluster or resourcePool or hostSystem
        datastore: my-vsphere-datastore2
        #datastoreCluster: my-vsphere-datastore-cluster # provide either datastore or datastoreCluster
    constraints:
      loadBalancerConfig:
        size: MEDIUM
        classes:
        - name: default
          ipPoolName: gardener_lb_vip
    dnsServers:
    - 10.10.10.11
    - 10.10.10.12
    machineImages:
    - name: coreos
      versions:
      - version: 3139.2.3
        path: gardener/templates/flatcar-3139.2.3
        guestId: other4xLinux64Guest
  kubernetes:
    versions:
    - version: 1.15.4
    - version: 1.16.0
    - version: 1.16.1
  machineImages:
  - name: flatcar
    versions:
    - version: 3139.2.3
  machineTypes:
  - name: std-02
    cpu: "2"
    gpu: "0"
    memory: 8Gi
    usable: true
  - name: std-04
    cpu: "4"
    gpu: "0"
    memory: 16Gi
    usable: true
  - name: std-08
    cpu: "8"
    gpu: "0"
    memory: 32Gi
    usable: true
  regions:
  - name: region1
    zones:
    - name: zone1
    - name: zone2
```

## Which versions of Kubernetes/vSphere are supported

This extension targets Kubernetes >= `v1.15` and vSphere `6.7 U3` or later.

- vSphere CSI driver needs vSphere `6.7 U3` or later,
  and Kubernetes >= `v1.14`
  (see [feature metrics](https://docs.vmware.com/en/VMware-vSphere-Container-Storage-Plug-in/2.0/vmware-vsphere-csp-getting-started/GUID-E59B13F5-6F49-4619-9877-DF710C365A1E.html) for more details)
- vSpere CPI driver needs vSphere `6.7 U3` or later,
  and Kubernetes >= `v1.11`
  (see [cloud-provider-vsphere CPI - Cloud Provider Interface](https://github.com/kubernetes/cloud-provider-vsphere/blob/master/docs/book/cloud_provider_interface.md#which-versions-of-kubernetesvsphere-support-it))

## Supported VM images

Currently, only Gardenlinux and Flatcar (CoreOS fork) are supported.
Virtual Machine Hardware must be version 15 or higher, but images are upgraded
automatically if their hardware has an older version.
