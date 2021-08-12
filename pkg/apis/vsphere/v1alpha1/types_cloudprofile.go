// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta `json:",inline"`

	// VsphereWithKubernetes if true, infrastructure and VMs are created on vSphere Kubernetes workloads (supervisor cluster)
	// +optional
	VsphereWithKubernetes *VsphereWithKubernetes `json:"vsphereWithKubernetes,omitempty"`

	// NamePrefix is used for naming NSX-T resources
	// unused if VsphereWithKubernetes is set
	NamePrefix string `json:"namePrefix,omitempty"`
	// Folder is the vSphere folder name to store the cloned machine VM (worker nodes)
	// unused if VsphereWithKubernetes is set
	Folder string `json:"folder,omitempty"`
	// Regions is the specification of regions and zones topology
	// unused if VsphereWithKubernetes is set
	Regions []RegionSpec `json:"regions,omitempty"`
	// DefaultClassStoragePolicyName is the name of the vSphere storage policy to use for the 'default-class' storage class
	DefaultClassStoragePolicyName string `json:"defaultClassStoragePolicyName"`
	// FailureDomainLabels are the tag categories used for regions and zones.
	// unused if VsphereWithKubernetes is set
	// +optional
	FailureDomainLabels *FailureDomainLabels `json:"failureDomainLabels,omitempty"`
	// DNSServers is a list of IPs of DNS servers used while creating subnets.
	// unused if VsphereWithKubernetes is set
	DNSServers []string `json:"dnsServers,omitempty"`
	// DHCPOptions contains optional options for DHCP like Domain name, NTP server,...
	// unused if VsphereWithKubernetes is set
	// +optional
	DHCPOptions []DHCPOption `json:"dhcpOptions,omitempty"`

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to provider-specific identifiers.
	MachineImages []MachineImages `json:"machineImages,omitempty"`
	// Constraints is an object containing constraints for certain values in the control plane config.
	Constraints Constraints `json:"constraints"`
	// CSIResizerDisabled is a flag to disable the CSI resizer (e.g. resizer is not supported for vSphere 6.7)
	// +optional
	CSIResizerDisabled *bool `json:"csiResizerDisabled,omitempty"`
	// MachineTypeOptions is the list of machine type options to set additional options for individual machine types.
	// unused if VsphereWithKubernetes is set
	// +optional
	MachineTypeOptions []MachineTypeOptions `json:"machineTypeOptions,omitempty"`
	// DockerDaemonOptions contains configuration options for docker daemon service
	// +optional
	DockerDaemonOptions *DockerDaemonOptions `json:"dockerDaemonOptions,omitempty"`
}

// FailureDomainLabels are the tag categories used for regions and zones in vSphere CSI driver and cloud controller.
// See Cloud Native Storage: Set Up Zones in the vSphere CNS Environment
// (https://docs.vmware.com/en/VMware-vSphere/6.7/Cloud-Native-Storage/GUID-9BD8CD12-CB24-4DF4-B4F0-A862D0C82C3B.html)
type FailureDomainLabels struct {
	// Region is the tag category used for region on vSphere data centers and/or clusters.
	Region string `json:"region"`
	// Zone is the tag category used for zones on vSphere data centers and/or clusters.
	Zone string `json:"zone"`
}

// DHCPOption contains a DHCP option by code
type DHCPOption struct {
	// Code is the tag according to the BOOTP Vendor Extensions and DHCP Options (see https://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml)
	// most important codes: 'Domain Name'=15 (only allowed for NSX-T 2.5, use code 119 for NSX-T >= 3.0), 'NTP server'=42, 'Domain Search': 119
	Code int `json:"code"`
	// Values are the values for the given code
	Values []string `json:"values"`
}

