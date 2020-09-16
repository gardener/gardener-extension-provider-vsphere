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

package validation

import (
	"fmt"

	"github.com/gardener/controller-manager-library/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

// ValidateControlPlaneConfig validates a ControlPlaneConfig object.
func ValidateControlPlaneConfig(controlPlaneConfig *apisvsphere.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, lbclass := range controlPlaneConfig.LoadBalancerClasses {
		if lbclass.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("loadBalancerClasses", "name"), "name of load balancer class must be set"))
		}
	}

	if controlPlaneConfig.LoadBalancerSize != nil {
		if !validLoadBalancerSizeValues.Has(*controlPlaneConfig.LoadBalancerSize) {
			allErrs = append(allErrs, field.NotSupported(field.NewPath("loadBalancerSize"),
				*controlPlaneConfig.LoadBalancerSize, validLoadBalancerSizeValues.List()))
		}
	}

	return allErrs
}

// ValidateControlPlaneConfigUpdate validates a ControlPlaneConfig object.
func ValidateControlPlaneConfigUpdate(oldConfig, newConfig *apisvsphere.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// load balancer size overwrite is immutable
	if isSet(oldConfig.LoadBalancerSize) != isSet(newConfig.LoadBalancerSize) ||
		isSet(oldConfig.LoadBalancerSize) && *oldConfig.LoadBalancerSize != *newConfig.LoadBalancerSize {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("loadBalancerSize"), "load balancer size cannot be changed"))
	}

	// default load balancer class overwrite is immutable
	constraintsClasses := []apisvsphere.LoadBalancerClass{
		{Name: apisvsphere.LoadBalancerDefaultClassName, IPPoolName: sp("dummy")},
	}
	oldDefault, _, err := OverwriteLoadBalancerClasses(constraintsClasses, oldConfig, nil)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath.Child("loadBalancerClasses"), err))
		return allErrs
	}
	newDefault, _, err := OverwriteLoadBalancerClasses(constraintsClasses, newConfig, nil)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath.Child("loadBalancerClasses"), err))
		return allErrs
	}
	if safeStr(oldDefault.IPPoolName) != safeStr(newDefault.IPPoolName) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("loadBalancerClasses", "ipPoolName"), fmt.Sprintf("default class %s is immutable", apisvsphere.LoadBalancerDefaultClassName)))
	}
	if safeStr(oldDefault.TCPAppProfileName) != safeStr(newDefault.TCPAppProfileName) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("loadBalancerClasses", "tcpAppProfileName"), fmt.Sprintf("default class %s is immutable", apisvsphere.LoadBalancerDefaultClassName)))
	}
	if safeStr(oldDefault.UDPAppProfileName) != safeStr(newDefault.UDPAppProfileName) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("loadBalancerClasses", "udpAppProfileName"), fmt.Sprintf("default class %s is immutable", apisvsphere.LoadBalancerDefaultClassName)))
	}
	return allErrs
}

// ValidateControlPlaneConfigAgainstCloudProfile validates the given ControlPlaneConfig against constraints in the given CloudProfile.
func ValidateControlPlaneConfigAgainstCloudProfile(cpConfig *apisvsphere.ControlPlaneConfig, shootRegion string, cloudProfile *gardencorev1beta1.CloudProfile, cloudProfileConfig *apisvsphere.CloudProfileConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	_, _, err := OverwriteLoadBalancerClasses(cloudProfileConfig.Constraints.LoadBalancerConfig.Classes, cpConfig, nil)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath.Child("loadBalancerClasses"), err))
	}

	return allErrs
}

func HasRelevantControlPlaneConfigUpdates(oldCpConfig *apisvsphere.ControlPlaneConfig, newCpConfig *apisvsphere.ControlPlaneConfig) bool {
	constraintsClasses := []apisvsphere.LoadBalancerClass{
		{Name: apisvsphere.LoadBalancerDefaultClassName, IPPoolName: sp("dummy")},
	}

	oldDefaultClass, _, _ := OverwriteLoadBalancerClasses(constraintsClasses, oldCpConfig, nil)
	newDefaultClass, _, _ := OverwriteLoadBalancerClasses(constraintsClasses, newCpConfig, nil)

	return oldDefaultClass == nil || newDefaultClass == nil ||
		!equality.Semantic.DeepEqual(*oldDefaultClass, *newDefaultClass)
}

// OverwriteLoadBalancerClasses uses load balancer constraints classes as defaults for cpConfig load balancer classes.
// The checkIPPoolName function is optional. It can be used to check that the name provided by the shoot manifest is
// authorized.
func OverwriteLoadBalancerClasses(constraintsClasses []apisvsphere.LoadBalancerClass, cpConfig *apisvsphere.ControlPlaneConfig,
	checkIPPoolName func(name string) error) (*apisvsphere.LoadBalancerClass, []apisvsphere.LoadBalancerClass, error) {
	var defaultClass *apisvsphere.LoadBalancerClass
	loadBalancersClasses := []apisvsphere.LoadBalancerClass{}
	if len(cpConfig.LoadBalancerClasses) == 0 {
		cpConfig.LoadBalancerClasses = []apisvsphere.CPLoadBalancerClass{{Name: apisvsphere.LoadBalancerDefaultClassName}}
	}
	for i, class := range constraintsClasses {
		if i == 0 || class.Name == apisvsphere.LoadBalancerDefaultClassName {
			class0 := class
			defaultClass = &class0
		}
	}
	if defaultClass == nil {
		return nil, nil, fmt.Errorf("no load balancer classes defined in cloud profile config")
	}

	for _, cpClass := range cpConfig.LoadBalancerClasses {
		lbClass := apisvsphere.LoadBalancerClass{Name: cpClass.Name}
		var constraintClass *apisvsphere.LoadBalancerClass
		for _, class := range constraintsClasses {
			if class.Name == cpClass.Name {
				constraintClass = &class
				break
			}
		}
		if !utils.IsEmptyString(cpClass.IPPoolName) {
			// if IP pool is set in shoot manifest, perform optional authorization check
			if checkIPPoolName != nil {
				err := checkIPPoolName(*cpClass.IPPoolName)
				if err != nil {
					return nil, nil, err
				}
			}
			lbClass.IPPoolName = cpClass.IPPoolName
		} else if constraintClass != nil && !utils.IsEmptyString(constraintClass.IPPoolName) {
			lbClass.IPPoolName = constraintClass.IPPoolName
		}
		if !utils.IsEmptyString(cpClass.TCPAppProfileName) {
			lbClass.TCPAppProfileName = cpClass.TCPAppProfileName
		} else if constraintClass != nil && !utils.IsEmptyString(constraintClass.TCPAppProfileName) {
			lbClass.TCPAppProfileName = constraintClass.TCPAppProfileName
		}
		if !utils.IsEmptyString(cpClass.UDPAppProfileName) {
			lbClass.UDPAppProfileName = cpClass.UDPAppProfileName
		} else if constraintClass != nil && !utils.IsEmptyString(constraintClass.UDPAppProfileName) {
			lbClass.UDPAppProfileName = constraintClass.UDPAppProfileName
		}
		loadBalancersClasses = append(loadBalancersClasses, lbClass)
		if lbClass.Name == defaultClass.Name {
			defaultClass = &lbClass
		}
	}

	if utils.IsEmptyString(defaultClass.IPPoolName) {
		return nil, nil, fmt.Errorf("load balancer default class %q must specify ipPoolName in cloud profile", defaultClass.Name)
	}

	return defaultClass, loadBalancersClasses, nil
}

func sp(s string) *string {
	return &s
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
