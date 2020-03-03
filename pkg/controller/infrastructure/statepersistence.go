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
	"encoding/json"
	"fmt"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	errors2 "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	infra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

const (
	versionKey     = "version"
	currentVersion = "1.0"
	stateKey       = "state"
)

func (a *actuator) loadStateFromConfigMap(ctx context.Context, namespace string) (*infra.NSXTInfraState, error) {
	obj := &v1.ConfigMap{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      vsphere.InfrastructureConfigMapName,
	}
	err := a.Client().Get(ctx, key, obj)
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	version := obj.Data[versionKey]
	stateJson := obj.Data[stateKey]
	if version != currentVersion {
		return nil, fmt.Errorf("unexpected version in state config map (%s): %s != %s", key, version, currentVersion)
	}
	if stateJson == "" {
		return nil, fmt.Errorf("state not found in config map (%s) : %s != %s", key, version, currentVersion)
	}
	state := &infra.NSXTInfraState{}
	err = json.Unmarshal([]byte(stateJson), state)
	if err != nil {
		return nil, errors2.Wrapf(err, "unmarshalling state config map (%s) failed", key)
	}
	return state, nil
}

func (a *actuator) saveStateToConfigMap(ctx context.Context, namespace string, state *infra.NSXTInfraState) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return errors2.Wrapf(err, "marshalling state failed")
	}
	stateJson := string(bytes)
	data := map[string]string{
		versionKey: currentVersion,
		stateKey:   stateJson,
	}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      vsphere.InfrastructureConfigMapName,
	}
	obj := &v1.ConfigMap{}
	err = a.Client().Get(ctx, key, obj)
	if errors.IsNotFound(err) {
		obj.Name = key.Name
		obj.Namespace = key.Namespace
		obj.Data = data
		return a.Client().Create(ctx, obj)
	}
	if err != nil {
		return err
	}
	return extensionscontroller.TryUpdate(ctx, retry.DefaultBackoff, a.Client(), obj, func() error {
		obj.Data = data
		return nil
	})
}

func (a *actuator) deleteStateFromConfigMap(ctx context.Context, namespace string) error {
	obj := &v1.ConfigMap{}
	obj.Name = vsphere.InfrastructureConfigMapName
	obj.Namespace = namespace
	return a.Client().Delete(ctx, obj)
}

func (a *actuator) logFailedSaveState(err error, state *infra.NSXTInfraState) {
	bytes, err2 := json.Marshal(state)
	stateString := ""
	if err2 == nil {
		stateString = string(bytes)
	} else {
		stateString = err2.Error()
	}
	a.logger.Error(err, "persisting infrastructure state failed", "state", stateString)
}