// RegionSpec specifies the topology of a region and its zones.
// A region consists of a Vcenter host, transport zone and optionally a data center.
// A zone in a region consists of a data center (if not specified in the region), a computer cluster,
// and optionally a resource zone or host system.
type RegionSpec struct {
	// Name is the name of the region
	Name string `json:"name"`
	// VsphereHost is the vSphere host
	VsphereHost string `json:"vsphereHost"`
	// VsphereInsecureSSL is a flag if insecure HTTPS is allowed for VsphereHost
	VsphereInsecureSSL bool `json:"vsphereInsecureSSL"`
	// NSXTHost is the NSX-T host
	NSXTHost string `json:"nsxtHost"`
	// NSXTInsecureSSL is a flag if insecure HTTPS is allowed for NSXTHost
	NSXTInsecureSSL bool `json:"nsxtInsecureSSL"`
	// NSXTRemoteAuth is a flag if NSX-T uses remote authentication (authentication done through the vIDM).
	NSXTRemoteAuth bool `json:"nsxtRemoteAuth"`
	// TransportZone is the NSX-T transport zone
	TransportZone string `json:"transportZone"`
	// LogicalTier0Router is the NSX-T logical tier 0 router
	LogicalTier0Router string `json:"logicalTier0Router"`
	// EdgeCluster is the NSX-T edge cluster
	EdgeCluster string `json:"edgeCluster"`
	// SNATIPPool is the NSX-T IP pool to allocate the SNAT ip address
	SNATIPPool string `json:"snatIPPool"`

	// Datacenter is the name of the vSphere data center (data center can either be defined at region or zone level)
	// +optional
	Datacenter *string `json:"datacenter,omitempty"`

	// Datastore is the vSphere datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.
	// +optional
	Datastore *string `json:"datastore,omitempty"`
	// DatastoreCluster is the vSphere  datastore cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.
	// +optional
	DatastoreCluster *string `json:"datastoreCluster,omitempty"`

	// Zones is the list of zone specifications of the region.
	Zones []ZoneSpec `json:"zones"`

	// CaFile is the optional CA file to be trusted when connecting to vCenter. If not set, the node's CA certificates will be used. Only relevant if InsecureFlag=0
	// +optional
	CaFile *string `json:"caFile,omitempty"`
	// Thumbprint is the optional vCenter certificate thumbprint, this ensures the correct certificate is used
	// +optional
	Thumbprint *string `json:"thumbprint,omitempty"`

	// DNSServers is a optional list of IPs of DNS servers used while creating subnets. If provided, it overwrites the global
	// DNSServers of the CloudProfileConfig
	// +optional
	DNSServers []string `json:"dnsServers,omitempty"`
	// DHCPOptions contains optional options for DHCP like Domain name, NTP server,...
	// If provided, it overwrites the global DHCPOptions of the CloudProfileConfig
	// +optional
	DHCPOptions []DHCPOption `json:"dhcpOptions,omitempty"`
	// MachineImages is the list of machine images that are understood by the controller. If provided, it overwrites the global
	// MachineImages of the CloudProfileConfig
	// +optional
	MachineImages []MachineImages `json:"machineImages,omitempty"`
}

// ZoneSpec specifies a zone of a region.
// A zone in a region consists of a data center (if not specified in the region), a computer cluster,
// and optionally a resource zone or host system.
type ZoneSpec struct {
	// Name is the name of the zone
	Name string `json:"name"`
	// Datacenter is the name of the vSphere data center (data center can either be defined at region or zone level)
	// +optional
	Datacenter *string `json:"datacenter,omitempty"`

	// ComputeCluster is the name of the vSphere compute cluster. Either ComputeCluster or ResourcePool or HostSystem must be specified
	// +optional
	ComputeCluster *string `json:"computeCluster,omitempty"`
	// ResourcePool is the name of the vSphere resource pool. Either ComputeCluster or ResourcePool or HostSystem must be specified
	// +optional
	ResourcePool *string `json:"resourcePool,omitempty"`
	// HostSystem is the name of the vSphere host system. Either ComputeCluster or ResourcePool or HostSystem must be specified
	// +optional
	HostSystem *string `json:"hostSystem,omitempty"`

	// Datastore is the vSphere datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.
	// +optional
	Datastore *string `json:"datastore,omitempty"`
	// DatastoreCluster is the vSphere  datastore cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.
	// +optional
	DatastoreCluster *string `json:"datastoreCluster,omitempty"`

	// SwitchUUID is the UUID of the virtual distributed switch the network is assigned to (only needed if there are multiple vds)
	// +optional
	SwitchUUID *string `json:"switchUuid,omitempty"`
}

// Constraints is an object containing constraints for the shoots.
type Constraints struct {
	// LoadBalancerConfig contains constraints regarding allowed values of the 'Lo' block in the control plane config.
	LoadBalancerConfig LoadBalancerConfig `json:"loadBalancerConfig"`
}

// MachineImages is a mapping from logical names and versions to provider-specific identifiers.
type MachineImages struct {
	// Name is the logical name of the machine image.
	Name string `json:"name"`
	// Versions contains versions and a provider-specific identifier.
	Versions []MachineImageVersion `json:"versions"`
}

