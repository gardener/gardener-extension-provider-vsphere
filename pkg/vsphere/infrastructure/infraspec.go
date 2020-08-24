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
	"fmt"

	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

const (
	ScopeGarden = "garden"
	ScopeShoot  = "shoot"
	// ScopeAuthorizedShoots is the tag name used to annotate a NSX-T object with allowed shoot-names
	ScopeAuthorizedShoots = "authorized-shoots"
)

func (s NSXTInfraSpec) FullClusterName() string {
	return fmt.Sprintf("%s--%s", s.GardenName, s.ClusterName)
}

func (s NSXTInfraSpec) CreateCommonTags() []common.Tag {
	return []common.Tag{
		{Scope: ScopeGarden, Tag: s.GardenName},
		{Scope: ScopeShoot, Tag: s.ClusterName},
	}
}

func (s NSXTInfraSpec) CreateTags() []model.Tag {
	return []model.Tag{
		{Scope: strptr(ScopeGarden), Tag: strptr(s.GardenID)},
		{Scope: strptr(ScopeShoot), Tag: strptr(s.ClusterName)},
	}
}

func strptr(s string) *string {
	return &s
}
