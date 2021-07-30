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
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// VirtualNetworkSpec is the partial spec of VirtualNetwork.vmware.com/v1alpha1
type VirtualNetworkSpec struct {
	Private     bool
	EnableDHCP  bool
	AddressCIDR string
}

// VirtualNetworkStatus is a partial status of VirtualNetwork.vmware.com/v1alpha1
type VirtualNetworkStatus struct {
	Ready         bool
	DefaultSNATIP *string
}

type internalVirtualNetworkStatus struct {
	Conditions    []v1.ComponentCondition `json:"conditions,omitempty"`
	DefaultSNATIP *string                 `json:"defaultSNATIP,omitempty"`
}

// CreateVirtualNetwork creates a VirtualNetwork for a shoot cluster
func CreateVirtualNetwork(ctx context.Context, client ctrlClient.Client, name ctrlClient.ObjectKey, spec VirtualNetworkSpec) error {
	obj := newVirtualNetworkObj(name)
	obj.Object["spec"] = map[string]interface{}{
		"private":     spec.Private,
		"enableDHCP":  spec.EnableDHCP,
		"addressCIDR": spec.AddressCIDR,
	}

	return client.Create(ctx, obj)
}

// GetVirtualNetwork gets partial status of a VirtualNetwork
func GetVirtualNetworkStatus(ctx context.Context, client ctrlClient.Client, name ctrlClient.ObjectKey) (*VirtualNetworkStatus, error) {
	obj := newVirtualNetworkObj(name)
	err := client.Get(ctx, name, obj)
	if err != nil {
		return nil, err
	}

	vnstatus := internalVirtualNetworkStatus{}
	if status := obj.Object["status"]; status != nil {
		bytes, err := json.Marshal(status)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bytes, &vnstatus)
		if err != nil {
			return nil, err
		}
	}

	ready := false
	var snat *string
	for _, cond := range vnstatus.Conditions {
		if cond.Type == "Ready" {
			ready = cond.Status == v1.ConditionTrue
		}
	}
	snat = vnstatus.DefaultSNATIP
	return &VirtualNetworkStatus{Ready: ready, DefaultSNATIP: snat}, nil
}

// DeleteVirtualNetwork deletes a VirtualNetwork
func DeleteVirtualNetwork(ctx context.Context, client ctrlClient.Client, name types.NamespacedName) error {
	obj := newVirtualNetworkObj(name)
	return client.Delete(ctx, obj)
}

func newVirtualNetworkObj(name types.NamespacedName) *unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	obj.SetKind("VirtualNetwork")
	obj.SetAPIVersion("vmware.com/v1alpha1")
	obj.SetNamespace(name.Namespace)
	obj.SetName(name.Name)
	return &obj
}
