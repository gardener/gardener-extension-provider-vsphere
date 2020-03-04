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

package ensurer

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/log"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"
)

type ensurer struct {
	logger logr.Logger
	// connector for simplified API (NSXT policy)
	connector vapiclient.Connector
	// nsxtClient is the NSX Manager client - based on go-vmware-nsxt SDK (Advanced API)
	nsxtClient *nsxt.APIClient
	tasks      []task.Task
}

var _ task.EnsurerContext = &ensurer{}

func (e *ensurer) Logger() logr.Logger {
	return e.logger
}

func (e *ensurer) Connector() vapiclient.Connector {
	return e.connector
}

func (e *ensurer) NSXTClient() *nsxt.APIClient {
	return e.nsxtClient
}

func (e *ensurer) TryRecover() bool {
	return true
}

func NewNSXTInfrastructureEnsurer(logger logr.Logger, nsxtConfig *vinfra.NSXTConfig) (vinfra.NSXTInfrastructureEnsurer, error) {
	log.SetLogger(NewLogrBridge(logger))
	connector, err := createConnector(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T connector failed")
	}
	nsxClient, err := createNSXClient(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T client failed")
	}

	tasks := []task.Task{
		task.NewLookupTier0GatewayTask(),
		task.NewLookupTransportZoneTask(),
		task.NewLookupEdgeClusterTask(),
		task.NewLookupSNATIPPoolTask(),
		task.NewTier1GatewayTask(),
		task.NewTier1GatewayLocaleServiceTask(),
		task.NewSegmentTask(),
		task.NewSNATIPAddressAllocationTask(),
		task.NewSNATIPAddressRealizationTask(),
		task.NewSNATRuleTask(),
		task.NewAdvancedLookupLogicalSwitchTask(),
		task.NewAdvancedDHCPProfileTask(),
		task.NewAdvancedDHCPServerTask(),
		task.NewAdvancedDHCPPortTask(),
		task.NewAdvancedDHCPIPPoolTask(),
	}

	return &ensurer{
		logger:     logger,
		connector:  connector,
		nsxtClient: nsxClient,
		tasks:      tasks,
	}, nil
}

func (e *ensurer) EnsureInfrastructure(spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) error {
	for _, task := range e.tasks {
		action, err := task.Ensure(e, spec, state)
		if err != nil {
			return errors.Wrapf(err, task.Label()+" failed")
		}
		keysAndVals := []interface{}{}
		name := task.Name(spec)
		if name != nil {
			keysAndVals = append(keysAndVals, "name", *name)
		}
		ref := task.Reference(state)
		if ref != nil {
			keysAndVals = append(keysAndVals, "id", ref.ID)
		}
		e.logger.Info(fmt.Sprintf("%s %s", task.Label(), action), keysAndVals...)
	}

	return nil
}

func (e *ensurer) EnsureInfrastructureDeleted(state *api.NSXTInfraState) error {
	for i := len(e.tasks) - 1; i >= 0; i-- {
		task := e.tasks[i]
		deleted, err := task.EnsureDeleted(e, state)
		if err != nil {
			return errors.Wrapf(err, "deleting "+task.Label()+" failed")
		}
		if deleted {
			e.logger.Info(task.Label() + " deleted")
		}
	}
	return nil
}
