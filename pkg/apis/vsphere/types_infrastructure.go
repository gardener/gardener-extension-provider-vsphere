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

package vsphere

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Ensurer_Version1_NSXT25 = "1"
	Ensurer_Version2_NSXT30 = "2"
)

var SupportedEnsurerVersions = []string{Ensurer_Version1_NSXT25, Ensurer_Version2_NSXT30}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta
	// OverwriteNSXTInfraVersion allows to fix the ensurer version used to create the NSXT-T infrastructure.
	// This is an advanced configuration to overwrite the automatic version selection.
	OverwriteNSXTInfraVersion *string
}

// VsphereConfig holds information about vSphere resources to use.
type VsphereConfig struct {
	// Folder is the folder name to store the cloned machine VM
	Folder string
	// Region is the vSphere region
	Region string
	// ZoneConfig holds information about zone
	ZoneConfigs map[string]ZoneConfig
}

// ZoneConfig holds zone specific information about vSphere resources to use.
type ZoneConfig struct {
	// Datacenter is the name of the data center
	Datacenter string
	// ComputeCluster is the name of the compute cluster. Either ComputeCluster or ResourcePool or HostSystem must be specified
	ComputeCluster string
	// ResourcePool is the name of the resource pool. Either ComputeCluster or ResourcePool or HostSystem must be specified
	ResourcePool string
	// HostSystem is the name of the host system. Either ComputeCluster or ResourcePool or HostSystem must be specified
	HostSystem string
	// Datastore is the datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified
	Datastore string
	// DatastoreCluster is the datastore cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified
	DatastoreCluster string
}

// Reference holds a NSXT object reference managed with the NSX-T simplified / intent-based API
type Reference struct {
	ID   string
	Path string
}

// AdvancedDHCPState holds IDs of objects managed with the NSX-T Advanced API
type AdvancedDHCPState struct {
	LogicalSwitchID *string
	ProfileID       *string
	ServerID        *string
	PortID          *string
	IPPoolID        *string
}

// NSXTInfraState holds the state of the infrastructure created with NSX-T
type NSXTInfraState struct {
	Version               *string
	EdgeClusterRef        *Reference
	TransportZoneRef      *Reference
	Tier0GatewayRef       *Reference
	SNATIPPoolRef         *Reference
	Tier1GatewayRef       *Reference
	LocaleServiceRef      *Reference
	SegmentRef            *Reference
	SNATIPAddressAllocRef *Reference
	SNATRuleRef           *Reference
	SNATIPAddress         *string
	SegmentName           *string
	DHCPServerConfigRef   *Reference
	AdvancedDHCP          AdvancedDHCPState
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta

	VsphereConfig VsphereConfig

	CreationStarted *bool
	NSXTInfraState  *NSXTInfraState
}
