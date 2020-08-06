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

package validation

import (
	"fmt"
	"regexp"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

var validLoadBalancerSizeValues = sets.NewString("SMALL", "MEDIUM", "LARGE")
var namePrefixPattern = regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(profileSpec *gardencorev1beta1.CloudProfileSpec, profileConfig *apisvsphere.CloudProfileConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	loadBalancerSizePath := field.NewPath("constraints", "loadBalancerConfig", "size")
	if profileConfig.Constraints.LoadBalancerConfig.Size == "" {
		allErrs = append(allErrs, field.Required(loadBalancerSizePath, "must provide the load balancer size"))
	} else {
		if !validLoadBalancerSizeValues.Has(profileConfig.Constraints.LoadBalancerConfig.Size) {
			allErrs = append(allErrs, field.NotSupported(loadBalancerSizePath,
				profileConfig.Constraints.LoadBalancerConfig.Size, validLoadBalancerSizeValues.List()))
		}
	}

	if profileConfig.NamePrefix == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("namePrefix"), "must provide name prefix for NSX-T resources"))
	} else if !namePrefixPattern.MatchString(profileConfig.NamePrefix) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("namePrefix"), profileConfig.NamePrefix,
			"must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"))
	}
	if profileConfig.DefaultClassStoragePolicyName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("defaultClassStoragePolicyName"), "must provide defaultClassStoragePolicyName"))
	}

	machineImagesPath := field.NewPath("machineImages")
	if len(profileConfig.MachineImages) == 0 {
		allErrs = append(allErrs, field.Required(machineImagesPath, "must provide at least one machine image"))
	}
	machineImageVersions := map[string]sets.String{}
	for _, image := range profileSpec.MachineImages {
		versions := sets.String{}
		for _, version := range image.Versions {
			versions.Insert(version.Version)
		}
		machineImageVersions[image.Name] = versions
	}

	checkMachineImage := func(idxPath *field.Path, machineImage apisvsphere.MachineImages) {
		definedVersions := sets.String{}
		if len(machineImage.Name) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), "must provide a name"))
		}
		versions, ok := machineImageVersions[machineImage.Name]
		if !ok {
			allErrs = append(allErrs, field.Forbidden(idxPath.Child("name"), "machineImage with this name is not defined in cloud profile spec"))
		}

		if len(machineImage.Versions) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("versions"), fmt.Sprintf("must provide at least one version for machine image %q", machineImage.Name)))
		}
		for j, version := range machineImage.Versions {
			jdxPath := idxPath.Child("versions").Index(j)

			if len(version.Version) == 0 {
				allErrs = append(allErrs, field.Required(jdxPath.Child("version"), "must provide a version"))
			} else {
				if definedVersions.Has(version.Version) {
					allErrs = append(allErrs, field.Duplicate(jdxPath.Child("version"), version.Version))
				}
				definedVersions.Insert(version.Version)
				if !versions.Has(version.Version) {
					allErrs = append(allErrs, field.Invalid(jdxPath.Child("version"), version.Version, "not defined as version in cloud profile spec"))
				}
			}
			if len(version.Path) == 0 {
				allErrs = append(allErrs, field.Required(jdxPath.Child("path"), "must provide a path of VM template"))
			}
		}
		missing := versions.Difference(definedVersions)
		if missing.Len() > 0 {
			allErrs = append(allErrs, field.Invalid(idxPath, strings.Join(missing.List(), ","), "missing versions"))
		}
	}
	for i, machineImage := range profileConfig.MachineImages {
		checkMachineImage(machineImagesPath.Index(i), machineImage)
	}

	machineTypeNames := sets.String{}
	for _, machineType := range profileSpec.MachineTypes {
		machineTypeNames.Insert(machineType.Name)
	}
	machineTypeOptionsNames := sets.String{}
	for i, machineTypeOptions := range profileConfig.MachineTypeOptions {
		idxPath := field.NewPath("machineTypeOptions").Index(i)
		if len(machineTypeOptions.Name) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), "must provide a name"))
			continue
		}
		if machineTypeOptionsNames.Has(machineTypeOptions.Name) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("name"), machineTypeOptions.Name))
		}
		machineTypeOptionsNames.Insert(machineTypeOptions.Name)
		if !machineTypeNames.Has(machineTypeOptions.Name) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), machineTypeOptions.Name, "machineType with this name is not defined"))
		}
	}

	regionsPath := field.NewPath("regions")
	if len(profileConfig.Regions) == 0 {
		allErrs = append(allErrs, field.Required(regionsPath, "must provide at least one region"))
	}
	for i, region := range profileConfig.Regions {
		regionPath := regionsPath.Index(i)
		if region.Name == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("name"), "must provide region name"))
		}
		if region.VsphereHost == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("vsphereHost"), fmt.Sprintf("must provide vSphere host for region %s", region.Name)))
		}
		if region.NSXTHost == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("nsxtHost"), fmt.Sprintf("must provide NSX-T  host for region %s", region.Name)))
		}
		if region.SNATIPPool == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("snatIPPool"), fmt.Sprintf("must provide SNAT IP pool for region %s", region.Name)))
		}
		if region.TransportZone == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("transportZone"), fmt.Sprintf("must provide transport zone for region %s", region.Name)))
		}
		if region.LogicalTier0Router == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("logicalTier0Router"), fmt.Sprintf("must provide logical tier 0 router for region %s", region.Name)))
		}
		if region.EdgeCluster == "" {
			allErrs = append(allErrs, field.Required(regionPath.Child("edgeCluster"), fmt.Sprintf("must provide edge cluster for region %s", region.Name)))
		}
		if len(region.Zones) == 0 {
			allErrs = append(allErrs, field.Required(regionPath.Child("zones"), fmt.Sprintf("must provide edge cluster for region %s", region.Name)))
		}
		if len(profileConfig.DNSServers) == 0 && len(region.DNSServers) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("dnsServers"), "must provide dnsServers globally or for each region"))
			allErrs = append(allErrs, field.Required(regionPath.Child("dnsServers"), fmt.Sprintf("must provide dnsServers globally or for region %s", region.Name)))
		}
		for j, zone := range region.Zones {
			zonePath := regionPath.Child("zones").Index(j)
			if zone.Name == "" {
				allErrs = append(allErrs, field.Required(zonePath.Child("name"), fmt.Sprintf("must provide zone name in zones for region %s", region.Name)))
			}
			if !isSet(zone.Datacenter) && !isSet(region.Datacenter) {
				allErrs = append(allErrs, field.Required(zonePath.Child("datacenter"), fmt.Sprintf("must provide data center either for region %s or its zone %s", region.Name, zone.Name)))
			}
			if !isSet(zone.Datastore) && !isSet(zone.DatastoreCluster) && !isSet(region.Datastore) && !isSet(region.DatastoreCluster) {
				allErrs = append(allErrs, field.Required(zonePath.Child("datastore"), fmt.Sprintf("must provide either data store or data store cluster for either region %s or its zone %s", region.Name, zone.Name)))
			}
			if !isSet(zone.ComputeCluster) && !isSet(zone.ResourcePool) && !isSet(zone.HostSystem) {
				allErrs = append(allErrs, field.Required(zonePath.Child("resourcePool"), fmt.Sprintf("must provide either compute cluster, resource pool, or hostsystem for region %s, zone %s", region.Name, zone.Name)))
			}
		}
		for i, machineImage := range region.MachineImages {
			checkMachineImage(regionPath.Child("machineImages").Index(i), machineImage)
		}
	}

	return allErrs
}

func isSet(s *string) bool {
	return s != nil && *s != ""
}
