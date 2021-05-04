
## Create Tanzu Cluster

For gardener a Tanzu Kubernetes „guest” cluster is used. Look here for the vSphere documentation [Provisioning Tanzu Kubernetes Clusters](https://docs.vmware.com/en/VMware-vSphere/7.0/vmware-vsphere-with-tanzu/GUID-2597788E-2FA4-420E-B9BA-9423F8F7FD9F.html)

### Virtual Machine Classes
For gardener the minimum Virtual Machine Classes must set to `best-effort-large`.
### Network Settings
For the deployment it is possible to provision the cluster with a minimal amount of configuration parameter. It is recommended to set the parameter `Default Pod CIDR`, `Default Services CIDR` with values which fit to your enviroment.

### Storage Class settings
The `storageClass` Parameter should be defined to avoid problems during deployment. 