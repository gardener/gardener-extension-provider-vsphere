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

package ensurer

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx/node"

	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

// GetNSXTVersion creates connection and retrieves the NSX-T version
func GetNSXTVersion(nsxtConfig *vinfra.NSXTConfig) (*string, error) {
	connector, err := createConnectorNiceError(nsxtConfig)
	if err != nil {
		return nil, err
	}
	return getNSXTVersion(connector)
}

func getNSXTVersion(connector client.Connector) (*string, error) {
	client := node.NewVersionClient(connector)
	nodeVersion, err := client.Get()
	if err != nil {
		return nil, err
	}
	return nodeVersion.ProductVersion, nil
}
