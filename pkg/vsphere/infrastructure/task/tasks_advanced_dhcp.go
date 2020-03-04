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
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/manager"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

type advancedLookupLogicalSwitchTask struct{ baseTask }

func NewAdvancedLookupLogicalSwitchTask() Task {
	return &advancedLookupLogicalSwitchTask{baseTask{label: "logical switch lookup (Advanced API)"}}
}

func (t *advancedLookupLogicalSwitchTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.LogicalSwitchID)
}

func (t *advancedLookupLogicalSwitchTask) Ensure(a EnsurerContext, _ vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	result, resp, err := a.NSXTClient().LogicalSwitchingApi.ListLogicalSwitches(a.NSXTClient().Context, nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("listing failed with unexpected HTTP status code %d", resp.StatusCode)
	}
	for _, obj := range result.Results {
		for _, tag := range obj.Tags {
			if tag.Scope == "policyPath" && tag.Tag == state.SegmentRef.Path {
				state.AdvancedDHCP.LogicalSwitchID = &obj.Id
				return actionFound, nil
			}
		}
	}
	return "", fmt.Errorf("not found by segment path %s", state.SegmentRef.Path)
}

type advancedDHCPProfileTask struct{ baseTask }

func NewAdvancedDHCPProfileTask() Task {
	return &advancedDHCPProfileTask{baseTask{label: "DHCP profile (Advanced API)"}}
}

func (t *advancedDHCPProfileTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.ProfileID)
}

