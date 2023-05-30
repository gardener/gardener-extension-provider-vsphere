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
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	nsxt "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/log"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"k8s.io/apimachinery/pkg/util/sets"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"
)

type ensurer struct {
	logger logr.Logger
	// connector for simplified API (NSXT policy)
	connector vapiclient.Connector
	// nsxtClient is the NSX Manager client - based on go-vmware-nsxt SDK (Advanced API)
	nsxtClient     *nsxt.APIClient
	shootNamespace string
	gardenID       string
}

type ShootContext struct {
	ShootNamespace string
	GardenID       string
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

func (e *ensurer) GetNSXTVersion() (*string, error) {
	return getNSXTVersion(e.connector)
}

func (e *ensurer) IsTryRecoverEnabled() bool {
	return true
}

func (e *ensurer) ShootNamespace() string {
	return e.shootNamespace
}

func (e *ensurer) GardenID() string {
	return e.gardenID
}

func NewNSXTInfrastructureEnsurer(logger logr.Logger, nsxtConfig *vinfra.NSXTConfig, shootCtx *ShootContext) (vinfra.NSXTInfrastructureEnsurer, error) {
	log.SetLogger(NewLogrBridge(logger))
	connector, err := createConnectorNiceError(nsxtConfig)
	if err != nil {
		return nil, err
	}
	nsxClient, err := createNSXClient(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T client failed")
	}

	obj := &ensurer{
		logger:     logger,
		connector:  connector,
		nsxtClient: nsxClient,
	}
	if shootCtx != nil {
		obj.shootNamespace = shootCtx.ShootNamespace
		obj.gardenID = shootCtx.GardenID
	}
	return obj, nil
}

func (e *ensurer) NewStateWithVersion(overwriteVersion *string) (*api.NSXTInfraState, error) {
	if overwriteVersion != nil {
		if !sets.NewString(api.SupportedEnsurerVersions...).Has(*overwriteVersion) {
			return nil, fmt.Errorf("invalid overwrite version %s", *overwriteVersion)
		}
		return &api.NSXTInfraState{
			Version: overwriteVersion,
		}, nil
	}

	nsxtVersion, err := getNSXTVersion(e.connector)
	if err != nil {
		return nil, err
	}
	version := api.Ensurer_Version2_NSXT30
	if strings.HasPrefix(*nsxtVersion, "2.") {
		version = api.Ensurer_Version1_NSXT25
	}
	state := &api.NSXTInfraState{
		Version: &version,
	}
	return state, nil
}

func (e *ensurer) CheckConnection() error {
	client := infra.NewTier0sClient(e.connector)
	_, err := client.List(nil, nil, nil, nil, nil, nil)
	return err
}

func (e *ensurer) getTasks(state *api.NSXTInfraState) []task.Task {
	nsxt3 := state.Version != nil && *state.Version != api.Ensurer_Version1_NSXT25
	tasks := []task.Task{
		task.NewLookupTier0GatewayTask(),
		task.NewLookupTransportZoneTask(),
		task.NewLookupEdgeClusterTask(),
		task.NewLookupSNATIPPoolTask(),
		task.NewTier1GatewayTask(),
		task.NewTier1GatewayLocaleServiceTask(),
	}
	if nsxt3 {
		tasks = append(tasks,
			task.NewDHCPServerConfigTask(),
		)
	}
	tasks = append(tasks,
		task.NewSegmentTask(),
		task.NewSNATIPAddressAllocationTask(),
		task.NewSNATIPAddressRealizationTask(),
		task.NewSNATRuleTask(),
	)
	if !nsxt3 {
		tasks = append(tasks,
			task.NewAdvancedLookupLogicalSwitchTask(),
			task.NewAdvancedDHCPProfileTask(),
			task.NewAdvancedDHCPServerTask(),
			task.NewAdvancedDHCPPortTask(),
			task.NewAdvancedDHCPIPPoolTask(),
		)
	}
	return tasks
}

func (e *ensurer) EnsureInfrastructure(spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) error {
	e.updateExternalState(spec, state)
	tasks := e.getTasks(state)
	for _, tsk := range tasks {
		_ = e.tryRecover(spec, state, tsk, false)

		action, err := tsk.Ensure(e, spec, state)
		if err != nil {
			return errors.Wrapf(err, tsk.Label()+" failed")
		}
		keysAndVals := []interface{}{}
		name := tsk.NameToLog(spec)
		if name != nil {
			keysAndVals = append(keysAndVals, "name", *name)
		}
		ref := tsk.Reference(state)
		if ref != nil {
			keysAndVals = append(keysAndVals, "id", ref.ID)
		}
		e.logger.Info(fmt.Sprintf("%s %s", tsk.Label(), action), keysAndVals...)
	}

	return nil
}

func (e *ensurer) updateExternalState(spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) {
	if spec.ExternalTier1GatewayPath != nil {
		b := true
		state.ExternalTier1Gateway = &b
	}
}

// tryRecover tries if the NSX-T reference has for some reason been lost and not be stored in the state.
// It then tries to find the object by the garden and shoot tag to restore the reference.
func (e *ensurer) tryRecover(spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState, tsk task.Task, lookup bool) error {
	if e.IsTryRecoverEnabled() && tsk.Reference(state) == nil {
		recovered := false
		if rt, ok := tsk.(task.RecoverableTask); ok {
			if rt.IsExternal(state) {
				if lookup {
					// external lookup may be needed for recover
					var err error
					_, err = tsk.Ensure(e, spec, state)
					return err
				}
				return nil
			}
			recovered = task.TryRecover(e, state, rt, spec.CreateTags())
		} else if rt, ok := tsk.(task.RecoverableAdvancedTask); ok {
			recovered = rt.TryRecover(e, state, spec.CreateCommonTags())
		} else if lookup {
			// not recoverable tasks are lookup tasks which may be needed for recover
			var err error
			_, err = tsk.Ensure(e, spec, state)
			return err
		}
		if recovered {
			e.logger.Info(fmt.Sprintf("%s state recovered", tsk.Label()))
		}
	}
	return nil
}

func (e *ensurer) EnsureInfrastructureDeleted(spec *vinfra.NSXTInfraSpec, state *api.NSXTInfraState) error {
	tasks := e.getTasks(state)
	if spec != nil {
		e.updateExternalState(*spec, state)
		// tryRecover needs the order of creation
		for _, tsk := range tasks {
			err := e.tryRecover(*spec, state, tsk, true)
			if err != nil {
				keysAndVals := []interface{}{}
				name := tsk.NameToLog(*spec)
				if name != nil {
					keysAndVals = append(keysAndVals, "name", *name)
				}
				e.logger.Info("try recover failed", keysAndVals...)
			}
		}
	}

	for i := len(tasks) - 1; i >= 0; i-- {
		tsk := tasks[i]

		deleted, err := tsk.EnsureDeleted(e, state)
		if err != nil {
			return errors.Wrapf(err, "deleting "+tsk.Label()+" failed")
		}
		if deleted {
			e.logger.Info(tsk.Label() + " deleted")
		}
	}
	return nil
}

func (e *ensurer) GetIPPoolTags(ipPoolName string) (map[string]string, error) {
	id, _, err := task.LookupIPPoolIDByName(e, ipPoolName)
	if err != nil {
		return nil, err
	}

	client := infra.NewIpPoolsClient(e.Connector())
	pool, err := client.Get(id)
	if err != nil {
		return nil, err
	}
	tags := task.TagsToMap(pool.Tags)
	return tags, nil
}

func (e *ensurer) CheckShootAuthorizationByTags(objectType, name string, tags map[string]string) error {
	return task.CheckShootAuthorizationByTags(e.logger, objectType, name, e.shootNamespace, e.gardenID, tags)
}
