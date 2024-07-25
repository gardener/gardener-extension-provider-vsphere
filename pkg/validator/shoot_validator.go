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

package validator

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	vspherevalidation "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"
)

type validationContext struct {
	shoot              *core.Shoot
	infraConfig        *vsphere.InfrastructureConfig
	cpConfig           *vsphere.ControlPlaneConfig
	cloudProfile       *gardencorev1beta1.CloudProfile
	cloudProfileConfig *vsphere.CloudProfileConfig
}

var (
	specPath           = field.NewPath("spec")
	providerConfigPath = specPath.Child("providerConfig")
	nwPath             = specPath.Child("networking")
	providerPath       = specPath.Child("provider")
	secretBindintPath  = specPath.Child("secretBindingName")
	infraConfigPath    = providerPath.Child("infrastructureConfig")
	cpConfigPath       = providerPath.Child("controlPlaneConfig")
	workersPath        = providerPath.Child("workers")
)

func (v *Shoot) validateShootCreation(ctx context.Context, shoot *core.Shoot) error {
	valContext, err := newValidationContext(ctx, v.client, shoot)
	if err != nil {
		return err
	}

	allErrs := field.ErrorList{}

	allErrs = append(allErrs, vspherevalidation.ValidateInfrastructureConfigAgainstCloudProfile(valContext.infraConfig, shoot.Spec.Region, valContext.cloudProfileConfig, infraConfigPath)...)
	allErrs = append(allErrs, vspherevalidation.ValidateControlPlaneConfigAgainstCloudProfile(valContext.cpConfig, shoot.Spec.Region, valContext.cloudProfile, valContext.cloudProfileConfig, cpConfigPath)...)
	allErrs = append(allErrs, v.validateShoot(valContext)...)
	if err := v.validateShootSecret(ctx, shoot); err != nil {
		allErrs = append(allErrs, field.Invalid(secretBindintPath, shoot.Spec.SecretBindingName, fmt.Sprintf("invalid cloud provider credentials: %v", err)))
	}
	return allErrs.ToAggregate()
}

func (v *Shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	oldValContext, err := newValidationContext(ctx, v.client, oldShoot)
	if err != nil {
		return err
	}

	valContext, err := newValidationContext(ctx, v.client, shoot)
	if err != nil {
		return err
	}

	allErrs := field.ErrorList{}

	allErrs = append(allErrs, vspherevalidation.ValidateNetworkingUpdate(oldShoot.Spec.Networking, shoot.Spec.Networking, nwPath)...)
	allErrs = append(allErrs, vspherevalidation.ValidateInfrastructureConfigUpdate(oldValContext.infraConfig, valContext.infraConfig, infraConfigPath)...)
	// Only validate against cloud profile when related configuration is updated.
	// This ensures that already running shoots won't break after constraints were removed from the cloud profile.
	if vspherevalidation.HasRelevantInfrastructureConfigUpdates(oldValContext.infraConfig, valContext.infraConfig) {
		allErrs = append(allErrs, vspherevalidation.ValidateInfrastructureConfigAgainstCloudProfile(valContext.infraConfig, shoot.Spec.Region, valContext.cloudProfileConfig, infraConfigPath)...)
	}

	allErrs = append(allErrs, vspherevalidation.ValidateControlPlaneConfigUpdate(oldValContext.cpConfig, valContext.cpConfig, cpConfigPath)...)
	// Only validate against cloud profile when related configuration is updated.
	// This ensures that already running shoots won't break after constraints were removed from the cloud profile.
	if vspherevalidation.HasRelevantControlPlaneConfigUpdates(oldValContext.cpConfig, valContext.cpConfig) {
		allErrs = append(allErrs, vspherevalidation.ValidateControlPlaneConfigAgainstCloudProfile(valContext.cpConfig, shoot.Spec.Region, valContext.cloudProfile, valContext.cloudProfileConfig, cpConfigPath)...)
	}

	allErrs = append(allErrs, vspherevalidation.ValidateWorkersUpdate(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers, workersPath)...)
	allErrs = append(allErrs, v.validateShoot(valContext)...)
	return allErrs.ToAggregate()
}

func (v *Shoot) validateShoot(context *validationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, vspherevalidation.ValidateNetworking(context.shoot.Spec.Networking, nwPath)...)
	allErrs = append(allErrs, vspherevalidation.ValidateInfrastructureConfig(context.infraConfig, infraConfigPath)...)
	allErrs = append(allErrs, vspherevalidation.ValidateControlPlaneConfig(context.cpConfig, context.shoot.Spec.Kubernetes.Version, cpConfigPath)...)
	allErrs = append(allErrs, vspherevalidation.ValidateWorkers(context.shoot.Spec.Provider.Workers, workersPath)...)
	return allErrs
}

func (v *Shoot) validateShootSecret(ctx context.Context, shoot *core.Shoot) error {
	var (
		secretBinding = &gardencorev1beta1.SecretBinding{}
	)

	if shoot.Spec.SecretBindingName == nil {
		return fmt.Errorf("secretBindingName can't be set to nil")
	}

	secretBindingKey := client.ObjectKey{
		Namespace: shoot.Namespace,
		Name:      *shoot.Spec.SecretBindingName,
	}

	if err := kutil.LookupObject(ctx, v.client, v.apiReader, secretBindingKey, secretBinding); err != nil {
		return err
	}

	var (
		secret    = &corev1.Secret{}
		secretKey = client.ObjectKey{
			Namespace: secretBinding.SecretRef.Namespace,
			Name:      secretBinding.SecretRef.Name,
		}
	)
	// Explicitly use the client.Reader to prevent controller-runtime to start Informer for Secrets
	// under the hood. The latter increases the memory usage of the component.
	if err := v.apiReader.Get(ctx, secretKey, secret); err != nil {
		return err
	}

	return vspherevalidation.ValidateCloudProviderSecret(secret)
}

func newValidationContext(ctx context.Context, c client.Client, shoot *core.Shoot) (*validationContext, error) {
	infraConfig := &vsphere.InfrastructureConfig{}
	if shoot.Spec.Provider.InfrastructureConfig != nil {
		var err error
		infraConfig, err = helper.DecodeInfrastructureConfig(shoot.Spec.Provider.InfrastructureConfig, infraConfigPath)
		if err != nil {
			return nil, err
		}
	}

	if shoot.Spec.Provider.ControlPlaneConfig == nil {
		return nil, field.Required(cpConfigPath, "controlPlaneConfig must be set for vSphere shoots")
	}
	cpConfig, err := helper.DecodeControlPlaneConfig(shoot.Spec.Provider.ControlPlaneConfig, cpConfigPath)
	if err != nil {
		return nil, err
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	cloudProfileKey := client.ObjectKey{
		Namespace: "",
		Name:      shoot.Spec.CloudProfileName,
	}

	if err := c.Get(ctx, cloudProfileKey, cloudProfile); err != nil {
		return nil, err
	}

	if cloudProfile.Spec.ProviderConfig == nil {
		return nil, fmt.Errorf("providerConfig is not given for cloud profile %q", cloudProfile.Name)
	}
	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile.Spec.ProviderConfig, providerConfigPath)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while reading the cloud profile %q: %v", cloudProfile.Name, err)
	}

	return &validationContext{
		shoot:              shoot,
		infraConfig:        infraConfig,
		cpConfig:           cpConfig,
		cloudProfile:       cloudProfile,
		cloudProfileConfig: cloudProfileConfig,
	}, nil
}
