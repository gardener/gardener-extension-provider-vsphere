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
	"io/ioutil"
	"os"
	"testing"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
)

type testdata struct {
	v1alpha1.VsphereWithKubernetes
	Username string `json:"username"`
	Password string `json:"password"`
}

func TestCreateUpdateDeleteNamespace(t *testing.T) {
	filename := os.Getenv("PROVIDER_VSPHERE_NAMESPACE_TESTDATA_FILE")
	if filename == "" {
		t.Skipf("No path to testdata specified by environmental variable PROVIDER_VSPHERE_NAMESPACE_TESTDATA_FILE")
		return
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("reading testdata from %s failed with %s", filename, err)
		return
	}

	cfg := testdata{}
	err = yaml.Unmarshal([]byte(content), &cfg)
	if err != nil {
		t.Errorf("Unmarshalling testdata failed with %s", err)
		return
	}

	cloudcfg := &v1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProfileConfig",
			APIVersion: "vsphere.provider.extensions.gardener.cloud/v1alpha1",
		},
		VsphereWithKubernetes: &cfg.VsphereWithKubernetes,
	}
	data, err := yaml.Marshal(&cloudcfg)
	if err != nil {
		t.Errorf("marshalling VsphereWithKubernetes failed with %s", err)
		return
	}

	vcloudcfg, err := helper.DecodeCloudProfileConfig(&runtime.RawExtension{Raw: data}, field.NewPath("dummy"))
	if err != nil {
		t.Errorf("decoding VsphereWithKubernetes failed with %s", err)
		return
	}

	vwk := vcloudcfg.VsphereWithKubernetes
	client, err := GetVsphereAPISession(vwk.Regions[0], vsphere.UserPass{Username: cfg.Username, Password: cfg.Password})
	if err != nil {
		t.Errorf("GetVsphereAPISession failed with %s", err)
		return
	}

	err = client.CreateNamespace("foo", vwk)
	if err != nil {
		t.Errorf("CreateNamespace failed with %s", err)
		return
	}

	cluster, err := client.GetNamespaceCluster("foo")
	if err != nil {
		t.Errorf("GetNamespaceCluster failed with %s", err)
		return
	}
	if cluster != vwk.Regions[0].Cluster {
		t.Errorf("cluster mismatch %s != %s", cluster, vwk.Regions[0].Cluster)
		return
	}

	modified, err := client.UpdateNamespace("foo", vwk)
	if err != nil {
		t.Errorf("UpdateNamespace failed with %s", err)
		return
	}
	if modified {
		t.Errorf("unexpected modified")
		return
	}

	vwk.VirtualMachineClasses = vwk.VirtualMachineClasses[1:]
	modified, err = client.UpdateNamespace("foo", vwk)
	if err != nil {
		t.Errorf("UpdateNamespace2 failed with %s", err)
		return
	}
	if !modified {
		t.Errorf("unexpected not modified")
		return
	}

	err = client.DeleteNamespace("foo")
	if err != nil {
		t.Errorf("DeleteNamespace failed with %s", err)
		return
	}
}