func (t *advancedDHCPProfileTask) Ensure(a EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	profile := manager.DhcpProfile{
		DisplayName:   spec.FullClusterName(),
		Description:   description,
		EdgeClusterId: state.EdgeClusterRef.ID,
		Tags:          spec.CreateCommonTags(),
	}

	if a.TryRecover() && state.AdvancedDHCP.ProfileID == nil {
		t.tryRecover(a, state, profile.Tags)
	}

	if state.AdvancedDHCP.ProfileID != nil {
		oldProfile, resp, err := a.NSXTClient().ServicesApi.ReadDhcpProfile(a.NSXTClient().Context, *state.AdvancedDHCP.ProfileID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ProfileID = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldProfile.DisplayName != profile.DisplayName ||
			oldProfile.EdgeClusterId != profile.EdgeClusterId ||
			!equalCommonTags(oldProfile.Tags, profile.Tags) {
			_, resp, err := a.NSXTClient().ServicesApi.UpdateDhcpProfile(a.NSXTClient().Context, *state.AdvancedDHCP.ProfileID, profile)
			if err != nil {
				return updatingErr(err)
			}
			if resp.StatusCode != http.StatusOK {
				return updatingStateCode(resp.StatusCode)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	createdProfile, resp, err := a.NSXTClient().ServicesApi.CreateDhcpProfile(a.NSXTClient().Context, profile)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.ProfileID = &createdProfile.Id
	return actionCreated, nil
}

func (t *advancedDHCPProfileTask) tryRecover(a EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := a.NSXTClient().ServicesApi.ListDhcpProfiles(a.NSXTClient().Context, nil)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	for _, item := range result.Results {
		if containsCommonTags(item.Tags, tags) {
			state.AdvancedDHCP.ProfileID = &item.Id
			return true
		}
	}
	return false
}

func (t *advancedDHCPProfileTask) EnsureDeleted(a EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ProfileID == nil {
		return false, nil
	}
	resp, err := a.NSXTClient().ServicesApi.DeleteDhcpProfile(a.NSXTClient().Context, *state.AdvancedDHCP.ProfileID)
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

type advancedDHCPServerTask struct{ baseTask }

func NewAdvancedDHCPServerTask() Task {
	return &advancedDHCPServerTask{baseTask{label: "DHCP server (Advanced API)"}}
}

func (t *advancedDHCPServerTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.ServerID)
}

func (t *advancedDHCPServerTask) Ensure(a EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	dhcpServerIP, err := cidrHostAndPrefix(spec.WorkersNetwork, 2)
	if err != nil {
		return "", errors.Wrapf(err, "DHCP server IP")
	}
	gatewayIP, err := cidrHost(spec.WorkersNetwork, 1)
	if err != nil {
		return "", errors.Wrapf(err, "gateway IP")
	}
	ipv4DhcpServer := manager.IPv4DhcpServer{
		DhcpServerIp:   dhcpServerIP,
		DnsNameservers: spec.DNSServers,
		GatewayIp:      gatewayIP,
	}

	server := manager.LogicalDhcpServer{
		Description:    description,
		DisplayName:    spec.FullClusterName(),
		Tags:           spec.CreateCommonTags(),
		DhcpProfileId:  *state.AdvancedDHCP.ProfileID,
		Ipv4DhcpServer: &ipv4DhcpServer,
	}

	if a.TryRecover() && state.AdvancedDHCP.ServerID == nil {
		t.tryRecover(a, state, server.Tags)
	}

	if state.AdvancedDHCP.ServerID != nil {
		oldServer, resp, err := a.NSXTClient().ServicesApi.ReadDhcpServer(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ServerID = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldServer.DisplayName != server.DisplayName ||
			oldServer.DhcpProfileId != server.DhcpProfileId ||
			oldServer.Ipv4DhcpServer == nil ||
			oldServer.Ipv4DhcpServer.DhcpServerIp != server.Ipv4DhcpServer.DhcpServerIp ||
			oldServer.Ipv4DhcpServer.GatewayIp != server.Ipv4DhcpServer.GatewayIp ||
			!equalOrderedStrings(oldServer.Ipv4DhcpServer.DnsNameservers, server.Ipv4DhcpServer.DnsNameservers) ||
			!equalCommonTags(oldServer.Tags, server.Tags) {
			_, resp, err := a.NSXTClient().ServicesApi.UpdateDhcpServer(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, server)
			if err != nil {
				return updatingErr(err)
			}
			if resp.StatusCode != http.StatusOK {
				return updatingStateCode(resp.StatusCode)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	createdServer, resp, err := a.NSXTClient().ServicesApi.CreateDhcpServer(a.NSXTClient().Context, server)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.ServerID = &createdServer.Id
	return actionCreated, nil
}

func (t *advancedDHCPServerTask) tryRecover(a EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := a.NSXTClient().ServicesApi.ListDhcpServers(a.NSXTClient().Context, nil)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	for _, item := range result.Results {
		if containsCommonTags(item.Tags, tags) {
			state.AdvancedDHCP.ServerID = &item.Id
			return true
		}
	}
	return false
}

func (t *advancedDHCPServerTask) EnsureDeleted(a EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ServerID == nil {
		return false, nil
	}
	resp, err := a.NSXTClient().ServicesApi.DeleteDhcpServer(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID)
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

type advancedDHCPPortTask struct{ baseTask }

func NewAdvancedDHCPPortTask() Task {
	return &advancedDHCPPortTask{baseTask{label: "DHCP port (Advanced API)"}}
}

func (t *advancedDHCPPortTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.PortID)
}

func (t *advancedDHCPPortTask) Ensure(a EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	attachment := manager.LogicalPortAttachment{
		AttachmentType: "DHCP_SERVICE",
		Id:             *state.AdvancedDHCP.ServerID,
	}
	port := manager.LogicalPort{
		DisplayName:     spec.FullClusterName(),
		Description:     description,
		LogicalSwitchId: *state.AdvancedDHCP.LogicalSwitchID,
		AdminState:      "UP",
		Tags:            spec.CreateCommonTags(),
		Attachment:      &attachment,
	}

	if a.TryRecover() && state.AdvancedDHCP.PortID == nil {
		t.tryRecover(a, state, port.Tags)
	}

	if state.AdvancedDHCP.PortID != nil {
		oldPort, resp, err := a.NSXTClient().LogicalSwitchingApi.GetLogicalPort(a.NSXTClient().Context, *state.AdvancedDHCP.PortID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.PortID = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldPort.DisplayName != port.DisplayName ||
			oldPort.LogicalSwitchId != port.LogicalSwitchId ||
			oldPort.AdminState != port.AdminState ||
			oldPort.Attachment == nil ||
			oldPort.Attachment.AttachmentType != port.Attachment.AttachmentType ||
			oldPort.Attachment.Id != port.Attachment.Id ||
			!equalCommonTags(oldPort.Tags, port.Tags) {
			_, resp, err := a.NSXTClient().LogicalSwitchingApi.UpdateLogicalPort(a.NSXTClient().Context, *state.AdvancedDHCP.PortID, port)
			if err != nil {
				return updatingErr(err)
			}
			if resp.StatusCode != http.StatusOK {
				return updatingStateCode(resp.StatusCode)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	createdPort, resp, err := a.NSXTClient().LogicalSwitchingApi.CreateLogicalPort(a.NSXTClient().Context, port)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.PortID = &createdPort.Id
	return actionCreated, nil
}

func (t *advancedDHCPPortTask) tryRecover(a EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := a.NSXTClient().LogicalSwitchingApi.ListLogicalPorts(a.NSXTClient().Context, nil)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	for _, item := range result.Results {
		if containsCommonTags(item.Tags, tags) {
			state.AdvancedDHCP.PortID = &item.Id
			return true
		}
	}
	return false
}

func (t *advancedDHCPPortTask) EnsureDeleted(a EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.PortID == nil {
		return false, nil
	}
	localVarOptionals := make(map[string]interface{})
	localVarOptionals["detach"] = true
	resp, err := a.NSXTClient().LogicalSwitchingApi.DeleteLogicalPort(a.NSXTClient().Context, *state.AdvancedDHCP.PortID, localVarOptionals)
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

type advancedDHCPIPPoolTask struct{ baseTask }

func NewAdvancedDHCPIPPoolTask() Task {
	return &advancedDHCPIPPoolTask{baseTask{label: "DHCP IP pool (Advanced API)"}}
}

func (t *advancedDHCPIPPoolTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.IPPoolID)
}

func (t *advancedDHCPIPPoolTask) Ensure(a EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	gatewayIP, err := cidrHost(spec.WorkersNetwork, 1)
	if err != nil {
		return "", errors.Wrapf(err, "gateway IP")
	}
	startIP, err := cidrHost(spec.WorkersNetwork, 10)
	if err != nil {
		return "", errors.Wrapf(err, "start IP of pool")
	}
	endIP, err := cidrHost(spec.WorkersNetwork, -1)
	if err != nil {
		return "", errors.Wrapf(err, "end IP of pool")
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
		Tags:             spec.CreateCommonTags(),
	}

	if a.TryRecover() && state.AdvancedDHCP.IPPoolID == nil {
		t.tryRecover(a, state, pool.Tags)
	}

	if state.AdvancedDHCP.IPPoolID != nil {
		oldPool, resp, err := a.NSXTClient().ServicesApi.ReadDhcpIpPool(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.IPPoolID = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldPool.DisplayName != pool.DisplayName ||
			oldPool.GatewayIp != pool.GatewayIp ||
			oldPool.LeaseTime != pool.LeaseTime ||
			oldPool.ErrorThreshold != pool.ErrorThreshold ||
			oldPool.WarningThreshold != pool.WarningThreshold ||
			len(oldPool.AllocationRanges) != 1 ||
			oldPool.AllocationRanges[0].Start != pool.AllocationRanges[0].Start ||
			oldPool.AllocationRanges[0].End != pool.AllocationRanges[0].End ||
			!equalCommonTags(oldPool.Tags, pool.Tags) {
			_, resp, err := a.NSXTClient().ServicesApi.UpdateDhcpIpPool(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID, pool)
			if err != nil {
				return updatingErr(err)
			}
			if resp.StatusCode != http.StatusOK {
				return updatingStateCode(resp.StatusCode)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	createdPool, resp, err := a.NSXTClient().ServicesApi.CreateDhcpIpPool(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, pool)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.IPPoolID = &createdPool.Id
	return actionCreated, nil
}

func (t *advancedDHCPIPPoolTask) tryRecover(a EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := a.NSXTClient().ServicesApi.ListDhcpIpPools(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, nil)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	for _, item := range result.Results {
		if containsCommonTags(item.Tags, tags) {
			state.AdvancedDHCP.IPPoolID = &item.Id
			return true
		}
	}
	return false
}

func (t *advancedDHCPIPPoolTask) EnsureDeleted(a EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.IPPoolID == nil {
		return false, nil
	}
	resp, err := a.NSXTClient().ServicesApi.DeleteDhcpIpPool(a.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
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

func updatingStateCode(statusCode int) (string, error) {
	return "", fmt.Errorf("updating failed with unexpected HTTP status code %d", statusCode)
}

func creatingStateCode(statusCode int) (string, error) {
	return "", fmt.Errorf("creating failed with unexpected HTTP status code %d", statusCode)
}

func containsCommonTags(itemTags []common.Tag, tags []common.Tag) bool {
outer:
	for _, tag := range tags {
		for _, t := range itemTags {
			if t.Scope == tag.Scope {
				if t.Tag == tag.Tag {
					continue outer
				} else {
					return false
				}
			}
		}
		return false
	}
	return true
}
