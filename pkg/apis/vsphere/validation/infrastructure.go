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

package validation

import (
	"reflect"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
)

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *api.InfrastructureConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !isValidEnsurerVersion(infra.OverwriteNSXTInfraVersion) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("overwriteNSXTInfraVersion"),
			*infra.OverwriteNSXTInfraVersion, api.SupportedEnsurerVersions))
	}

	if infra.Networks != nil {
		pathNetworks := fldPath.Child("networks")
		if infra.Networks.Tier1GatewayPath == "" {
			allErrs = append(allErrs, field.Required(pathNetworks.Child("tier1GatewayPath"), "required if networks is specified"))
		}
		if infra.Networks.LoadBalancerServicePath == "" {
			allErrs = append(allErrs, field.Required(pathNetworks.Child("loadBalancerServicePath"), "required if networks is specified"))
		}
	}
	return allErrs
}

func isValidEnsurerVersion(version *string) bool {
	return version == nil || sets.NewString(api.SupportedEnsurerVersions...).Has(*version)
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *api.InfrastructureConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// networks is immutable
	if !reflect.DeepEqual(oldConfig.Networks, newConfig.Networks) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("networks"), "networks settings cannot be changed"))
	}

	return allErrs
}

// ValidateInfrastructureConfigAgainstCloudProfile validates the given InfrastructureConfig against constraints in the given CloudProfile.
func ValidateInfrastructureConfigAgainstCloudProfile(infra *api.InfrastructureConfig, shootRegion string, cloudProfileConfig *api.CloudProfileConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// HasRelevantInfrastructureConfigUpdates returns true if given InfrastructureConfig has relevant changes
func HasRelevantInfrastructureConfigUpdates(oldInfra *api.InfrastructureConfig, newInfra *api.InfrastructureConfig) bool {
	return false
}
