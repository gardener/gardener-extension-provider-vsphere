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
	"net"

	"github.com/gardener/gardener/pkg/apis/core"
	validationutils "github.com/gardener/gardener/pkg/utils/validation"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateNetworking validates the network settings of a Shoot.
func ValidateNetworking(networking *core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if networking == nil {
		allErrs = append(allErrs, field.Required(fldPath, "networking can not be nil for vSphere shoots"))
		return allErrs
	}

	if networking.Nodes == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("nodes"), "a nodes CIDR must be provided for vSphere shoots"))
	}

	return allErrs
}

// ValidateNetworkingUpdate validates updates to shoot's networking settings.
func ValidateNetworkingUpdate(oldNetworking, networking *core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if oldNetworking != nil && oldNetworking.Nodes != nil {
		if _, _, err := net.ParseCIDR(*oldNetworking.Nodes); err == nil {
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(networking.Nodes, oldNetworking.Nodes, fldPath.Child("nodes"))...)
		}
	}

	return allErrs
}

// ValidateWorkers validates the workers of a Shoot.
func ValidateWorkers(workers []core.Worker, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, worker := range workers {

		workerFldPath := fldPath.Index(i)
		if len(worker.Zones) == 0 {
			allErrs = append(allErrs, field.Required(workerFldPath.Child("zones"), "at least one zone must be configured"))
			continue
		}

		zones := sets.NewString()
		for j, zone := range worker.Zones {
			if zones.Has(zone) {
				allErrs = append(allErrs, field.Invalid(workerFldPath.Child("zones").Index(j), zone, "must only be specified once per worker group"))
				continue
			}
			zones.Insert(zone)
		}
	}

	return allErrs
}

// ValidateWorkersUpdate validates updates on Workers.
func ValidateWorkersUpdate(oldWorkers, newWorkers []core.Worker, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, newWorker := range newWorkers {
		for _, oldWorker := range oldWorkers {
			if newWorker.Name == oldWorker.Name {
				if validationutils.ShouldEnforceImmutability(newWorker.Zones, oldWorker.Zones) {
					allErrs = append(allErrs, apivalidation.ValidateImmutableField(newWorker.Zones, oldWorker.Zones, fldPath.Index(i).Child("zones"))...)
				}
				break
			}
		}
	}
	return allErrs
}
