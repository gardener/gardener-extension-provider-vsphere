
## Create Tanzu Cluster

For gardener a Tanzu Kubernetes „guest” cluster is used. Look here for the vSphere documentation [Provisioning Tanzu Kubernetes Clusters](https://docs.vmware.com/en/VMware-vSphere/7.0/vmware-vsphere-with-tanzu/GUID-2597788E-2FA4-420E-B9BA-9423F8F7FD9F.html)

### Virtual Machine Classes
For gardener the minimum Virtual Machine Classes must set to `best-effort-large`.
### Network Settings
For the deployment it is possible to provision the cluster with a minimal amount of configuration parameter. It is recommended to set the parameter `Default Pod CIDR`, `Default Services CIDR` with values which fit to your enviroment.

### Storage Class settings
The `storageClass` Parameter should be defined to avoid problems during deployment. 

  Example:

    ```yaml
    apiVersion: run.tanzu.vmware.com/v1alpha1      #TKG API endpoint
    kind: TanzuKubernetesCluster                   #required parameter
    metadata:
    name: tkg-cluster-1                          #cluster name, user defined
    namespace: ns1                               #supervisor namespace
    spec:
    distribution:
        version: v1.17				 #resolved kubernetes version
    topology:
        controlPlane:
        count: 1                                 #number of control plane nodes
        class: best-effort-small                 #vmclass for control plane nodes
        storageClass: vsan-default-storage-policy         #storageclass for control plane
        workers:
        count: 3                                 #number of worker nodes
        class: best-effort-large                 #vmclass for worker nodes
        storageClass: vsan-default-storage-policy         #storageclass for worker nodes
    settings:
        network:
        cni:
            name: calico
        services:
            cidrBlocks: ["198.51.100.0/12"]        #Cannot overlap with Supervisor Cluster
        pods:
            cidrBlocks: ["192.0.2.0/16"]           #Cannot overlap with Supervisor Cluster
    ```
