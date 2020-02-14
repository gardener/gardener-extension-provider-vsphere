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

package infrastructure

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/log"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
)

type ensurer struct {
	logger logr.Logger

	// connector for simplified API (NSXT policy)
	connector vapiclient.Connector
	// NSX Manager client - based on go-vmware-nsxt SDK (Advanced API)
	nsxClient *nsxt.APIClient
	tasks     []task
}

func NewNSXTInfrastructureEnsurer(logger logr.Logger, nsxtConfig *NsxtConfig) (NSXTInfrastructureEnsurer, error) {
	log.SetLogger(NewLogrBridge(logger))
	connector, err := createConnector(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T connector failed")
	}
	nsxClient, err := createNSXClient(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T client failed")
	}

	tasks := []task{
		newLookupTier0GatewayTask(),
		newLookupTransportZone(),
		newLookupEdgeClusterTask(),
		newLookupSNATIPPoolTask(),
		newTier1GatewayTask(),
		newTier1GatewayLocaleServiceTask(),
		newSegmentTask(),
		newSNATIPAddressAllocationTask(),
		newSNATIPAddressRealizationTask(),
		newSNATRuleTask(),
		newAdvancedLookupEdgeClusterTask(),
		newAdvancedLookupLogicalSwitchTask(),
		newAdvancedDHCPProfileTask(),
		newAdvancedDHCPServerTask(),
		newAdvancedDHCPPortTask(),
		newAdvancedDHCPIPPoolTask(),
	}

	return &ensurer{
		logger:    logger,
		connector: connector,
		nsxClient: nsxClient,
		tasks:     tasks,
	}, nil
}

func (e *ensurer) EnsureInfrastructure(spec NSXTInfraSpec, state *NSXTInfraState) error {
	for _, task := range e.tasks {
		err := task.Ensure(e, spec, state)
		if err != nil {
			return errors.Wrapf(err, task.Label()+" failed")
		}
		keysAndVals := []interface{}{}
		name := task.name(spec)
		if name != nil {
			keysAndVals = append(keysAndVals, "name", *name)
		}
		ref := task.reference(state)
		if ref != nil {
			keysAndVals = append(keysAndVals, "id", ref.ID)
		}
		e.logger.Info(task.Label()+" ensured", keysAndVals...)
	}

	return nil
}

func (e *ensurer) EnsureInfrastructureDeleted(spec NSXTInfraSpec, state *NSXTInfraState) error {
	for i := len(e.tasks) - 1; i >= 0; i-- {
		task := e.tasks[i]
		deleted, err := task.EnsureDeleted(e, spec, state)
		if err != nil {
			return errors.Wrapf(err, "deleting "+task.Label()+" failed")
		}
		if deleted {
			e.logger.Info(task.Label() + " deleted")
		}
	}
	return nil
}
