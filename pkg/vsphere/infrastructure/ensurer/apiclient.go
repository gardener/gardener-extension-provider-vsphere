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
	nsxt "github.com/vmware/go-vmware-nsxt"

	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

func createNSXClient(nsxtConfig *vinfra.NSXTConfig) (*nsxt.APIClient, error) {
	retriesConfig := nsxt.ClientRetriesConfiguration{
		MaxRetries:    30,
		RetryMinDelay: 500,
		RetryMaxDelay: 5000,
	}
	cfg := nsxt.Configuration{
		BasePath:             "/api/v1",
		Host:                 nsxtConfig.Host,
		Scheme:               "https",
		UserAgent:            "gardener-extension-provider-vsphere",
		UserName:             nsxtConfig.User,
		Password:             nsxtConfig.Password,
		ClientAuthCertFile:   nsxtConfig.ClientAuthCertFile,
		ClientAuthKeyFile:    nsxtConfig.ClientAuthKeyFile,
		CAFile:               nsxtConfig.CAFile,
		Insecure:             nsxtConfig.InsecureFlag,
		RemoteAuth:           nsxtConfig.RemoteAuth,
		RetriesConfiguration: retriesConfig,
	}
	client, err := nsxt.NewAPIClient(&cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}
