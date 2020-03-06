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
}

// VsphereConfig holds information about vSphere resources to use.
type VsphereConfig struct {
	// Folder is the folder name to store the cloned machine VM
	Folder string `json:"folder,omitempty"`
	// Region is the vSphere region
	Region string `json:"region"`
	// ZoneConfig holds information about zone
	ZoneConfigs map[string]ZoneConfig `json:"zoneConfigs"`
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
	EdgeClusterRef        *Reference        `json:"edgeClusterRef,omitempty"`
	TransportZoneRef      *Reference        `json:"transportZoneRef,omitempty"`
	Tier0GatewayRef       *Reference        `json:"tier0GatewayRef,omitempty"`
	SNATIPPoolRef         *Reference        `json:"snatIPPoolRef,omitempty"`
	Tier1GatewayRef       *Reference        `json:"tier1GatewayRef,omitempty"`
	LocaleServiceRef      *Reference        `json:"localeServiceRef,omitempty"`
	SegmentRef            *Reference        `json:"segmentRef,omitempty"`
	SNATIPAddressAllocRef *Reference        `json:"snatIPAddressAllocRef,omitempty"`
	SNATRuleRef           *Reference        `json:"snatRuleRef,omitempty"`
	SNATIPAddress         *string           `json:"snatIPAddress,omitempty"`
	SegmentName           *string           `json:"segmentName,omitempty"`
	AdvancedDHCP          AdvancedDHCPState `json:"advancedDHCP"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`

	VsphereConfig VsphereConfig `json:"vsphereConfig"`

	NSXTInfraState *NSXTInfraState `json:"nsxtInfraState,omitempty"`
}
