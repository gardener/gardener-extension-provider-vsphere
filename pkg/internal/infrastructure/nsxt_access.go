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
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/manager"
	"github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std"
	vapierrors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/ip_pools"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/realized_state"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/sites/enforcement_points"
	t1nat "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/tier_1s/nat"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

const (
	description            = "created by gardener-extension-provider-vsphere"
	defaultSite            = "default"
	policyEnforcementPoint = "default"
)

type nsxtAccess struct {
	logger logr.Logger

	// connector for simplified API (NSXT policy)
	connector vapiclient.Connector
	// NSX Manager client - based on go-vmware-nsxt SDK (Advanced API)
	nsxClient *nsxt.APIClient
	actuators []infraActuator
}

type creatorFunc func(spec NSXTInfraSpec, state *NSXTInfraState) error
type deletorFunc func(spec NSXTInfraSpec, state *NSXTInfraState) (deleted bool, err error)
type nameFunc func(spec NSXTInfraSpec) string
type refFunc func(state *NSXTInfraState) *Reference

type infraActuator struct {
	name     string
	creator  creatorFunc
	deletor  deletorFunc
	nameInfo nameFunc
	refInfo  refFunc
}

func NewNSXTAccess(logger logr.Logger, nsxtConfig *NsxtConfig) (*nsxtAccess, error) {
	connector, err := createConnector(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T connector failed")
	}
	nsxClient, err := createNSXClient(nsxtConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating NSX-T client failed")
	}
	access := &nsxtAccess{
		logger:    logger,
		connector: connector,
		nsxClient: nsxClient,
	}

	toReference := func(s *string) *Reference {
		if s == nil {
			return nil
		}
		return &Reference{ID: *s}
	}

	access.actuators = []infraActuator{
		{
			name:     "tier-0 gateway lookup",
			creator:  access.lookupTier0GatewayRefByName,
			nameInfo: func(spec NSXTInfraSpec) string { return spec.Tier0GatewayName },
			refInfo:  func(state *NSXTInfraState) *Reference { return state.Tier0GatewayRef },
		},
		{
			name:     "transport zone lookup",
			creator:  access.lookupTransportZoneRefByName,
			nameInfo: func(spec NSXTInfraSpec) string { return spec.TransportZoneName },
			refInfo:  func(state *NSXTInfraState) *Reference { return state.TransportZoneRef },
		},
		{
			name:     "edge cluster lookup",
			creator:  access.lookupEdgeClusterRefByName,
			nameInfo: func(spec NSXTInfraSpec) string { return spec.EdgeClusterName },
			refInfo:  func(state *NSXTInfraState) *Reference { return state.EdgeClusterRef },
		},
		{
			name:    "tier-1 gateway",
			creator: access.ensureTier1Gateway,
			deletor: access.ensureTier1GatewayDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return state.Tier1GatewayRef },
		},
		{
			name:    "segment",
			creator: access.ensureSegment,
			deletor: access.ensureSegmentDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return state.SegmentRef },
		},
		{
			name:    "SNAT IP address allocation",
			creator: access.ensureSNATIPAddressAllocation,
			deletor: access.ensureSNATIPAddressAllocationDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return state.SNATIPAddressAllocRef },
		},
		{
			name:    "SNAT IP address realization",
			creator: access.ensureSNATIPAddressRealized,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.SNATIPAddress) },
		},
		{
			name:    "SNAT rule",
			creator: access.ensureSNATRule,
			deletor: access.ensureSNATRuleDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return state.SNATRuleRef },
		},
		{
			name:     "edge cluster lookup (Advanced API)",
			creator:  access.advancedLookupEdgeClusterIdByName,
			nameInfo: func(spec NSXTInfraSpec) string { return spec.EdgeClusterName },
			refInfo:  func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.EdgeClusterID) },
		},
		{
			name:    "logical switch lookup (Advanced API)",
			creator: access.advancedLockupLogicalSwitchID,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.LogicalSwitchID) },
		},
		{
			name:    "DHCP profile (Advanced API)",
			creator: access.advancedEnsureDHCPProfile,
			deletor: access.advancedEnsureDHCPProfileDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.ProfileID) },
		},
		{
			name:    "DHCP server (Advanced API)",
			creator: access.advancedEnsureDHCPServer,
			deletor: access.advancedEnsureDHCPServerDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.ServerID) },
		},
		{
			name:    "DHCP port (Advanced API)",
			creator: access.advancedEnsureDHCPPort,
			deletor: access.advancedEnsureDHCPPortDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.PortID) },
		},
		{
			name:    "DHCP IP pool (Advanced API)",
			creator: access.advancedEnsureDHCPIPPool,
			deletor: access.advancedEnsureDHCPIPPoolDeleted,
			refInfo: func(state *NSXTInfraState) *Reference { return toReference(state.AdvancedDHCP.IPPoolID) },
		},
	}

	return access, nil
}

