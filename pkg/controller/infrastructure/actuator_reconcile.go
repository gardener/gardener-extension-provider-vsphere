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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
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
	region             *apisvsphere.RegionSpec
	spec               infrastructure.NSXTInfraSpec
	state              *apisvsphere.NSXTInfraState
	ensurer            infrastructure.NSXTInfrastructureEnsurer
}

func (a *actuator) prepareReconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) (*preparedReconcile, error) {
	cloudProfileConfig, err := apishelper.GetCloudProfileConfig(cluster)
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
		User:         creds.NSXTUsername,
		Password:     creds.NSXTPassword,
		Host:         region.NSXTHost,
		InsecureFlag: region.NSXTInsecureSSL,
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

	ensurer, err := ensurer.NewNSXTInfrastructureEnsurer(a.logger, nsxtConfig)
	if err != nil {
		return nil, err
	}

	infraStatus, err := apishelper.GetInfrastructureStatus(infra.Namespace, infra.Status.ProviderStatus)
	if err != nil {
		return nil, err
	}

	prepared := &preparedReconcile{
		cloudProfileConfig: cloudProfileConfig,
		region:             region,
		ensurer:            ensurer,
		spec:               spec,
	}
	if infraStatus != nil {
		prepared.state = infraStatus.NSXTInfraState
	}
	return prepared, nil
}

func (a *actuator) reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	prepared, err := a.prepareReconcile(ctx, infra, cluster)
	if err != nil {
		return err
	}

	state := prepared.state
	if state == nil {
		state = &apisvsphere.NSXTInfraState{}
	}
	err = prepared.ensurer.EnsureInfrastructure(prepared.spec, state)
	errUpdate := a.updateProviderStatus(ctx, infra, state, prepared.cloudProfileConfig, prepared.region)
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
	cloudProfileConfig *apisvsphere.CloudProfileConfig,
	region *apisvsphere.RegionSpec,
) error {
	status, err := a.makeProviderInfrastructureStatus(newState, cloudProfileConfig, region)
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
	cloudProfileConfig *apisvsphere.CloudProfileConfig,
	region *apisvsphere.RegionSpec,
) (*apisvsphere.InfrastructureStatus, error) {
	safe := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

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
	status := &apisvsphere.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
		VsphereConfig: apisvsphere.VsphereConfig{
			Folder:      cloudProfileConfig.Folder,
			Region:      region.Name,
			ZoneConfigs: zoneConfigs,
		},
		NSXTInfraState: state,
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
