# Using the vSphere provider extension with Gardener as end-user

The [`core.gardener.cloud/v1alpha1.Shoot` resource](https://github.com/gardener/gardener/blob/master/example/90-shoot.yaml) declares a few fields that are meant to contain provider-specific configuration.

In this document we are describing how this configuration looks like for VMware vSphere and provide an example `Shoot` manifest with minimal configuration that you can use to create an vSphere cluster (modulo the landscape-specific information like cloud profile names, secret binding names, etc.).

## Provider secret data

Every shoot cluster references a `SecretBinding` which itself references a `Secret`, and this `Secret` contains the provider credentials of your vSphere tenant.
It contains two authentication sets. One for the vSphere host and another for the NSX-T host, which is needed to set up the network infrastructure.
This `Secret` must look as follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: core-vsphere
  namespace: garden-dev
type: Opaque
data:
  vsphereHost: base64(vsphere-host)
  vsphereUsername: base64(vsphere-username)
  vspherePassword: base64(vsphere-password)
  vsphereInsecureSSL: base64("true"|"false")
  nsxtHost: base64(NSX-T-host)
  nsxtUsername: base64(NSX-T-username)
  nsxtPassword: base64(NSX-T-password)
  nsxtInsecureSSL: base64("true"|"false")
```

Here `base64(...)` are only a placeholders for the Base64 encoded values.

## `InfrastructureConfig`

The infrastructure configuration is used for advanced scenarios only.
Nodes on all zones are using IP addresses from the common nodes network as the network is managed by NSX-T.
The infrastructure controller will create several network objects using NSX-T. A network segment is used as the subnet
for the VMs (nodes), a tier-1 gateway, a DHCP server, and a SNAT for the nodes.

An example `InfrastructureConfig` for the vSphere extension looks as follows.
You only need to specify it, if you either want to use an existing Tier-1 gateway and load balancer service pair
or if you want to overwrite the automatic selection of the NSX-T version.

```yaml
infrastructureConfig:
  apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
  kind: InfrastructureConfig
  #overwriteNSXTInfraVersion: '1'
  #networks:
  #  tier1GatewayPath: /infra/tier-1s/tier1gw-b8213651-9659-4180-8bfd-1e16228e8dcb
  #  loadBalancerServicePath: /infra/lb-services/708c5cb1-e5d0-4b16-906f-ec7177a1485d
```

### Advanced configuration settings

#### Section networks

By default, the infrastructure controller creates a separate Tier-1 gateway for each shoot cluster
and the cloud controller manager (`vsphere-cloud-provider`) creates a load balancer service.

If an existing Tier-1 gateway should be used, you can specify its 'path'. In this case, there
must also be a load balancer service defined for this tier-1 gateway and its 'path' needs to be specified, too.
In the NSX-T manager UI, the path of the tier-1 gateway can be found at `Networking / Tier-1 Gateways`.
Then select `Copy path to clipboard` from the context menu of the tier-1 gateway 
(click on the three vertical dots on the left of the row). Do the same with the 
corresponding load balancer at `Networking / Load balancing / Tab Load Balancers`
For security reasons the referenced Tier-1 gateway in NSX-T must have a tag with scope `authorized-shoots` and its
tag value consists of a comma-separated list of the allowed shoot names (optionally with wildcard `*`)

Example:

```yaml
infrastructureConfig:
  apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
  kind: InfrastructureConfig
  networks:
    tier1GatewayPath: /infra/tier-1s/tier1gw-b8213651-9659-4180-8bfd-1e16228e8dcb
    loadBalancerServicePath: /infra/lb-services/708c5cb1-e5d0-4b16-906f-ec7177a1485d
```

Please ensure, that the worker nodes cidr (shoot manifest `spec.networking.nodes`) do not overlap with
other existing segments of the selected tier-1 gateway.

#### Option overwriteNSXTInfraVersion
The option `overwriteNSXTInfraVersion` can be used to change the network objects created during the initial infrastructure creation. 
By default the infra-version is automatically selected according to the NSX-T version. The infra-version `'1'` is used 
for NSX-T 2.5, and infra-version `'2'` for NSX-T versions >= 3.0. The difference is creation of the the logical DHCP server.
For NSX-T 2.5, only the DHCP server of the "Advanced API" is usable. For NSX-T >= 3.0 the new DHCP server is default, 
but for special purposes infra-version `'1'` is also allowed.

## `ControlPlaneConfig`

The control plane configuration mainly contains values for the vSphere-specific control plane components.
Today, the only component deployed by the vSphere extension is the `cloud-controller-manager`.

An example `ControlPlaneConfig` for the vSphere extension looks as follows:

```yaml
apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
kind: ControlPlaneConfig
loadBalancerClasses:
  - name: mypubliclbclass
  - name: myprivatelbclass
    ipPoolName: pool42 # optional overwrite
loadBalancerSize: SMALL
cloudControllerManager:
  featureGates:
    CustomResourceValidation: true
```

The `loadBalancerClasses` optionally defines the load balancer classes to be used.
The specified names must be defined in the constraints section of the cloud profile.
If the list contains a load balancer named "default", it is used as the default load balancer.
Otherwise the first one is also the default.
If no classes are specified the default load balancer class is used as defined in the cloud profile constraints section.
If the ipPoolName is overwritten, the IP pool object in NSX-T must have a tag with scope `authorized-shoots` and its
tag value consists of a comma-separated list of the allowed shoot names (optionally with wildcard `*`)

The `loadBalancerSize` is optional and overwrites the default value specified in the cloud profile config.
It must be one of the values `SMALL`, `MEDIUM`, or `LARGE`. `SMALL` can manage 10 service ports,
`MEDIUM` 100, and `LARGE` 1000. 

The `cloudControllerManager.featureGates` contains an optional map of explicitly enabled or disabled feature gates.
For production usage it's not recommend to use this field at all as you can enable alpha features or disable beta/stable features, potentially impacting the cluster stability.
If you don't want to configure anything for the `cloudControllerManager` simply omit the key in the YAML specification.

## Example `Shoot` manifest (one availability zone)

Please find below an example `Shoot` manifest for one availability zone:

```yaml
apiVersion: core.gardener.cloud/v1alpha1
kind: Shoot
metadata:
  name: johndoe-vsphere
  namespace: garden-dev
spec:
  cloudProfileName: vsphere
  region: europe-1
  secretBindingName: core-vsphere
  provider:
    type: vsphere
   
    #infrastructureConfig:
    #  apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
    #  kind: InfrastructureConfig
    #  overwriteNSXTInfraVersion: '1'

    controlPlaneConfig:
      apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
      kind: ControlPlaneConfig
    #  loadBalancerClasses:
    #  - name: mylbclass

    workers:
    - name: worker-xoluy
      machine:
        type: std-04
      minimum: 2
      maximum: 2
      zones:
      - europe-1a
  networking:
    nodes: 10.250.0.0/16
    type: calico
  kubernetes:
    version: 1.16.1
  maintenance:
    autoUpdate:
      kubernetesVersion: true
      machineImageVersion: true
  addons:
    kubernetes-dashboard:
      enabled: true
    nginx-ingress:
      enabled: true
```