// MachineImageVersion contains a version and a provider-specific identifier.
type MachineImageVersion struct {
	// Version is the version of the image.
	Version string `json:"version"`
	// Path is the path of the VM template.
	// if VsphereWithKubernetes is set, it contains the name of the `VirtualMachineImage.vmoperator.vmware.com` resource
	Path string `json:"path"`
	// GuestID is the optional guestId to overwrite the guestId of the VM template.
	// unused if VsphereWithKubernetes is set
	// +optional
	GuestID *string `json:"guestId,omitempty"`
}

// LoadBalancerConfig contains the constraints for usable load balancer classes
type LoadBalancerConfig struct {
	// Size is the NSX-T load balancer size ("SMALL", "MEDIUM", or "LARGE")
	Size string `json:"size"`
	// Classes are the defined load balancer classes
	Classes []LoadBalancerClass `json:"classes"`
}

// LoadBalancerClass defines a restricted network setting for generic LoadBalancer classes.
type LoadBalancerClass struct {
	// Name is the name of the LB class
	Name string `json:"name"`
	// IPPoolName is the name of the NSX-T IP pool (must be set for the default load balancer class).
	// +optional
	IPPoolName *string `json:"ipPoolName"`
	// TCPAppProfileName is the profile name of the load balaner profile for TCP
	// +optional
	TCPAppProfileName *string `json:"tcpAppProfileName,omitempty"`
	// UDPAppProfileName is the profile name of the load balaner profile for UDP
	// +optional
	UDPAppProfileName *string `json:"udpAppProfileName,omitempty"`
}

// MachineTypeOptions defines additional VM options for an machine type given by name
type MachineTypeOptions struct {
	// Name is the name of the machine type
	Name string `json:"name"`
	// MemoryReservationLockedToMax is flag to reserve all guest OS memory (no swapping in ESXi host)
	// +optional
	MemoryReservationLockedToMax *bool `json:"memoryReservationLockedToMax,omitempty"`
	// ExtraConfig allows to specify additional VM options.
	// e.g. sched.swap.vmxSwapEnabled=false to disable the VMX process swap file
	// +optional
	ExtraConfig map[string]string `json:"extraConfig,omitempty"`
}

// DockerDaemonOptions contains configuration options for Docker daemon service
type DockerDaemonOptions struct {
	// HTTPProxyConf contains HTTP/HTTPS proxy configuration for Docker daemon
	// +optional
	HTTPProxyConf *string `json:"httpProxyConf,omitempty"`
	// InsecureRegistries adds the given registries to Docker on the worker nodes
	// (see https://docs.docker.com/registry/insecure/)
	// +optional
	InsecureRegistries []string `json:"insecureRegistries,omitempty"`
}

// VsphereWithKubernetes contains settings for using "VSphere with Kubernetes" (experimental)
type VsphereWithKubernetes struct {
	// Namespace optionally specifies the namespace on the vSphere supervisor cluster (and implicitly the T1 Gateway)
	// If two shoot clusters use the same namespace, they can see the node network segments of each other.
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// StoragePolicies are the identifier of the storage policy assigned to a namespace (at least one is needed)
	StoragePolicies []string `json:"storagePolicies,omitempty"`

	// ContentLibraries are the content libraries identifiers to use to find OS images (at least one is needed)
	ContentLibraries []string `json:"contentLibraries,omitempty"`

	// VirtualMachineClasses are the names of `virtualmachineclass.vmoperator.vmware.com` allowed (at least one is needed)
	VirtualMachineClasses []string `json:"virtualMachineClasses,omitempty"`

	// Regions is the specification of regions and zones topology
	Regions []K8sRegionSpec `json:"regions"`
}

// K8sRegionSpec is the VsphereWithKubernetes specific region spec
type K8sRegionSpec struct {
	// Name is the name of the region
	Name string `json:"name"`

	// Cluster is the vSphere cluster id
	Cluster string `json:"cluster"`

	// VsphereHost is the vSphere host
	VsphereHost string `json:"vsphereHost"`
	// VsphereInsecureSSL is a flag if insecure HTTPS is allowed for VsphereHost
	VsphereInsecureSSL bool `json:"vsphereInsecureSSL"`

	// Zones is the list of zone specifications of the region.
	Zones []K8sZoneSpec `json:"zones"`
}

// K8sZoneSpec specifies a zone of a K8s region.
// currently only a placeholder
type K8sZoneSpec struct {
	// Name is the name of the zone
	Name string `json:"name"`

	// VMStorageClassName is the name of the storage class object used for VMs
	VMStorageClassName string `json:"vmStorageClassName"`
}
