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

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/cmd/infra-cli/loadbalancer"
)

func (a *actuator) delete(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	state, creationStarted, err := a.getInfrastructureState(infra)
	if err != nil {
		return err
	}
	if state == nil || creationStarted == nil || !*creationStarted {
		// no state or creation has not started (e.g. wrong credentials) => nothing to do
		return nil
	}

	prepared, err := a.prepareReconcile(ctx, log, infra, cluster)
	if err != nil {
		return err
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
		log.Info(fmt.Sprintf("warning: cleanup of load balancers failed with: %s", err), "infra", infra.Name)
	}

	err = prepared.ensurer.EnsureInfrastructureDeleted(&prepared.spec, state)
	errUpdate := a.updateProviderStatus(ctx, log, infra, state, prepared, creationStarted)
	if err != nil {
		return err
	}
	return errUpdate
}
