# vSphere / NSX-T Preparation for Gardener Extension "vSphere Provider"

- [vSphere / NSX-T Preparation for Gardener Extension "vSphere Provider"](#vsphere--nsx-t-preparation-for-gardener-extension-vsphere-provider)
  - [vSphere Preparation](#vsphere-preparation)
    - [Create Folders](#create-folders)
    - [Upload VM Templates for Worker Nodes](#upload-vm-templates-for-worker-nodes)
    - [Prepare for Kubernetes Zones and Regions](#prepare-for-kubernetes-zones-and-regions)
      - [Create Resource Pool(s)](#create-resource-pools)
      - [Tag Regions and Zones](#tag-regions-and-zones)
      - [Storage policies](#storage-policies)
        - [Tag Zone Storages](#tag-zone-storages)
        - [Create or clone VM Storage Policy](#create-or-clone-vm-storage-policy)
  - [NSX-T Prepartion](#nsx-t-prepartion)
    - [Create IP pools](#create-ip-pools)
      - [Sizing the IP pools](#sizing-the-ip-pools)
    - [Check edge cluster sizing](#check-edge-cluster-sizing)
  - [Get VDS UUIDs](#get-vds-uuids)

Several preparational steps are necessary for VMware vSphere and NSX-T, before this extension can be used
to create Gardener shoot clusters.

The main version target of this extension is vSphere 7.x together with NSX-T 3.x.
The recommended environment is a system setup with VMware Cloud Foundation (VCF) 4.1.
Older versions like vSphere 6.7U3 with NSX-T 2.5 or 3.0 should still work, but are not tested extensively.

## vSphere Preparation

### User and Role Creation

This extension needs credentials for both the vSphere/vCenter and the NSX-T endpoints.
This section guides through the creation of appropriate roles and users.

#### vCenter/vSphere

The vCenter/vSphere user used for this provider should have been assigned to a role including these permissions
(use vCenter/vSphere Client / Menu Administration / Access Control / Role to define a role and assign it to the user
with `Global Permissions`)

* Datastore
    * Allocate space
    * Browse datastore
    * Low level file operations
    * Remove file
    * Update virtual machine files
    * Update virtual machine metadata
* Global
    * Cancel task
    * Manage custom attributes
    * Set custom attribute
* Network
    * Assign network
* Resource
    * Assign virtual machine to resource pool
* Tasks
    * Create task
    * Update task
* vApp
    * Add virtual machine
    * Assign resource pool
    * Assign vApp
    * Clone
    * Power off
    * Power on
    * View OVF environment
    * vApp application configuration
    * vApp instance configuration
    * vApp managedBy configuration
    * vApp resource configuration
* Virtual machine
    * Change Configuration
        * Acquire disk lease
        * Add existing disk
        * Add new disk
        * Add or remove device
        * Advanced configuration
        * Change CPU count
        * Change Memory
        * Change Settings
        * Change Swapfile placement
        * Change resource
        * Configure Host USB device
        * Configure Raw device
        * Configure managedBy
        * Display connection settings
        * Extend virtual disk
        * Modify device settings
        * Query Fault Tolerance compatibility
        * Query unowned files
        * Reload from path
        * Remove disk
        * Rename
        * Reset guest information
        * Set annotation
        * Toggle disk change tracking
        * Toggle fork parent
        * Upgrade virtual machine compatibility
    * Edit Inventory
        * Create from existing
        * Create new
        * Move
        * Register
        * Remove
        * Unregister
    * Guest operations
        * Guest operation alias modification
        * Guest operation alias query
        * Guest operation modifications
        * Guest operation program execution
        * Guest operation queries
    * Interaction
        * Power off
        * Power on
        * Reset
    * Provisioning
        * Allow disk access
        * Allow file access
        * Allow read-only disk access
        * Allow virtual machine files upload
        * Clone template
        * Clone virtual machine
        * Customize guest
        * Deploy template
        * Mark as virtual machine
        * Modify customization specification
        * Promote disks
        * Read customization specifications

#### NSX-T

The NSX-T API is accessed from the infrastructure controller of the vsphere-provider for setting up the network infrastructure resources and the cloud-controller-manager for managing load balancers. Currently, the NSX-T user must have the `Enterprise Admin` role.


### Create Folders

Two folders need to be created:
    - a folder which will contain the VMs of the shoots (cloud profile `spec.providerConfig.folder`)
    - a folder containing templates (used by cloud profile `spec.providerConfig.machineImages[*].versions[*].path`)

In vSphere client:

1. From the *Menu* in the vSphere Client toolbar choose *VMs and Templates*
2. Select the vSphere Datacenter of the work load vCenter in the browser
3. From the context menu select *New Folder* \> *New VM and Template Folder*, set folder name to e.g. "gardener"
4. From the context menu of the new folder *gardener* select *New Folder*, set folder name to "templates"

### Upload VM Templates for Worker Nodes

Upload [gardenlinux OVA](https://github.com/gardenlinux/gardenlinux/releases) or 
[flatcar OVA](https://stablereleaseflatcar-linuxnet/amd64-usr/current/flatcar_production_vmware_ova.ova) templates.

1. From the context menu of the folder `gardener/templates` choose *Deploy OVF Template...*
2. Adjust name if needed
3. Select any compute cluster as compute resource
4. Select a storage (e.g. VSAN)
5. Select any network (not important)
6. No need to customize the template
7. After deployment is finished select from the context menu of the new deployed VM *Template* \> *Convert To Template*

### Prepare for Kubernetes Zones and Regions

If the vSphere infrastructure is setup for multiple availabilities zones and Kubernetes should be topology aware, there need to be defined two labels in the cloud profile (section `spec.providerConfig.failureDomainLabels`)

```yaml
    failureDomainLabels:
      region: k8s-region
      zone: k8s-zone
```

See also: [deploying_csi_with_zones](https://vsphere-csi-driver.sigs.k8s.io/driver-deployment/deploying_csi_with_zones.html)

A Kubernetes zone can either be a vCenter or one of its datacenters

Zones must be subresources of it. If the region is a complete vCenter, the zone must specify datacenter and either compute cluster or resource pool.
Otherwise, i.e. tf the region is a datacenter, the zone must specify either compute cluster or resource pool.

In the following steps it is assumed:
    - the region is specified by a datacenter
    - the zone is specified by a compute clusters or one of its resource pools

#### Create Resource Pool(s)

Create a resource pool for every zone:

1. From the *Menu* in the vSphere Client toolbar choose *Hosts and Clusters*
2. From the context menu of the compute cluster select *New Resource Pool...* and provide the name of the zone. CPU and Memory settings are optional.

#### Tag Regions and Zones

Eeach zone must be tagged with the category defined by the label defined in the cloud profile (`spec.providerConfig.failureDomainLabels.region`).
Assuming that the region is a datacenter and the region label is `k8s-region`:

1. From the *Menu* in the vSphere Client toolbar choose *Hosts and Clusters*
2. Select the region's datacenter in the browser
3. In the *Summary* tab there is a subwindow titled *Tags*. Click the *Assign...* link.
4. In the *Assign Tag* dialog select the *ADD TAG* link above of the table
5. In the *Create Tag* dialog choose the *k8s-region* category. If it is not defined, click the *Create New Category* link to create the category.
6. Enter the *Name* of the region.
7. Back in the *Assign Tag* mark the checkbox of the region tag you just have created.
8. Click the *ASSIGN* button

Assuiming that the zone are specified by resource pools and the zone label is `k8s-zone`:

1. From the *Menu* in the vSphere Client toolbar choose *Hosts and Clusters*
2. Select the zone's compute cluster in the browser
3. In the *Summary* tab there is a subwindow titled *Tags*. Click the *Assign...* link.
4. In the *Assign Tag* dialog select the *ADD TAG* link above of the table
5. In the *Create Tag* dialog choose the *k8s-zone* category. If it is not defined, click the *Create New Category* link to create the category.
6. Enter the *Name* of the zone.
7. Back in the *Assign Tag* mark the checkbox of the zone tag you just have created.
8. Click the *ASSIGN* button

#### Storage policies

Each zone can have a separate storage. In this case a storage policy is needed to be compatible with all the zone storages.

##### Tag Zone Storages

For each zone tag the storage with the corresponding `k8s-zone` tag for the zone.

1. From the *Menu* in the vSphere Client toolbar choose *Storage*
2. Select the zone's storage in the browser
3. In the *Summary* tab there is a subwindow titled *Tags*. Click the *Assign...* link.
4. In the *Assign Tag* dialog select the *ADD TAG* link above of the table
5. In the *Create Tag* dialog choose the *k8s-zone* category. If it is not defined, click the *Create New Category* link to create the category.
6. Enter the *Name* of the zone.
7. Back in the *Assign Tag* mark the checkbox of the zone tag you just have created.
8. Click the *ASSIGN* button

##### Create or clone VM Storage Policy

1. From the *Menu* in the vSphere Client toolbar choose *Policies and Profiles*
2. In the *Policiies and Profiles* list select *VM Storage Policies*
3. Create or clone an exisitng storage policy

    a) set name, e.g. "&lt;region-name> Storage Policy" (will be needed for the cloud profile later)

    b) On the page *Policy structure* check only the checkbox *Enable tag based placement rules*

    c) On the page *Tage based placement* press the *ADD TAG RULE* button.

    d) For *Rule 1* select

       *Tag category* =  *k8s-zone*
       *Usage option* = *Use storage tagged with*
       *Tags* = *all zone tags*.

    e) Validate the compatible storages on the page *Storage compatibility*

    f) Press *FINISH* on the *Review and finish* page

## NSX-T Prepartion

A shared NSX-T is needed for all zones of a region.
External IP address ranges are needed for SNAT and load balancers.
Besides the edge cluster must sized large enough to deal with the load balancers of all shoots.

### Create IP pools

Two IP pools are needed for external IP addresses.

1. IP pool for **SNAT**
   The IP pool name needs to be specified in the cloud profile at `spec.providerConfig.regions[*].snatIPPool`. Each shoot cluster needs one SNAT IP address for outgoing traffic.
2. IP pool(s) for the **load balancers**
   The IP pool name(s) need to be specified in the cloud profile at `spec.providerConfig.contraints.loadBalancerConfig.classes[*].ipPoolName`. An IP address is needed for every port of every Kubernetes service of type `LoadBalancer`.

To create them, follow these steps in the NSX-T Manager UI in the web browser:

1. From the *toolbar* at the top of the page choose *Networking*
2. From the left side list choose *IP Address Pools* below the *IP Management*
3. Press the *ADD IP ADRESS POOL* button
4. Enter *Name*
5. Enter at least one subnet by clicking on *Sets*
6. Press the *Save* button

#### Sizing the IP pools

Each shoot cluster needs one IP address for SNAT and at least two IP addresses for load balancers VIPs (kube-apiservcer and Gardener shoot-seed VPN). A third IP address may be needed for ingress.
Depending on the payload of a shoot cluster, there may be additional services of type `LoadBalancer`. An IP address is needed for every port of every Kubernetes service of type `LoadBalancer`.

### Check edge cluster sizing

For load balancer related configurations limitations of NSX-T, please see the web pages [VMware Configuration Maximums](https://configmax.vmware.com/guest?vmwareproduct=NSX-T%20Data%20Center&release=NSX-T%20Data%20Center%203.1.0&categories=20-0). The link shows the limitations for NSX-T 3.1, if you have another version, please select the version from the left panel under *Select Version* and press the *VIEW LIMITS* button to update the view.

By default settings, each shoot cluster has an own T1 gateway and an own LB service (instance) of "T-shirt" size `SMALL`.

Examples for limitations on NSX-T 3.1 using *Large Edge Node* and *SMALL* load balancers instances:

1. There is a limit of 40 small LB instances per egde cluster (for HA 40 per pair of edge nodes)

    => maximum number of shoot clusters = 40 * (number of edge nodes) / 2

2. For `SMALL` load balancers, there is a maximum of 20 virtual servers. A virtual server is needed for every port of a service of type `LoadBalancer`

   => maximum number of services/ports pairs = 20 * (number of edge nodes) / 2

   The load balancer "T-shirt" size can be set on cloud profile level (`spec.providerConfig.contraints.loadBalancerConfig.size`) or in the shoot manifest (`spec.provider.controlPlaneConfig.loadBalancerSize`)

3. The number of pool members is limited to 7,500. For every K8s service port, every worker node is a pool member.

   => If every shoot cluster has an average number of 15 worker nodes, there can be 500 service/port pairs over all shoot clusters per pair of edge nodes

## Get VDS UUIDs

This step is only needed, if there are several VDS (virtual distributed switches) for each zone.

In this case, their UUIDs need to be fetched and set in the cloud profile at `spec.providerConfig.regions[*].zones[*].switchUuid`.

Unfortunately, they are not displayed in the vSphere Client.

Here the command line tool `govc` is used to look them
up.

1. Run `govc find / -type DistributedVirtualSwitch` to get the full path of all vds/dvs
2. For each switch run `govc dvs.portgroup.info <switch-path> | grep DvsUuid`
