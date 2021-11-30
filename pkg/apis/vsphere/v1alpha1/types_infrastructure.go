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

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Networks contains optional existing network infrastructure to use.
	// If not defined, NSX-T Tier-1 gateway and load balancer are created for the shoot cluster.
	// unused if VsphereWithKubernetes is set
	// +optional
	Networks *Networks `json:"networks,omitempty"`
	// OverwriteNSXTInfraVersion allows to fix the ensurer version used to create the NSXT-T infrastructure.
	// This is an advanced configuration to overwrite the automatic version selection.
	// unused if VsphereWithKubernetes is set
	// +optional
	OverwriteNSXTInfraVersion *string `json:"overwriteNSXTInfraVersion,omitempty"`
}

// Networks contains existing NSX-T network infrastructure to use.
type Networks struct {
	// Tier1GatewayPath is the path of the existing NSX-T Tier-1 Gateway to use.
	Tier1GatewayPath string `json:"tier1GatewayPath"`
	// LoadBalancerServicePath is the path of the existing NSX-T load balancer service assigned to the Tier-1 Gateway
	LoadBalancerServicePath string `json:"loadBalancerServicePath"`
}

// VsphereConfig holds information about vSphere resources to use.
type VsphereConfig struct {
	// Folder is the folder name to store the cloned machine VM
	// not filled if VsphereWithKubernetes is set
	Folder string `json:"folder,omitempty"`
	// Region is the vSphere region
	Region string `json:"region"`
	// ZoneConfig holds information about zone
	// not filled if VsphereWithKubernetes is set
	ZoneConfigs map[string]ZoneConfig `json:"zoneConfigs"`
	// Namespace is the vSphere Kubernetes namespace
	// only filled if VsphereWithKubernetes is set
	Namespace string `json:"namespace,omitempty"`
}

// ZoneConfig holds zone specific information about vSphere resources to use.
type ZoneConfig struct {
	// Datacenter is the name of the data center
	Datacenter string `json:"datacenter"`
	// ComputeCluster is the name of the compute cluster. Either ComputeCluster or ResourcePool or HostSystem must be specified
	ComputeCluster string `json:"computeCluster,omitempty"`
	// ResourcePool is the name of the resource pool. Either ComputeCluster or ResourcePool or HostSystem must be specified
	ResourcePool string `json:"resourcePool,omitempty"`
	// HostSystem is the name of the host system. Either ComputeCluster or ResourcePool or HostSystem must be specified
	HostSystem string `json:"hostSystem,omitempty"`
	// Datastore is the datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified
	Datastore string `json:"datastore,omitempty"`
	// DatastoreCluster is the datastore  cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified
	DatastoreCluster string `json:"datastoreCluster,omitempty"`
	// SwitchUUID is the UUID of the virtual distributed switch the network is assigned to (only needed if there are multiple vds)
	SwitchUUID string `json:"switchUuid,omitempty"`
}

// Reference holds a NSXT object reference managed with the NSX-T simplified / intent-based API
type Reference struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// AdvancedDHCPState holds IDs of objects managed with the NSX-T Advanced API
type AdvancedDHCPState struct {
	LogicalSwitchID *string `json:"logicalSwitchID,omitempty"`
	ProfileID       *string `json:"profileID,omitempty"`
	ServerID        *string `json:"serverID,omitempty"`
	PortID          *string `json:"portID,omitempty"`
	IPPoolID        *string `json:"ipPoolID,omitempty"`
}

// NSXTInfraState holds the state of the infrastructure created with NSX-T
type NSXTInfraState struct {
	Version               *string           `json:"version,omitempty"`
	EdgeClusterRef        *Reference        `json:"edgeClusterRef,omitempty"`
	TransportZoneRef      *Reference        `json:"transportZoneRef,omitempty"`
	Tier0GatewayRef       *Reference        `json:"tier0GatewayRef,omitempty"`
	SNATIPPoolRef         *Reference        `json:"snatIPPoolRef,omitempty"`
	Tier1GatewayRef       *Reference        `json:"tier1GatewayRef,omitempty"`
	ExternalTier1Gateway  *bool             `json:"externalTier1Gateway,omitempty"`
	LocaleServiceRef      *Reference        `json:"localeServiceRef,omitempty"`
	SegmentRef            *Reference        `json:"segmentRef,omitempty"`
	SNATIPAddressAllocRef *Reference        `json:"snatIPAddressAllocRef,omitempty"`
	SNATRuleRef           *Reference        `json:"snatRuleRef,omitempty"`
	SNATIPAddress         *string           `json:"snatIPAddress,omitempty"`
	SegmentName           *string           `json:"segmentName,omitempty"`
	DHCPServerConfigRef   *Reference        `json:"dhcpServerConfigRef,omitempty"`
	AdvancedDHCP          AdvancedDHCPState `json:"advancedDHCP"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`

	// not filled if VsphereWithKubernetes is set
	// +optional
	VsphereConfig *VsphereConfig `json:"vsphereConfig,omitempty"`
	// not filled if VsphereWithKubernetes is set
	// +optional
	CreationStarted *bool `json:"creationStarted,omitempty"`
	// not filled if VsphereWithKubernetes is set
	// +optional
	NSXTInfraState *NSXTInfraState `json:"nsxtInfraState,omitempty"`

	// VirtualNetwork is the name of the network segment in the vSphere Kubernetes namespace
	// only filled if VsphereWithKubernetes is set
	// +optional
	VirtualNetwork *string `json:"virtualNetwork,omitempty"`
	// NCPRouterID is the identifier of the Tier1 gateway (router) of the vSphere Kubernetes namespace
	// only filled if VsphereWithKubernetes is set
	// +optional
	NCPRouterID *string `json:"ncpRouterID,omitempty"`
}
