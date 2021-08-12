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
	"crypto/tls"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/go-resty/resty/v2"
)

const apiSessionID = "vmware-api-session-id"

// GetVsphereAPISession gets a vsphere-api-session
func newVsphereAPIClient(region vsphere.K8sRegionSpec, username, password string) (*VsphereAPIClient, error) {
	url := fmt.Sprintf("https://%s/api/session", region.VsphereHost)

	client := newClient(region.VsphereInsecureSSL)
	resp, err := client.R().SetBasicAuth(username, password).Post(url)
	if err != nil {
		return nil, fmt.Errorf("POST %s for create session failed: %w", url, err)
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status())
	}

	session, err := strconv.Unquote(resp.String())
	if err != nil {
		return nil, fmt.Errorf("unquoting session %s failed: %w", resp.String(), err)
	}

	apiClient := &VsphereAPIClient{apiSession: session, region: region}
	return apiClient, nil
}

func newClient(insecure bool) *resty.Client {
	client := resty.New()
	if insecure {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	//client.SetDebug(true)
	client.SetRetryCount(3).SetRetryWaitTime(1 * time.Second)
	return client
}

type VsphereAPIClient struct {
	apiSession string
	region     vsphere.K8sRegionSpec
}

type namespaceBody struct {
	AccessList    []access       `json:"access_list,omitempty"`
	Cluster       string         `json:"cluster"`
	Creator       *creator       `json:"creator,omitempty"`
	Description   string         `json:"description,omitempty"`
	Namespace     string         `json:"namespace"`
	Networks      []string       `json:"networks,omitempty"`
	StorageSpecs  []storageSpec  `json:"storage_specs,omitempty"`
	VmServiceSpec *vmServiceSpec `json:"vm_service_spec,omitempty"`
}

type access struct {
	Domain      string `json:"domain"`
	Role        string `json:"role"`
	Subject     string `json:"subject"`
	SubjectType string `json:"subject_type"`
}

type creator struct {
	Domain  string `json:"domain"`
	Subject string `json:"subject"`
}

type storageSpec struct {
	Limit  int    `json:"limit,omitempty"`
	Policy string `json:"policy"`
}

type vmServiceSpec struct {
	ContentLibraries []string `json:"content_libraries,omitempty"`
	VmClasses        []string `json:"vm_classes,omitempty"`
}

func (apiclient *VsphereAPIClient) CreateNamespace(namespace string, vwk *vsphere.VsphereWithKubernetes) error {
	client := newClient(apiclient.region.VsphereInsecureSSL)

	url := fmt.Sprintf("https://%s/api/vcenter/namespaces/instances/%s", apiclient.region.VsphereHost, namespace)
	resp, err := client.R().
		SetHeader(apiSessionID, apiclient.apiSession).
		Get(url)
	if err != nil {
		return fmt.Errorf("GET request %s failed: %w", url, err)
	}
	switch resp.StatusCode() {
	case http.StatusNotFound:
		// expected
	case http.StatusOK:
		return NewAlreadyExistsError(fmt.Sprintf("namespace %s is already existing", namespace))
	default:
		return fmt.Errorf("GET request %s has unexpected status %s", url, resp.Status())
	}

	url = fmt.Sprintf("https://%s/api/vcenter/namespaces/instances", apiclient.region.VsphereHost)
	var storageSpecs []storageSpec
	for _, policy := range vwk.StoragePolicies {
		storageSpecs = append(storageSpecs, storageSpec{Policy: policy})
	}
	content := &namespaceBody{
		Namespace:    namespace,
		Cluster:      apiclient.region.Cluster,
		Description:  "created by gardener-extension-provider-vsphere",
		StorageSpecs: storageSpecs,
		VmServiceSpec: &vmServiceSpec{
			ContentLibraries: vwk.ContentLibraries,
			VmClasses:        vwk.VirtualMachineClasses,
		},
	}
	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader(apiSessionID, apiclient.apiSession).
		SetBody(content).
		Post(url)

	if err != nil {
		return fmt.Errorf("POST request %s failed: %w", url, err)
	}

	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("POST request %s has unexpected status %s", url, resp.Status())
	}
	return nil
}

