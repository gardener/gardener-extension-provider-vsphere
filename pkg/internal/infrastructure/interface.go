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
	EdgeClusterRef        *Reference        `json:"edgeClusterRef,omitEmpty"`
	TransportZoneRef      *Reference        `json:"transportZoneRef,omitEmpty"`
	Tier0GatewayRef       *Reference        `json:"tier0GatewayRef,omitEmpty"`
	SNATIPPoolRef         *Reference        `json:"snatIPPoolRef,omitEmpty"`
	Tier1GatewayRef       *Reference        `json:"tier1GatewayRef,omitEmpty"`
	LocaleServiceRef      *Reference        `json:"localeServiceRef,omitEmpty"`
	SegmentRef            *Reference        `json:"segmentRef,omitEmpty"`
	SNATIPAddressAllocRef *Reference        `json:"snatIPAddressAllocRef,omitEmpty"`
	SNATRuleRef           *Reference        `json:"snatRuleRef,omitEmpty"`
	SNATIPAddress         *string           `json:"snatIPAddress,omitEmpty"`
	AdvancedDHCP          AdvancedDHCPState `json:"advancedDHCP"`
}

type AdvancedDHCPState struct {
	EdgeClusterID   *string `json:"edgeClusterID,omitEmpty"`
	LogicalSwitchID *string `json:"logicalSwitchID,omitEmpty"`
	ProfileID       *string `json:"profileID,omitEmpty"`
	ServerID        *string `json:"serverID,omitEmpty"`
	PortID          *string `json:"portID,omitEmpty"`
	IPPoolID        *string `json:"ipPoolID,omitEmpty"`
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
