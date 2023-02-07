// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	. "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"
)

var _ = Describe("ControlPlaneConfig validation", func() {
	var (
		nilPath *field.Path

		controlPlane *api.ControlPlaneConfig
	)

	BeforeEach(func() {
		controlPlane = &api.ControlPlaneConfig{
			LoadBalancerClasses: []api.CPLoadBalancerClass{
				{Name: "default"},
			},
		}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for a valid configuration", func() {
			Expect(ValidateControlPlaneConfig(controlPlane, "", nilPath)).To(BeEmpty())
		})

		It("should require the name of a load balancer class", func() {
			controlPlane.LoadBalancerClasses[0].Name = ""

			errorList := ValidateControlPlaneConfig(controlPlane, "", nilPath)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("loadBalancerClasses.name"),
			}))))
		})

		It("should check valid value for load balancer size", func() {
			s := "LARGE"
			controlPlane.LoadBalancerSize = &s
			errorList := ValidateControlPlaneConfig(controlPlane, "", nilPath)
			Expect(errorList).To(BeEmpty())

			s2 := "foo"
			controlPlane.LoadBalancerSize = &s2
			errorList = ValidateControlPlaneConfig(controlPlane, "", nilPath)
			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("loadBalancerSize"),
			}))))
		})

		It("should fail with invalid CCM feature gates", func() {
			controlPlane.CloudControllerManager = &api.CloudControllerManagerConfig{
				FeatureGates: map[string]bool{
					"AnyVolumeDataSource":      true,
					"CustomResourceValidation": true,
					"Foo":                      true,
				},
			}

			errorList := ValidateControlPlaneConfig(controlPlane, "1.23.14", nilPath)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("cloudControllerManager.featureGates.CustomResourceValidation"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("cloudControllerManager.featureGates.Foo"),
				})),
			))
		})
	})

	Describe("#ValidateControlPlaneConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfigUpdate(controlPlane, controlPlane, nilPath)).To(BeEmpty())
		})

		It("should not allow to change the load balancer size", func() {
			newControlPlane := &api.ControlPlaneConfig{
				LoadBalancerClasses: []api.CPLoadBalancerClass{
					{Name: "default"},
				},
				LoadBalancerSize: sp("LARGE"),
			}
			Expect(ValidateControlPlaneConfigUpdate(controlPlane, newControlPlane, nilPath)).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("loadBalancerSize"),
			}))))
		})

		It("should not allow to change the default load balancer class", func() {
			oldControlPlane := &api.ControlPlaneConfig{
				LoadBalancerClasses: []api.CPLoadBalancerClass{
					{Name: "default", IPPoolName: sp("oldpool"), TCPAppProfileName: sp("tcpprof")},
				},
			}
			newControlPlane := &api.ControlPlaneConfig{
				LoadBalancerClasses: []api.CPLoadBalancerClass{
					{Name: "default", IPPoolName: sp("newpool"), UDPAppProfileName: sp("udpprof")},
				},
			}
			Expect(ValidateControlPlaneConfigUpdate(oldControlPlane, newControlPlane, nilPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("loadBalancerClasses.ipPoolName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("loadBalancerClasses.tcpAppProfileName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("loadBalancerClasses.udpAppProfileName"),
				})),
			))
		})
	})

	Describe("#ValidateControlPlaneConfigAgainstCloudProfile", func() {
		var (
			region = "foo"
			zone   = "some-zone"

			cloudProfileConfig *api.CloudProfileConfig
			cloudProfile       *gardencorev1beta1.CloudProfile
		)

		BeforeEach(func() {
			cloudProfile = &gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					Regions: []gardencorev1beta1.Region{
						{
							Name: region,
							Zones: []gardencorev1beta1.AvailabilityZone{
								{Name: zone},
							},
						},
					},
				},
			}

			cloudProfileConfig = &api.CloudProfileConfig{
				Constraints: api.Constraints{
					LoadBalancerConfig: api.LoadBalancerConfig{
						Size: "MEDIUM",
						Classes: []api.LoadBalancerClass{
							{
								Name:       "default",
								IPPoolName: sp("lbpool"),
							},
							{
								Name:       "public",
								IPPoolName: sp("lbpool2"),
							},
						},
					},
				},
			}
		})

		It("should return no errors if the name is not defined in the constraints", func() {
			controlPlane.LoadBalancerClasses[0].Name = "bar"

			errorList := ValidateControlPlaneConfigAgainstCloudProfile(controlPlane, "testRegion", cloudProfile, cloudProfileConfig, nilPath)

			Expect(errorList).To(BeEmpty())
		})

	})
})
