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

package task

import (
	"github.com/go-logr/logr"
	nsxt "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/common"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

type EnsurerContext interface {
	Logger() logr.Logger
	// Connector for simplified API (NSXT policy)
	Connector() vapiclient.Connector
	// NSXTClient NSX Manager client - based on go-vmware-nsxt SDK (Advanced API)
	NSXTClient() *nsxt.APIClient
	// IsTryRecoverEnabled returns if NSX-T object should be searched by tag if no reference is set in state
	IsTryRecoverEnabled() bool
	// GetNSXTVersion retrieves the NSX-T version
	GetNSXTVersion() (*string, error)
	// ShootNamespace returns the shoot namespace used for authorization
	ShootNamespace() string
	// GardenID returns the garden cluster identity used for authorization
	GardenID() string
}

type Task interface {
	Label() string
	Ensure(ctx EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (action string, err error)
	EnsureDeleted(ctx EnsurerContext, state *api.NSXTInfraState) (deleted bool, err error)
	NameToLog(spec vinfra.NSXTInfraSpec) *string
	Reference(state *api.NSXTInfraState) *api.Reference
}

type RecoverableTask interface {
	ListAll(a EnsurerContext, state *api.NSXTInfraState, cursor *string) (interface{}, error)
	SetRecoveredReference(state *api.NSXTInfraState, ref *api.Reference, displayName *string)
	IsExternal(state *api.NSXTInfraState) bool
}

type RecoverableAdvancedTask interface {
	TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool
}