func (a *nsxtAccess) EnsureInfrastructure(spec NSXTInfraSpec, state *NSXTInfraState) error {
	for _, actuator := range a.actuators {
		if actuator.creator != nil {
			err := actuator.creator(spec, state)
			if err != nil {
				return errors.Wrapf(err, actuator.name+" failed")
			}
			keysAndVals := []interface{}{}
			if actuator.nameInfo != nil {
				keysAndVals = append(keysAndVals, "name", actuator.nameInfo(spec))
			}
			if actuator.refInfo != nil {
				keysAndVals = append(keysAndVals, "id", actuator.refInfo(state))
			}
			a.logger.Info(actuator.name+" ensured", keysAndVals...)
		}
	}

	return nil
}

func (a *nsxtAccess) EnsureInfrastructureDeleted(spec NSXTInfraSpec, state *NSXTInfraState) error {
	for i := len(a.actuators) - 1; i >= 0; i-- {
		actuator := a.actuators[i]
		if actuator.deletor != nil {
			deleted, err := actuator.deletor(spec, state)
			if err != nil {
				return errors.Wrapf(err, "deleting "+actuator.name+" failed")
			}
			if deleted {
				a.logger.Info(actuator.name + " deleted")
			}
		}
	}
	return nil
}

func (a *nsxtAccess) lookupTier0GatewayRefByName(spec NSXTInfraSpec, state *NSXTInfraState) error {
	name := spec.Tier0GatewayName
	client := infra.NewDefaultTier0sClient(a.connector)
	var cursor *string
	total := 0
	count := 0
	for {
		result, err := client.List(cursor, nil, nil, nil, nil, nil)
		if err != nil {
			return nicerVAPIError(err)
		}
		for _, item := range result.Results {
			if *item.DisplayName == name {
				// found
				state.Tier0GatewayRef = &Reference{ID: *item.Id, Path: *item.Path}
				return nil
			}
		}
		if cursor == nil {
			total = int(*result.ResultCount)
		}
		count += len(result.Results)
		if count >= total {
			return fmt.Errorf("not found: %s", name)
		}
		cursor = result.Cursor
	}
}

func (a *nsxtAccess) lookupEdgeClusterRefByName(spec NSXTInfraSpec, state *NSXTInfraState) error {
	name := spec.EdgeClusterName
	client := enforcement_points.NewDefaultEdgeClustersClient(a.connector)
	result, err := client.List(defaultSite, policyEnforcementPoint, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nicerVAPIError(err)
	}
	for _, item := range result.Results {
		if *item.DisplayName == name {
			state.EdgeClusterRef = &Reference{ID: *item.Id, Path: *item.Path}
			return nil
		}
	}
	return fmt.Errorf("not found: %s", name)
}

func (a *nsxtAccess) lookupTransportZoneRefByName(spec NSXTInfraSpec, state *NSXTInfraState) error {
	name := spec.TransportZoneName
	client := enforcement_points.NewDefaultTransportZonesClient(a.connector)
	result, err := client.List(defaultSite, policyEnforcementPoint, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nicerVAPIError(err)
	}
	for _, item := range result.Results {
		if *item.DisplayName == name {
			state.TransportZoneRef = &Reference{ID: *item.Id, Path: *item.Path}
			return nil
		}
	}
	return fmt.Errorf("not found: %s", name)
}

