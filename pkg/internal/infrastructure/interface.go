/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package infrastructure

type NSXTInfraSpec struct {
	EdgeClusterName   string
	TransportZoneName string
	Tier0GatewayName  string
	SNATIPPoolName    string
	GardenName        string
	ClusterName       string
	WorkersNetwork    string
	DNSServers        []string
}

type Reference struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

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

type AdvancedDHCPState struct {
	LogicalSwitchID *string `json:"logicalSwitchID,omitempty"`
	ProfileID       *string `json:"profileID,omitempty"`
	ServerID        *string `json:"serverID,omitempty"`
	PortID          *string `json:"portID,omitempty"`
	IPPoolID        *string `json:"ipPoolID,omitempty"`
}

// NsxtConfig contains the NSX-T specific configuration
type NsxtConfig struct {
	// NSX-T username.
	User string `json:"user"`
	// NSX-T password in clear text.
	Password string `json:"password"`
	// NSX-T host.
	Host string `json:"host"`
	// InsecureFlag is to be set to true if NSX-T uses self-signed cert.
	InsecureFlag bool `json:"insecure-flag"`

	VMCAccessToken     string `json:"vmcAccessToken"`
	VMCAuthHost        string `json:"vmcAuthHost"`
	ClientAuthCertFile string `json:"client-auth-cert-file"`
	ClientAuthKeyFile  string `json:"client-auth-key-file"`
	CAFile             string `json:"ca-file"`
}

type NSXTInfrastructureEnsurer interface {
	EnsureInfrastructure(spec NSXTInfraSpec, state *NSXTInfraState) error
	EnsureInfrastructureDeleted(spec NSXTInfraSpec, state *NSXTInfraState) error
}
