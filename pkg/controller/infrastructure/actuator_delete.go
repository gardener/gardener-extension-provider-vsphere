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

package infrastructure

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/cmd/infra-cli/loadbalancer"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/withkubernetes"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *actuator) delete(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	state, creationStarted, err := a.getInfrastructureState(infra)
	if err != nil {
		return err
	}

	prepared, err2 := a.prepareReconcile(ctx, infra, cluster)
	if err2 != nil {
		return err2
	}
	if prepared.k8sClient != nil {
		if creationStarted != nil {
			return fmt.Errorf("invalid state: creationStarted for vsphere with kubernetes?")
		}
		return a.deleteK8s(ctx, prepared, infra, cluster)
	}

	if state == nil || creationStarted == nil || !*creationStarted {
		// no state or creation has not started (e.g. wrong credentials) => nothing to do
		return nil
	}

	// try to cleanup any possible left-offs from the cloud-provider-vsphere load balancer controller
	ipPoolName, err := prepared.getDefaultLoadBalancerIPPoolName()
	if err == nil {
		lbstate := &loadbalancer.DestroyState{
			ClusterName:       cluster.ObjectMeta.Name + "-" + a.gardenID,
			Owner:             a.gardenID,
			DefaultIPPoolName: *ipPoolName,
		}
		err = loadbalancer.DestroyAll(prepared.nsxtConfig, lbstate)
	}
	if err != nil {
		a.logger.Info(fmt.Sprintf("warning: cleanup of load balancers failed with: %s", err), "infra", infra.Name)
	}

	err = prepared.ensurer.EnsureInfrastructureDeleted(&prepared.spec, state)
	errUpdate := a.updateProviderStatus(ctx, infra, state, prepared, creationStarted)
	if err != nil {
		return err
	}
	return errUpdate
}

func (a *actuator) deleteK8s(ctx context.Context, prepared *preparedReconcile, _ *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	vwk := prepared.cloudProfileConfig.VsphereWithKubernetes
	namespace, createNamespace := withkubernetes.CalcSupervisorNamespace(cluster, vwk)

	err := a.deleteNetwork(ctx, prepared.k8sClient, ctrlClient.ObjectKey{Namespace: namespace, Name: cluster.ObjectMeta.Name})
	if err != nil {
		return err
	}

	if createNamespace {
		err := a.deleteCCMServiceAccount(ctx, prepared, namespace)
		if err != nil {
			return err
		}

		err = prepared.apiClient.DeleteNamespace(namespace)
		if err != nil {
			return fmt.Errorf("deletion of namespace %s failed: %s", namespace, err)
		}
	}

	return nil
}

func (a *actuator) deleteCCMServiceAccount(ctx context.Context, prepared *preparedReconcile, namespace string) error {
	chartPath := filepath.Join(vsphere.InternalChartsPath, "supervisor-service-account-ccm")
	return withkubernetes.DeleteServiceAccount(ctx, prepared.k8sRestConfig, chartPath, "", namespace)
}

func (a *actuator) deleteNetwork(ctx context.Context, client ctrlClient.Client, name ctrlClient.ObjectKey) error {
	err := withkubernetes.DeleteVirtualNetwork(ctx, client, name)
	if err != nil {
		return fmt.Errorf("deletion of virtual network %s failed: %s", name, err)
	}

	return nil
}
