/*
 * Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubernetes

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type kubetestdata struct {
	Kubeconfig  []byte `json:"kubeconfig"`
	Namespace   string `json:"namespace"`
	AddressCIDR string `json:"addressCIDR"`
}

func TestCreateUpdateDeleteVirtualNetwork(t *testing.T) {
	filename := os.Getenv("PROVIDER_VSPHERE_VIRTUAL_NETWORK_TESTDATA_FILE")
	if filename == "" {
		t.Skipf("No path to testdata specified by environmental variable PROVIDER_VSPHERE_VIRTUAL_NETWORK_TESTDATA_FILE")
		return
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("reading testdata from %s failed with %s", filename, err)
		return
	}

	cfg := kubetestdata{}
	err = yaml.Unmarshal([]byte(content), &cfg)
	if err != nil {
		t.Errorf("Unmarshalling testdata failed with %s", err)
		return
	}

	client, err := CreateVsphereKubernetesClient(cfg.Kubeconfig)
	if err != nil {
		t.Errorf("CreateVsphereKubernetesClient failed with %s", err)
		return
	}

	ctx := context.TODO()
	name := ctrlClient.ObjectKey{
		Namespace: cfg.Namespace,
		Name:      "functest",
	}
	spec := VirtualNetworkSpec{
		Private:     true,
		EnableDHCP:  true,
		AddressCIDR: cfg.AddressCIDR,
	}
	err = CreateVirtualNetwork(ctx, client, name, spec)
	if err != nil {
		t.Errorf("CreateVirtualNetwork failed with %s", err)
		return
	}

	var status *VirtualNetworkStatus
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		status, err = GetVirtualNetworkStatus(ctx, client, name)
		if err != nil {
			t.Errorf("GetVirtualNetworkStatus failed with %s", err)
			return
		}
		if status.Ready {
			break
		}
	}
	if !status.Ready {
		t.Errorf("VirtualNetworkStatus ready expected")
		return
	}
	if status.DefaultSNATIP == nil {
		t.Errorf("VirtualNetworkStatus DefaultSNATIP == nil")
		return
	}

	err = DeleteVirtualNetwork(ctx, client, name)
	if err != nil {
		t.Errorf("DeleteVirtualNetwork failed with %s", err)
		return
	}
}
