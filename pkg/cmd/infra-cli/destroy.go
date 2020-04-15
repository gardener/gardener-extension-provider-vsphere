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

package infra_cli

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/ensurer"
)

func DestroyInfrastructure(logger logr.Logger, cfg *infrastructure.NSXTConfig, stateString *string, specString *string) (*string, error) {
	if stateString == nil && specString == nil {
		return nil, fmt.Errorf("Either state or spec is needed to destroy infrastructure")
	}
	infrastructureEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(logger, cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "creating ensurer failed")
	}
	var state *vsphere.NSXTInfraState
	if stateString != nil {
		state = &vsphere.NSXTInfraState{}
		err = yaml.Unmarshal([]byte(*stateString), state)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshalling state failed")
		}
	} else {
		state, err = infrastructureEnsurer.NewStateWithVersion()
		if err != nil {
			return nil, errors.Wrapf(err, "NewStateWithVersion failed")
		}
	}
	var spec *infrastructure.NSXTInfraSpec
	if specString != nil {
		spec = &infrastructure.NSXTInfraSpec{}
		err = yaml.Unmarshal([]byte(*specString), spec)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshalling spec failed")
		}
	}
	err = infrastructureEnsurer.EnsureInfrastructureDeleted(spec, state)
	if err != nil {
		resultingState := showState(logger, state)
		return &resultingState, errors.Wrapf(err, "destroying infrastructure failed")
	}
	logger.Info("done")
	return nil, nil
}

func showState(logger logr.Logger, state *vsphere.NSXTInfraState) string {
	bytes, err := json.Marshal(state)
	if err != nil {
		logger.Info("Marshalling state failed", "err", err.Error())
		return ""
	}
	stateStr := string(bytes)
	logger.Info(stateStr)
	return stateStr
}
