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
}

// NSXTConfig contains the NSX-T specific configuration
type NSXTConfig struct {
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
	EnsureInfrastructure(spec NSXTInfraSpec, state *api.NSXTInfraState) error
	EnsureInfrastructureDeleted(state *api.NSXTInfraState) error
}
