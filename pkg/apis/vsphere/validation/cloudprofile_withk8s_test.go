// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation_test

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	. "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("ValidateCloudProfileConfig for vSphere with Kubernetes", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var cloudProfileConfig *apisvsphere.CloudProfileConfig
		var cloudProfileSpec *gardencorev1beta1.CloudProfileSpec

		BeforeEach(func() {
			cloudProfileConfig = &apisvsphere.CloudProfileConfig{
				DefaultClassStoragePolicyName: "default-class",
				Constraints: apisvsphere.Constraints{
					LoadBalancerConfig: apisvsphere.LoadBalancerConfig{
						Size: "MEDIUM",
						Classes: []apisvsphere.LoadBalancerClass{
							{
								Name:       "default",
								IPPoolName: sp("lbpool"),
							},
						},
					},
				},
				MachineImages: []apisvsphere.MachineImages{
					{
						Name: "gardenlinux",
						Versions: []apisvsphere.MachineImageVersion{
							{
								Version: "477.1.0",
								Path:    "gardenlinux-dev-vm-operator-477.1",
							},
							{
								Version: "318.8.0",
								Path:    "gardenlinux-dev-vm-operator-318.8",
							},
						},
					},
				},
				VsphereWithKubernetes: &apisvsphere.VsphereWithKubernetes{
					StoragePolicies:       []string{"11-22-33-44"},
					ContentLibraries:      []string{"55-66-77-88"},
					VirtualMachineClasses: []string{"best-effort-large", "best-effort-medium", "best-effort-small", "best-effort-xlarge"},
					Regions: []apisvsphere.K8sRegionSpec{
						{
							Name:        "region1",
							Cluster:     "domain-c123",
							VsphereHost: "vsphere.somewhere",
							Zones: []apisvsphere.K8sZoneSpec{
								{
									Name:               "region1a",
									VMStorageClassName: "vmstorageclass",
								},
							},
						},
					},
				},
			}
			cloudProfileSpec = &gardencorev1beta1.CloudProfileSpec{
				MachineTypes: []gardencorev1beta1.MachineType{
					{
						Name: "best-effort-medium",
					},
					{
						Name: "best-effort-large",
					},
				},
				MachineImages: []gardencorev1beta1.MachineImage{
					{
						Name: "gardenlinux",
						Versions: []gardencorev1beta1.MachineImageVersion{
							{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "318.8.0"}},
							{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "477.1.0"}},
						},
					},
				},
			}
		})

		Context("machine image validation", func() {
			It("should validate valid machine image version configuration", func() {
				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf())
			})

			It("should validate valid machine image version configuration", func() {
				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf())
			})

			It("should enforce that at least one machine image has been defined", func() {
				cloudProfileConfig.MachineImages = []apisvsphere.MachineImages{}

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages"),
				}))))
			})

			It("should forbid unsupported machine image configuration", func() {
				cloudProfileConfig.MachineImages = []apisvsphere.MachineImages{{}}

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("machineImages[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].versions"),
				}))))
			})

			It("should forbid unsupported machine image version configuration", func() {
				cloudProfileConfig.MachineImages = []apisvsphere.MachineImages{
					{
						Name:     "abc",
						Versions: []apisvsphere.MachineImageVersion{{}},
					},
				}

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("machineImages[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].versions[0].version"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].versions[0].path"),
				}))))
			})
		})

		Context("load balancer validation", func() {
			It("should have a load balancer size", func() {
				cloudProfileConfig.Constraints.LoadBalancerConfig.Size = ""

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("constraints.loadBalancerConfig.size"),
					"Detail": Equal("must provide the load balancer size"),
				}))))
			})

			It("should have a valid load balancer size value", func() {
				cloudProfileConfig.Constraints.LoadBalancerConfig.Size = "foo"

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("constraints.loadBalancerConfig.size"),
				}))))
			})
		})

		Context("resources validation", func() {
			It("should complain about missing vsphereWithKubernetes settings", func() {
				cloudProfileConfig.VsphereWithKubernetes = &apisvsphere.VsphereWithKubernetes{}

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.contentLibraries"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.virtualMachineClasses"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.storagePolicies"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions"),
				}))))
			})

			It("should complain about incomplete vsphereWithKubernetes.regions", func() {
				cloudProfileConfig.VsphereWithKubernetes.Regions = append(cloudProfileConfig.VsphereWithKubernetes.Regions, apisvsphere.K8sRegionSpec{})

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[1].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[1].cluster"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[1].vsphereHost"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[1].zones"),
				}))))
			})

			It("should complain about incomplete vsphereWithKubernetes.regions.zones", func() {
				cloudProfileConfig.VsphereWithKubernetes.Regions[0].Zones = []apisvsphere.K8sZoneSpec{{}}

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[0].zones[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.regions[0].zones[0].vmStorageClassName"),
				}))))
			})

			It("should complain about incomplete vsphereWithKubernetes.storagepolicies", func() {
				cloudProfileConfig.VsphereWithKubernetes.StoragePolicies = append(cloudProfileConfig.VsphereWithKubernetes.StoragePolicies, "")

				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("vsphereWithKubernetes.storagePolicies[1]"),
				}))))
			})

			It("should not complain for valid vsphereWithKubernetes settings", func() {
				errorList := ValidateCloudProfileConfig(cloudProfileSpec, cloudProfileConfig)
				Expect(errorList).To(ConsistOf())
			})
		})
	})
})