func (apiclient *VsphereAPIClient) GetNamespaceCluster(namespace string) (string, error) {
	ns, err := apiclient.getNamespace(namespace)
	if err != nil {
		return "", err
	}
	return ns.Cluster, nil
}

func (apiclient *VsphereAPIClient) getNamespace(namespace string) (*namespaceBody, error) {
	client := newClient(apiclient.region.VsphereInsecureSSL)

	url := fmt.Sprintf("https://%s/api/vcenter/namespaces/instances/%s", apiclient.region.VsphereHost, namespace)
	resp, err := client.R().
		SetHeader(apiSessionID, apiclient.apiSession).
		SetResult(&namespaceBody{}).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET request %s failed: %w", url, err)
	}
	switch resp.StatusCode() {
	case http.StatusNotFound:
		return nil, NewNotFoundError(fmt.Sprintf("namespace %s is not existing", namespace))
	case http.StatusOK:
		// expected
	default:
		return nil, fmt.Errorf("GET request %s has unexpected status %s", url, resp.Status())
	}

	ns := resp.Result().(*namespaceBody)
	return ns, nil
}

func (apiclient *VsphereAPIClient) UpdateNamespace(namespace string, vwk *vsphere.VsphereWithKubernetes) (bool, error) {
	ns, err := apiclient.getNamespace(namespace)
	if err != nil {
		return false, err
	}

	if ns.Cluster != apiclient.region.Cluster {
		return false, fmt.Errorf("mismatching cluster %s != %s, cannot be updated", ns.Cluster, apiclient.region.Cluster)
	}

	modified := len(ns.StorageSpecs) != len(vwk.StoragePolicies)
	oldPolicies := map[string]struct{}{}
	for _, policy := range ns.StorageSpecs {
		oldPolicies[policy.Policy] = struct{}{}
		if policy.Limit != 0 {
			modified = true
		}
	}
	var storageSpecs []storageSpec
	for _, policy := range vwk.StoragePolicies {
		if _, ok := oldPolicies[policy]; !ok {
			modified = true
		}
		storageSpecs = append(storageSpecs, storageSpec{Policy: policy})
	}
	ns.StorageSpecs = storageSpecs

	if ns.VmServiceSpec == nil {
		modified = true
	} else {
		if !reflect.DeepEqual(ns.VmServiceSpec.VmClasses, vwk.VirtualMachineClasses) {
			ns.VmServiceSpec.VmClasses = vwk.VirtualMachineClasses
			modified = true
		}
		if !reflect.DeepEqual(ns.VmServiceSpec.ContentLibraries, vwk.ContentLibraries) {
			ns.VmServiceSpec.ContentLibraries = vwk.ContentLibraries
			modified = true
		}
	}
	if !modified {
		return false, nil
	}

	client := newClient(apiclient.region.VsphereInsecureSSL)
	url := fmt.Sprintf("https://%s/api/vcenter/namespaces/instances/%s", apiclient.region.VsphereHost, namespace)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader(apiSessionID, apiclient.apiSession).
		SetBody(ns).
		Patch(url)

	if err != nil {
		return true, fmt.Errorf("PATCH request %s failed: %w", url, err)
	}

	if resp.StatusCode() != http.StatusNoContent {
		return true, fmt.Errorf("PATCH request %s has unexpected status %s", url, resp.Status())
	}
	return true, nil
}

func (apiclient *VsphereAPIClient) DeleteNamespace(namespace string) error {
	client := newClient(apiclient.region.VsphereInsecureSSL)

	url := fmt.Sprintf("https://%s/api/vcenter/namespaces/instances/%s", apiclient.region.VsphereHost, namespace)
	resp, err := client.R().
		SetHeader(apiSessionID, apiclient.apiSession).
		Delete(url)

	if err != nil {
		return fmt.Errorf("DELETE request %s failed: %w", url, err)
	}
	switch resp.StatusCode() {
	case http.StatusNotFound:
		return nil
	case http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("GET request %s has unexpected status %s", url, resp.Status())
	}
}