func (a *nsxtAccess) lookupSNATIPPoolRefByName(spec NSXTInfraSpec, state *NSXTInfraState) error {
	name := spec.SNATIPPoolName
	client := enforcement_points.NewDefaultTransportZonesClient(a.connector)
	result, err := client.List(defaultSite, policyEnforcementPoint, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nicerVAPIError(err)
	}
	for _, item := range result.Results {
		if *item.DisplayName == name {
			state.SNATIPPoolRef = &Reference{ID: *item.Id, Path: *item.Path}
			return nil
		}
	}
	return fmt.Errorf("not found: %s", name)
}

func (a *nsxtAccess) ensureTier1Gateway(spec NSXTInfraSpec, state *NSXTInfraState) error {
	client := infra.NewDefaultTier1sClient(a.connector)

	tier1 := model.Tier1{
		DisplayName:             strptr(spec.FullClusterName()),
		Description:             strptr(description),
		FailoverMode:            strptr("PREEMPTIVE"),
		Tags:                    spec.createTags(),
		RouteAdvertisementTypes: []string{"TIER1_STATIC_ROUTES", "TIER1_NAT", "TIER1_LB_VIP", "TIER1_LB_SNAT"},
		Tier0Path:               &state.Tier0GatewayRef.Path,
	}

	if state.Tier1GatewayRef != nil {
		oldTier1, err := client.Get(state.Tier1GatewayRef.ID)
		if isNotFoundError(err) {
			state.Tier1GatewayRef = nil
			return a.ensureTier1Gateway(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldTier1.DisplayName != tier1.DisplayName ||
			oldTier1.FailoverMode != tier1.FailoverMode ||
			oldTier1.Tier0Path == nil ||
			*oldTier1.Tier0Path != *tier1.Tier0Path ||
			!equalStrings(oldTier1.RouteAdvertisementTypes, tier1.RouteAdvertisementTypes) ||
			!equalTags(oldTier1.Tags, tier1.Tags) {
			_, err := client.Update(state.Tier1GatewayRef.ID, tier1)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
		}
		return nil
	}

	id := a.generateID()
	createdObj, err := client.Update(id, tier1)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	state.Tier1GatewayRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return nil
}

func (a *nsxtAccess) ensureTier1GatewayDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	client := infra.NewDefaultTier1sClient(a.connector)
	if state.Tier1GatewayRef == nil {
		return false, nil
	}
	err := client.Delete(state.Tier1GatewayRef.ID)
	if err != nil {
		return false, err
	}
	state.Tier1GatewayRef = nil
	return true, nil
}

func (a *nsxtAccess) ensureSegment(spec NSXTInfraSpec, state *NSXTInfraState) error {
	client := infra.NewDefaultSegmentsClient(a.connector)

	_, network, err := net.ParseCIDR(spec.WorkersNetwork)
	if err != nil {
		return errors.Wrapf(err, "Parsing workers CIDR %s failed", spec.WorkersNetwork)
	}
	subnet := model.SegmentSubnet{
		GatewayAddress: strptr(network.String()),
	}
	segment := model.Segment{
		DisplayName:       strptr(spec.FullClusterName()),
		Description:       strptr(description),
		ConnectivityPath:  strptr(state.Tier1GatewayRef.Path),
		TransportZonePath: strptr(state.TransportZoneRef.Path),
		Tags:              spec.createTags(),
		Subnets:           []model.SegmentSubnet{subnet},
	}

	if state.SegmentRef != nil {
		oldSegment, err := client.Get(state.SegmentRef.ID)
		if isNotFoundError(err) {
			state.SegmentRef = nil
			return a.ensureTier1Gateway(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldSegment.DisplayName != segment.DisplayName ||
			oldSegment.ConnectivityPath == nil ||
			*oldSegment.ConnectivityPath != *segment.ConnectivityPath ||
			oldSegment.TransportZonePath == nil ||
			*oldSegment.TransportZonePath != *segment.TransportZonePath ||
			len(oldSegment.Subnets) != 1 ||
			oldSegment.Subnets[0].GatewayAddress == nil ||
			*oldSegment.Subnets[0].GatewayAddress != *segment.Subnets[0].GatewayAddress ||
			!equalTags(oldSegment.Tags, segment.Tags) {
			_, err := client.Update(state.SegmentRef.ID, segment)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
		}
		return nil
	}

	id := a.generateID()
	createdObj, err := client.Update(id, segment)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	state.SegmentRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return nil
}

func (a *nsxtAccess) ensureSegmentDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	client := infra.NewDefaultSegmentsClient(a.connector)
	if state.SegmentRef == nil {
		return false, nil
	}
	err := client.Delete(state.SegmentRef.ID)
	if err != nil {
		return false, err
	}
	state.SegmentRef = nil
	return true, nil
}

func (a *nsxtAccess) ensureSNATIPAddressAllocation(spec NSXTInfraSpec, state *NSXTInfraState) error {
	client := ip_pools.NewDefaultIpAllocationsClient(a.connector)

	allocation := model.IpAddressAllocation{
		DisplayName: strptr(spec.FullClusterName() + "_SNAT"),
		Description: strptr("SNAT IP address for all nodes. " + description),
		Tags:        spec.createTags(),
	}

	if state.SNATIPAddressAllocRef != nil {
		_, err := client.Get(state.SNATIPPoolRef.ID, state.SNATIPAddressAllocRef.ID)
		if err == nil {
			// IP address allocation is never updated
			return nil
		}
		if !isNotFoundError(err) {
			return errors.Wrapf(err, "reading")
		}
	}

	id := a.generateID()
	createdObj, err := client.Update(state.SNATIPPoolRef.ID, id, allocation)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	state.SNATIPPoolRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return nil
}

func (a *nsxtAccess) ensureSNATIPAddressAllocationDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	client := ip_pools.NewDefaultIpAllocationsClient(a.connector)
	if state.SNATIPAddressAllocRef == nil {
		return false, nil
	}
	err := client.Delete(state.SNATIPPoolRef.ID, state.SNATIPAddressAllocRef.ID)
	if err != nil {
		return false, err
	}
	state.SNATIPAddressAllocRef = nil
	return true, nil
}

func (a *nsxtAccess) ensureSNATIPAddressRealized(_ NSXTInfraSpec, state *NSXTInfraState) error {
	ipAddress, err := a.getRealizedIPAddress(state.SNATIPAddressAllocRef.Path, 15*time.Second)
	if err != nil {
		return err
	}
	state.SNATIPAddress = ipAddress
	return nil
}

func (a *nsxtAccess) ensureSNATRule(spec NSXTInfraSpec, state *NSXTInfraState) error {
	client := t1nat.NewDefaultNatRulesClient(a.connector)

	rule := model.PolicyNatRule{
		DisplayName:       strptr(spec.FullClusterName()),
		Description:       strptr(description),
		Action:            "SNAT",
		Enabled:           boolptr(true),
		Logging:           boolptr(true),
		Tags:              spec.createTags(),
		SourceNetwork:     strptr(spec.WorkersNetwork),
		TranslatedNetwork: strptr(fmt.Sprintf("%s/32", *state.SNATIPAddress)),
	}

	if state.SNATRuleRef != nil {
		oldRule, err := client.Get(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID)
		if isNotFoundError(err) {
			state.SegmentRef = nil
			return a.ensureTier1Gateway(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldRule.DisplayName != rule.DisplayName ||
			oldRule.Action != rule.Action ||
			oldRule.Enabled == nil ||
			*oldRule.Enabled != *rule.Enabled ||
			oldRule.Logging == nil ||
			*oldRule.Logging != *rule.Logging ||
			oldRule.SourceNetwork == nil ||
			*oldRule.SourceNetwork != *rule.SourceNetwork ||
			oldRule.TranslatedNetwork == nil ||
			*oldRule.TranslatedNetwork != *rule.TranslatedNetwork ||
			!equalTags(oldRule.Tags, rule.Tags) {
			_, err := client.Update(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID, rule)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
		}
		return nil
	}

	id := a.generateID()
	createdObj, err := client.Update(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, id, rule)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	state.SNATRuleRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return nil
}

func (a *nsxtAccess) ensureSNATRuleDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	client := t1nat.NewDefaultNatRulesClient(a.connector)
	if state.SNATRuleRef == nil {
		return false, nil
	}
	err := client.Delete(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID)
	if err != nil {
		return false, err
	}
	state.SNATRuleRef = nil
	return true, nil
}

func (a *nsxtAccess) advancedLookupEdgeClusterIdByName(spec NSXTInfraSpec, state *NSXTInfraState) error {
	name := spec.EdgeClusterName
	objList, _, err := a.nsxClient.NetworkTransportApi.ListEdgeClusters(a.nsxClient.Context, nil)
	if err != nil {
		return err
	}
	for _, obj := range objList.Results {
		if obj.DisplayName == name {
			state.AdvancedDHCP.EdgeClusterID = &obj.Id
			return nil
		}
	}
	return fmt.Errorf("not found: %s", name)
}

func (a *nsxtAccess) advancedEnsureDHCPProfile(spec NSXTInfraSpec, state *NSXTInfraState) error {
	profile := manager.DhcpProfile{
		DisplayName:   spec.FullClusterName(),
		Description:   description,
		EdgeClusterId: *state.AdvancedDHCP.EdgeClusterID,
		Tags:          spec.createCommonTags(),
	}

	if state.AdvancedDHCP.ProfileID != nil {
		oldProfile, resp, err := a.nsxClient.ServicesApi.ReadDhcpProfile(a.nsxClient.Context, *state.AdvancedDHCP.ProfileID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ProfileID = nil
			return a.advancedEnsureDHCPProfile(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldProfile.DisplayName != profile.DisplayName ||
			oldProfile.EdgeClusterId != profile.EdgeClusterId ||
			!equalCommonTags(oldProfile.Tags, profile.Tags) {
			_, resp, err := a.nsxClient.ServicesApi.UpdateDhcpProfile(a.nsxClient.Context, *state.AdvancedDHCP.ProfileID, profile)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("updating failed with unexpected HTTP status code %d", resp.StatusCode)
			}
		}
		return nil
	}

	createdProfile, resp, err := a.nsxClient.ServicesApi.CreateDhcpProfile(a.nsxClient.Context, profile)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("creating failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	state.AdvancedDHCP.ProfileID = &createdProfile.Id
	return nil
}

func (a *nsxtAccess) advancedEnsureDHCPProfileDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ProfileID == nil {
		return false, nil
	}
	resp, err := a.nsxClient.ServicesApi.DeleteDhcpProfile(a.nsxClient.Context, *state.AdvancedDHCP.ProfileID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		state.AdvancedDHCP.ProfileID = nil
		return false, nil
	}
	if err != nil {
		return false, err
	}
	state.AdvancedDHCP.ProfileID = nil
	return true, nil
}

func (a *nsxtAccess) advancedEnsureDHCPServer(spec NSXTInfraSpec, state *NSXTInfraState) error {
	dhcpServerIP, err := cidrHost(spec.WorkersNetwork, 2)
	if err != nil {
		return errors.Wrapf(err, "DHCP server IP")
	}
	gatewayIP, err := cidrHost(spec.WorkersNetwork, 1)
	if err != nil {
		return errors.Wrapf(err, "Gateway IP")
	}
	ipv4DhcpServer := manager.IPv4DhcpServer{
		DhcpServerIp:   dhcpServerIP,
		DnsNameservers: spec.DNSServers,
		GatewayIp:      gatewayIP,
	}

	server := manager.LogicalDhcpServer{
		Description:    description,
		DisplayName:    spec.FullClusterName(),
		Tags:           spec.createCommonTags(),
		DhcpProfileId:  *state.AdvancedDHCP.ProfileID,
		Ipv4DhcpServer: &ipv4DhcpServer,
	}

	if state.AdvancedDHCP.ServerID != nil {
		oldServer, resp, err := a.nsxClient.ServicesApi.ReadDhcpServer(a.nsxClient.Context, *state.AdvancedDHCP.ServerID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ServerID = nil
			return a.advancedEnsureDHCPServer(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldServer.DisplayName != server.DisplayName ||
			oldServer.DhcpProfileId != server.DhcpProfileId ||
			oldServer.Ipv4DhcpServer == nil ||
			oldServer.Ipv4DhcpServer.DhcpServerIp != server.Ipv4DhcpServer.DhcpServerIp ||
			oldServer.Ipv4DhcpServer.GatewayIp != server.Ipv4DhcpServer.GatewayIp ||
			!equalOrderedStrings(oldServer.Ipv4DhcpServer.DnsNameservers, server.Ipv4DhcpServer.DnsNameservers) ||
			!equalCommonTags(oldServer.Tags, server.Tags) {
			_, resp, err := a.nsxClient.ServicesApi.UpdateDhcpServer(a.nsxClient.Context, *state.AdvancedDHCP.ServerID, server)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("updating failed with unexpected HTTP status code %d", resp.StatusCode)
			}
		}
		return nil
	}

	createdServer, resp, err := a.nsxClient.ServicesApi.CreateDhcpServer(a.nsxClient.Context, server)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("creating failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	state.AdvancedDHCP.ServerID = &createdServer.Id
	return nil
}

func (a *nsxtAccess) advancedEnsureDHCPServerDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ServerID == nil {
		return false, nil
	}
	resp, err := a.nsxClient.ServicesApi.DeleteDhcpServer(a.nsxClient.Context, *state.AdvancedDHCP.ServerID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		state.AdvancedDHCP.ServerID = nil
		return false, nil
	}
	if err != nil {
		return false, err
	}
	state.AdvancedDHCP.ServerID = nil
	return true, nil
}

func (a *nsxtAccess) advancedLockupLogicalSwitchID(_ NSXTInfraSpec, state *NSXTInfraState) error {
	if state.SegmentRef == nil {
		return fmt.Errorf("missing segmentIdents")
	}
	result, resp, err := a.nsxClient.LogicalSwitchingApi.ListLogicalSwitches(a.nsxClient.Context, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("listing failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	for _, obj := range result.Results {
		for _, tag := range obj.Tags {
			if tag.Scope == "policyPath" && tag.Tag == state.SegmentRef.Path {
				state.AdvancedDHCP.LogicalSwitchID = &obj.Id
				return nil
			}
		}
	}
	return fmt.Errorf("not found by segment path %s", state.SegmentRef.Path)
}

func (a *nsxtAccess) advancedEnsureDHCPPort(spec NSXTInfraSpec, state *NSXTInfraState) error {
	attachment := manager.LogicalPortAttachment{
		AttachmentType: "DHCP_SERVICE",
		Id:             *state.AdvancedDHCP.ServerID,
	}
	port := manager.LogicalPort{
		DisplayName:     spec.FullClusterName(),
		Description:     description,
		LogicalSwitchId: *state.AdvancedDHCP.LogicalSwitchID,
		AdminState:      "UP",
		Tags:            spec.createCommonTags(),
		Attachment:      &attachment,
	}

	if state.AdvancedDHCP.PortID != nil {
		oldPort, resp, err := a.nsxClient.LogicalSwitchingApi.GetLogicalPort(a.nsxClient.Context, *state.AdvancedDHCP.PortID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.PortID = nil
			return a.advancedEnsureDHCPPort(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldPort.DisplayName != port.DisplayName ||
			oldPort.LogicalSwitchId != port.LogicalSwitchId ||
			oldPort.AdminState != port.AdminState ||
			oldPort.Attachment == nil ||
			oldPort.Attachment.AttachmentType != port.Attachment.AttachmentType ||
			oldPort.Attachment.Id != port.Attachment.Id ||
			!equalCommonTags(oldPort.Tags, port.Tags) {
			_, resp, err := a.nsxClient.LogicalSwitchingApi.UpdateLogicalPort(a.nsxClient.Context, *state.AdvancedDHCP.PortID, port)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("updating failed with unexpected HTTP status code %d", resp.StatusCode)
			}
		}
		return nil
	}

	createdPort, resp, err := a.nsxClient.LogicalSwitchingApi.CreateLogicalPort(a.nsxClient.Context, port)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("creating failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	state.AdvancedDHCP.PortID = &createdPort.Id
	return nil
}

func (a *nsxtAccess) advancedEnsureDHCPPortDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.PortID == nil {
		return false, nil
	}
	resp, err := a.nsxClient.LogicalSwitchingApi.DeleteLogicalPort(a.nsxClient.Context, *state.AdvancedDHCP.PortID, nil)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		state.AdvancedDHCP.PortID = nil
		return false, nil
	}
	if err != nil {
		return false, err
	}
	state.AdvancedDHCP.PortID = nil
	return true, nil
}

func (a *nsxtAccess) advancedEnsureDHCPIPPool(spec NSXTInfraSpec, state *NSXTInfraState) error {
	gatewayIP, err := cidrHost(spec.WorkersNetwork, 1)
	if err != nil {
		return errors.Wrapf(err, "Gateway IP")
	}
	startIP, err := cidrHost(spec.WorkersNetwork, 10)
	if err != nil {
		return errors.Wrapf(err, "Start IP of pool")
	}
	endIP, err := cidrHost(spec.WorkersNetwork, -1)
	if err != nil {
		return errors.Wrapf(err, "End IP of pool")
	}
	ipPoolRange := manager.IpPoolRange{
		Start: startIP,
		End:   endIP,
	}
	pool := manager.DhcpIpPool{
		DisplayName:      spec.FullClusterName(),
		Description:      description,
		GatewayIp:        gatewayIP,
		LeaseTime:        7200,
		ErrorThreshold:   98,
		WarningThreshold: 70,
		AllocationRanges: []manager.IpPoolRange{ipPoolRange},
		Tags:             spec.createCommonTags(),
	}

	if state.AdvancedDHCP.IPPoolID != nil {
		oldPool, resp, err := a.nsxClient.ServicesApi.ReadDhcpIpPool(a.nsxClient.Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.IPPoolID = nil
			return a.advancedEnsureDHCPIPPool(spec, state)
		}
		if err != nil {
			return errors.Wrapf(err, "reading")
		}
		if oldPool.DisplayName != pool.DisplayName ||
			oldPool.GatewayIp != pool.GatewayIp ||
			oldPool.LeaseTime != pool.LeaseTime ||
			oldPool.ErrorThreshold == pool.ErrorThreshold ||
			oldPool.WarningThreshold == pool.WarningThreshold ||
			len(oldPool.AllocationRanges) != 1 ||
			oldPool.AllocationRanges[0].Start != pool.AllocationRanges[0].Start ||
			oldPool.AllocationRanges[0].End != pool.AllocationRanges[0].End ||
			!equalCommonTags(oldPool.Tags, pool.Tags) {
			_, resp, err := a.nsxClient.ServicesApi.UpdateDhcpIpPool(a.nsxClient.Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID, pool)
			if err != nil {
				return errors.Wrapf(err, "updating")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("updating failed with unexpected HTTP status code %d", resp.StatusCode)
			}
		}
		return nil
	}

	createdPool, resp, err := a.nsxClient.ServicesApi.CreateDhcpIpPool(a.nsxClient.Context, *state.AdvancedDHCP.ServerID, pool)
	if err != nil {
		return errors.Wrapf(err, "creating")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("creating failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	state.AdvancedDHCP.IPPoolID = &createdPool.Id
	return nil
}

func (a *nsxtAccess) advancedEnsureDHCPIPPoolDeleted(_ NSXTInfraSpec, state *NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.IPPoolID == nil {
		return false, nil
	}
	resp, err := a.nsxClient.ServicesApi.DeleteDhcpIpPool(a.nsxClient.Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		state.AdvancedDHCP.IPPoolID = nil
		return false, nil
	}
	if err != nil {
		return false, err
	}
	state.AdvancedDHCP.IPPoolID = nil
	return true, nil
}

func (a *nsxtAccess) generateID() string {
	return uuid.New().String()
}

func (a *nsxtAccess) getRealizedIPAddress(ipAllocationPath string, timeout time.Duration) (*string, error) {
	client := realized_state.NewDefaultRealizedEntitiesClient(a.connector)

	// wait for realized state
	limit := time.Now().Add(timeout)
	sleepIncr := 100 * time.Millisecond
	sleepMax := 1000 * time.Millisecond
	sleep := sleepIncr
	for time.Now().Before(limit) {
		time.Sleep(sleep)
		sleep += sleepIncr
		if sleep > sleepMax {
			sleep = sleepMax
		}
		list, err := client.List(ipAllocationPath)
		if err != nil {
			return nil, nicerVAPIError(err)
		}
		for _, realizedResource := range list.Results {
			for _, attr := range realizedResource.ExtendedAttributes {
				if *attr.Key == "allocation_ip" {
					return &attr.Values[0], nil
				}
			}
		}
	}
	return nil, fmt.Errorf("timeout of wait for realized state of IP allocation")
}

func nicerVAPIError(err error) error {
	switch vapiError := err.(type) {
	case vapierrors.InvalidRequest:
		// Connection errors end up here
		return nicerVapiErrorData("InvalidRequest", vapiError.Data, vapiError.Messages)
	case vapierrors.NotFound:
		return nicerVapiErrorData("NotFound", vapiError.Data, vapiError.Messages)
	case vapierrors.Unauthorized:
		return nicerVapiErrorData("Unauthorized", vapiError.Data, vapiError.Messages)
	case vapierrors.Unauthenticated:
		return nicerVapiErrorData("Unauthenticated", vapiError.Data, vapiError.Messages)
	case vapierrors.InternalServerError:
		return nicerVapiErrorData("InternalServerError", vapiError.Data, vapiError.Messages)
	case vapierrors.ServiceUnavailable:
		return nicerVapiErrorData("ServiceUnavailable", vapiError.Data, vapiError.Messages)
	}

	return err
}

func nicerVapiErrorData(errorMsg string, apiErrorDataValue *data.StructValue, messages []std.LocalizableMessage) error {
	if apiErrorDataValue == nil {
		if len(messages) > 0 {
			return fmt.Errorf("%s (%s)", errorMsg, messages[0].DefaultMessage)
		}
		return fmt.Errorf("%s (no additional details provided)", errorMsg)
	}

	var typeConverter = bindings.NewTypeConverter()
	typeConverter.SetMode(bindings.REST)
	rawData, err := typeConverter.ConvertToGolang(apiErrorDataValue, model.ApiErrorBindingType())

	if err != nil {
		return fmt.Errorf("%s (failed to extract additional details due to %s)", errorMsg, err)
	}
	apiError := rawData.(model.ApiError)
	details := fmt.Sprintf(" %s: %s (code %v)", errorMsg, *apiError.ErrorMessage, *apiError.ErrorCode)

	if len(apiError.RelatedErrors) > 0 {
		details += "\nRelated errors:\n"
		for _, relatedErr := range apiError.RelatedErrors {
			details += fmt.Sprintf("%s (code %v)", *relatedErr.ErrorMessage, relatedErr.ErrorCode)
		}
	}
	return fmt.Errorf(details)
}

func equalOrderedStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, ai := range a {
		found := false
		for _, bi := range b {
			if ai == bi {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func equalCommonTags(a, b []common.Tag) bool {
	if len(a) != len(b) {
		return false
	}
	for _, ai := range a {
		found := false
		for _, bi := range b {
			if ai.Scope == bi.Scope && ai.Tag == bi.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func equalTags(a, b []model.Tag) bool {
	if len(a) != len(b) {
		return false
	}
	for _, ai := range a {
		found := false
		for _, bi := range b {
			if *ai.Scope == *bi.Scope && *ai.Tag == *bi.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
