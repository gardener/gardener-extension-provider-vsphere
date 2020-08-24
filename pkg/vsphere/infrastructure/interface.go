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

import api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"

type NSXTInfraSpec struct {
	EdgeClusterName   string   `json:"edgeClusterName"`
	TransportZoneName string   `json:"transportZoneName"`
	Tier0GatewayName  string   `json:"tier0GatewayName"`
	SNATIPPoolName    string   `json:"snatIPPoolName"`
	GardenID          string   `json:"gardenID"`
	GardenName        string   `json:"gardenName"`
	ClusterName       string   `json:"clusterName"`
	WorkersNetwork    string   `json:"workersNetwork"`
	DNSServers        []string `json:"dnsServers"`

	ExternalTier1GatewayPath *string `json:"externalTier1GatewayPath,omitempty"`
}

// NSXTConfig contains the NSX-T specific configuration
type NSXTConfig struct {
	// NSX-T username.
	User string `json:"user,omitempty"`
	// NSX-T password in clear text.
	Password string `json:"password,omitempty"`
	// NSX-T host.
	Host string `json:"host,omitempty"`
	// InsecureFlag is to be set to true if NSX-T uses self-signed cert.
	InsecureFlag bool `json:"insecure-flag,omitempty"`
	// RemoteAuth is to be set to true if NSX-T uses remote authentication (authentication done through the vIDM).
	RemoteAuth bool `json:"remote-auth,omitempty"`

	VMCAccessToken     string `json:"vmcAccessToken,omitempty"`
	VMCAuthHost        string `json:"vmcAuthHost,omitempty"`
	ClientAuthCertFile string `json:"client-auth-cert-file,omitempty"`
	ClientAuthKeyFile  string `json:"client-auth-key-file,omitempty"`
	CAFile             string `json:"ca-file,omitempty"`
}

// NSXTInfrastructureEnsurer ensures that infrastructure is completed created or deleted
type NSXTInfrastructureEnsurer interface {
	// CheckConnection checks if the NSX-T REST API is reachable with the given endpoint and credentials.
	CheckConnection() error
	// NewStateWithVersion creates empty state with version depending on NSX-T backend or overwritten version
	NewStateWithVersion(overwriteVersion *string) (*api.NSXTInfraState, error)
	// EnsureInfrastructure ensures that the infrastructure is complete
	// It checks all infrastructure objects and creates missing one or updates them if important attributes have been changed.
	// It can even recover objects not recorded in the state.
	EnsureInfrastructure(spec NSXTInfraSpec, state *api.NSXTInfraState) error
	// EnsureInfrastructureDeleted ensures that all infrastructure objects are deleted
	// It even trys to recover objects not recorded in the state before deleting them.
	EnsureInfrastructureDeleted(spec *NSXTInfraSpec, state *api.NSXTInfraState) error
	// GetIPPoolTags retrieves the tags of an IP pool
	GetIPPoolTags(ipPoolName string) (map[string]string, error)
	// CheckShootAuthorizationByTags checks if shoot namespace is allowed in the tags
	CheckShootAuthorizationByTags(objectType, name string, tags map[string]string) error
}
