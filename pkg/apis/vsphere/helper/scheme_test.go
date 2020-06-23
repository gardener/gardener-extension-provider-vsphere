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

package helper_test

import (
	vsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/install"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Helper (decode)", func() {
	var (
		s                  *runtime.Scheme
		ctx                common.ClientContext
		cluster            *controller.Cluster
		cloudProfileConfig *vsphere.CloudProfileConfig
	)

	BeforeEach(func() {
		s = scheme.Scheme
		install.Install(s)
		ctx = common.ClientContext{}
		err := ctx.InjectScheme(s)
		if err != nil {
			panic(err)
		}

		cluster = &controller.Cluster{
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			CloudProfile: &gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: "coreos",
							Versions: []gardencorev1beta1.ExpirableVersion{
								{
									Version: "2191.5.0",
								},
							},
						},
					},
					ProviderConfig: &runtime.RawExtension{Raw: []byte(`
apiVersion: vsphere.provider.extensions.gardener.cloud/v1alpha1
kind: CloudProfileConfig
defaultClassStoragePolicyName: "vSAN Default Storage Policy"
namePrefix: nameprefix
folder: gardener
regions:
- name: testregion
  vsphereHost: vsphere.host.internal
  vsphereInsecureSSL: true
  nsxtHost: nsxt.host.internal
  nsxtInsecureSSL: true
  transportZone: tz
  logicalTier0Router: lt0router
  edgeCluster: edgecluster
  snatIPPool: snatIpPool
  datacenter: scc01-DC
  datastore: A800_VMwareB
  zones:
  - name: testzone
    computeCluster: scc01w01-DEV
constraints:
  loadBalancerConfig:
    size: SMALL
    classes:
    - name: default
      ipPoolName: lbpool
dnsServers:
- "1.2.3.4"
machineImages:
- name: coreos
  versions:
  - version: 2191.5.0
    path: gardener/templates/coreos-2191.5.0
    guestId: coreos64Guest
`)},
				},
			},
		}

		cloudProfileConfig = &vsphere.CloudProfileConfig{
			DefaultClassStoragePolicyName: "vSAN Default Storage Policy",
			NamePrefix:                    "nameprefix",
			Folder:                        "gardener",
			Regions: []vsphere.RegionSpec{
				{
					Name:               "testregion",
					VsphereHost:        "vsphere.host.internal",
					VsphereInsecureSSL: true,
					NSXTHost:           "nsxt.host.internal",
					NSXTInsecureSSL:    true,
					TransportZone:      "tz",
					LogicalTier0Router: "lt0router",
					EdgeCluster:        "edgecluster",
					SNATIPPool:         "snatIpPool",
					Datacenter:         sp("scc01-DC"),
					Datastore:          sp("A800_VMwareB"),
					Zones: []vsphere.ZoneSpec{
						{
							Name:           "testzone",
							ComputeCluster: sp("scc01w01-DEV"),
						},
					},
				},
			},
			Constraints: vsphere.Constraints{
				LoadBalancerConfig: vsphere.LoadBalancerConfig{
					Size: "SMALL",
					Classes: []vsphere.LoadBalancerClass{
						{
							Name:       "default",
							IPPoolName: sp("lbpool"),
						},
					},
				},
			},
			DNSServers: []string{"1.2.3.4"},
			MachineImages: []vsphere.MachineImages{
				{Name: "coreos",
					Versions: []vsphere.MachineImageVersion{
						{
							Version: "2191.5.0",
							Path:    "gardener/templates/coreos-2191.5.0",
							GuestID: sp("coreos64Guest"),
						},
					},
				},
			},
		}
	})

	Describe("#GetCloudProfileConfig", func() {
		It("should decode the CloudProfileConfig", func() {
			result, err := helper.GetCloudProfileConfig(cluster)
			Expect(err).To(BeNil())

			Expect(result).To(Equal(cloudProfileConfig))
		})
	})
})
