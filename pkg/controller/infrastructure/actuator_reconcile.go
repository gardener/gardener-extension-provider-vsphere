/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 *
 */

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	apishelper "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	apisvspherev1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/ensurer"
)

type preparedReconcile struct {
	cloudProfileConfig *apisvsphere.CloudProfileConfig
	infraConfig        *apisvsphere.InfrastructureConfig
	region             *apisvsphere.RegionSpec
	spec               infrastructure.NSXTInfraSpec
	ensurer            infrastructure.NSXTInfrastructureEnsurer
}

func (a *actuator) prepareReconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) (*preparedReconcile, error) {
	cloudProfileConfig, err := apishelper.GetCloudProfileConfig(cluster)
	if err != nil {
		return nil, err
	}

	infraConfig, err := apishelper.GetInfrastructureConfig(cluster)
	if err != nil {
		return nil, err
	}

	creds, err := vsphere.GetCredentials(ctx, a.Client(), infra.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	region := apishelper.FindRegion(infra.Spec.Region, cloudProfileConfig)
	if region == nil {
		return nil, fmt.Errorf("region %q not found in cloud profile", infra.Spec.Region)
	}
	if len(region.Zones) == 0 {
		return nil, fmt.Errorf("region %q has no zones in cloud profile", infra.Spec.Region)
	}
	dnsServers := cloudProfileConfig.DNSServers
	if len(region.DNSServers) > 0 {
		dnsServers = region.DNSServers
	}

	nsxtConfig := &infrastructure.NSXTConfig{
		User:         creds.NSXT_Infrastructure().Username,
		Password:     creds.NSXT_Infrastructure().Password,
		Host:         region.NSXTHost,
		InsecureFlag: region.NSXTInsecureSSL,
		RemoteAuth:   region.NSXTRemoteAuth,
	}

	spec := infrastructure.NSXTInfraSpec{
		EdgeClusterName:   region.EdgeCluster,
		TransportZoneName: region.TransportZone,
		Tier0GatewayName:  region.LogicalTier0Router,
		SNATIPPoolName:    region.SNATIPPool,
		GardenID:          a.gardenID,
		GardenName:        cloudProfileConfig.NamePrefix,
		ClusterName:       infra.Namespace,
		WorkersNetwork:    *cluster.Shoot.Spec.Networking.Nodes,
		DNSServers:        dnsServers,
	}

	infraEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(a.logger, nsxtConfig)
	if err != nil {
		return nil, err
	}

	prepared := &preparedReconcile{
		cloudProfileConfig: cloudProfileConfig,
		infraConfig:        infraConfig,
		region:             region,
		spec:               spec,
		ensurer:            infraEnsurer,
	}
	return prepared, nil
}

func (a *actuator) getInfrastructureState(infra *extensionsv1alpha1.Infrastructure) (*apisvsphere.NSXTInfraState, *bool, error) {
	infraStatus, err := apishelper.GetInfrastructureStatus(infra.Namespace, infra.Status.ProviderStatus)
	if err != nil {
		return nil, nil, err
	}
	if infraStatus == nil {
		return nil, nil, nil
	}
	return infraStatus.NSXTInfraState, infraStatus.CreationStarted, nil
}

func (a *actuator) reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	state, creationStarted, err := a.getInfrastructureState(infra)
	if err != nil {
		return err
	}

	prepared, err := a.prepareReconcile(ctx, infra, cluster)
	if creationStarted == nil || !*creationStarted {
		// early status update to allow deletion on wrong credentials
		if err == nil {
			err = prepared.ensurer.CheckConnection()
		}
		b := err == nil
		creationStarted = &b
		errUpdate := a.updateProviderStatus(ctx, infra, state, prepared, creationStarted)
		if err != nil {
			return err
		}
		if errUpdate != nil {
			return errUpdate
		}
	}
	if err != nil {
		return err
	}

	if state == nil {
		state, err = prepared.ensurer.NewStateWithVersion(prepared.infraConfig.OverwriteNSXTInfraVersion)
		if err != nil {
			return errors.Wrapf(err, "NewStateWithVersion failed")
		}
	}
	err = prepared.ensurer.EnsureInfrastructure(prepared.spec, state)
	errUpdate := a.updateProviderStatus(ctx, infra, state, prepared, creationStarted)
	if err != nil {
		return err
	}
	return errUpdate
}

// Helper functions

func (a *actuator) updateProviderStatus(
	ctx context.Context,
	infra *extensionsv1alpha1.Infrastructure,
	newState *apisvsphere.NSXTInfraState,
	prepared *preparedReconcile,
	creationStarted *bool,
) error {
	status, err := a.makeProviderInfrastructureStatus(newState, prepared, creationStarted)
	if err == nil {
		err = a.doUpdateProviderStatus(ctx, infra, status)
	}
	if err != nil {
		a.logFailedSaveState(err, newState)
	}
	return err
}

func (a *actuator) makeProviderInfrastructureStatus(
	state *apisvsphere.NSXTInfraState,
	prepared *preparedReconcile,
	creationStarted *bool,
) (*apisvsphere.InfrastructureStatus, error) {
	safe := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	status := &apisvsphere.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
		NSXTInfraState:  state,
		CreationStarted: creationStarted,
	}

	if prepared != nil {
		cloudProfileConfig := prepared.cloudProfileConfig
		region := prepared.region
		zoneConfigs := map[string]apisvsphere.ZoneConfig{}
		for _, z := range region.Zones {
			datacenter := region.Datacenter
			if z.Datacenter != nil {
				datacenter = z.Datacenter
			}
			if datacenter == nil {
				return nil, fmt.Errorf("datacenter not set in zone %s", z.Name)
			}
			datastore := region.Datastore
			datastoreCluster := region.DatastoreCluster
			if z.Datastore != nil {
				datastore = z.Datastore
				datastoreCluster = nil
			} else if z.DatastoreCluster != nil {
				datastore = nil
				datastoreCluster = z.DatastoreCluster
			}
			zoneConfigs[z.Name] = apisvsphere.ZoneConfig{
				Datacenter:       safe(datacenter),
				ComputeCluster:   safe(z.ComputeCluster),
				ResourcePool:     safe(z.ResourcePool),
				HostSystem:       safe(z.HostSystem),
				Datastore:        safe(datastore),
				DatastoreCluster: safe(datastoreCluster),
			}
		}
		status.VsphereConfig = apisvsphere.VsphereConfig{
			Folder:      cloudProfileConfig.Folder,
			Region:      region.Name,
			ZoneConfigs: zoneConfigs,
		}
	}

	return status, nil
}

func (a *actuator) doUpdateProviderStatus(
	ctx context.Context,
	infra *extensionsv1alpha1.Infrastructure,
	status *apisvsphere.InfrastructureStatus,
) error {

	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.Client(), infra, func() error {
		statusV1alpha1 := &apisvspherev1alpha1.InfrastructureStatus{}
		err := a.Scheme().Convert(status, statusV1alpha1, nil)
		if err != nil {
			return err
		}
		statusV1alpha1.SetGroupVersionKind(apisvspherev1alpha1.SchemeGroupVersion.WithKind("InfrastructureStatus"))
		infra.Status.ProviderStatus = &runtime.RawExtension{Object: statusV1alpha1}
		return nil
	})
}

func (a *actuator) logFailedSaveState(err error, state *apisvsphere.NSXTInfraState) {
	bytes, err2 := json.Marshal(state)
	stateString := ""
	if err2 == nil {
		stateString = string(bytes)
	} else {
		stateString = err2.Error()
	}
	a.logger.Error(err, "persisting infrastructure state failed", "state", stateString)
}
