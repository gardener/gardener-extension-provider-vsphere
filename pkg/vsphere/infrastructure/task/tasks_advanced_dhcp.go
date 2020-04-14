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
	"reflect"

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

func (t *advancedLookupLogicalSwitchTask) Ensure(ctx EnsurerContext, _ vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	result, resp, err := ctx.NSXTClient().LogicalSwitchingApi.ListLogicalSwitches(ctx.NSXTClient().Context, nil)
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////

type advancedDHCPProfileTask struct{ baseTask }

func NewAdvancedDHCPProfileTask() Task {
	return &advancedDHCPProfileTask{baseTask{label: "DHCP profile (Advanced API)"}}
}

func (t *advancedDHCPProfileTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.ProfileID)
}

func (t *advancedDHCPProfileTask) Ensure(ctx EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	profile := manager.DhcpProfile{
		DisplayName:   spec.FullClusterName(),
		Description:   description,
		EdgeClusterId: state.EdgeClusterRef.ID,
		Tags:          spec.CreateCommonTags(),
	}

	if state.AdvancedDHCP.ProfileID != nil {
		oldProfile, resp, err := ctx.NSXTClient().ServicesApi.ReadDhcpProfile(ctx.NSXTClient().Context, *state.AdvancedDHCP.ProfileID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ProfileID = nil
			return t.Ensure(ctx, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldProfile.DisplayName != profile.DisplayName ||
			oldProfile.EdgeClusterId != profile.EdgeClusterId ||
			!containsCommonTags(oldProfile.Tags, profile.Tags) {
			profile.Tags = mergeCommonTags(profile.Tags, oldProfile.Tags)
			_, resp, err := ctx.NSXTClient().ServicesApi.UpdateDhcpProfile(ctx.NSXTClient().Context, *state.AdvancedDHCP.ProfileID, profile)
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

	createdProfile, resp, err := ctx.NSXTClient().ServicesApi.CreateDhcpProfile(ctx.NSXTClient().Context, profile)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.ProfileID = &createdProfile.Id
	return actionCreated, nil
}

func (t *advancedDHCPProfileTask) TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := ctx.NSXTClient().ServicesApi.ListDhcpProfiles(ctx.NSXTClient().Context, nil)
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

func (t *advancedDHCPProfileTask) EnsureDeleted(ctx EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ProfileID == nil {
		return false, nil
	}
	resp, err := ctx.NSXTClient().ServicesApi.DeleteDhcpProfile(ctx.NSXTClient().Context, *state.AdvancedDHCP.ProfileID)
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////

type advancedDHCPServerTask struct{ baseTask }

func NewAdvancedDHCPServerTask() Task {
	return &advancedDHCPServerTask{baseTask{label: "DHCP server (Advanced API)"}}
}

func (t *advancedDHCPServerTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.ServerID)
}

func (t *advancedDHCPServerTask) Ensure(ctx EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	cfg, err := newDHCPConfig(spec)
	if err != nil {
		return "", err
	}
	ipv4DhcpServer := manager.IPv4DhcpServer{
		DhcpServerIp:   cfg.DHCPServerAddress,
		DnsNameservers: cfg.DNSServers,
		GatewayIp:      cfg.GatewayIP,
	}

	server := manager.LogicalDhcpServer{
		Description:    description,
		DisplayName:    spec.FullClusterName(),
		Tags:           spec.CreateCommonTags(),
		DhcpProfileId:  *state.AdvancedDHCP.ProfileID,
		Ipv4DhcpServer: &ipv4DhcpServer,
	}

	if state.AdvancedDHCP.ServerID != nil {
		oldServer, resp, err := ctx.NSXTClient().ServicesApi.ReadDhcpServer(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.ServerID = nil
			return t.Ensure(ctx, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if oldServer.DisplayName != server.DisplayName ||
			oldServer.DhcpProfileId != server.DhcpProfileId ||
			oldServer.Ipv4DhcpServer == nil ||
			oldServer.Ipv4DhcpServer.DhcpServerIp != server.Ipv4DhcpServer.DhcpServerIp ||
			oldServer.Ipv4DhcpServer.GatewayIp != server.Ipv4DhcpServer.GatewayIp ||
			!reflect.DeepEqual(oldServer.Ipv4DhcpServer.DnsNameservers, server.Ipv4DhcpServer.DnsNameservers) ||
			!containsCommonTags(oldServer.Tags, server.Tags) {
			server.Tags = mergeCommonTags(server.Tags, oldServer.Tags)
			_, resp, err := ctx.NSXTClient().ServicesApi.UpdateDhcpServer(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, server)
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

	createdServer, resp, err := ctx.NSXTClient().ServicesApi.CreateDhcpServer(ctx.NSXTClient().Context, server)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.ServerID = &createdServer.Id
	return actionCreated, nil
}

func (t *advancedDHCPServerTask) TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := ctx.NSXTClient().ServicesApi.ListDhcpServers(ctx.NSXTClient().Context, nil)
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

func (t *advancedDHCPServerTask) EnsureDeleted(ctx EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.ServerID == nil {
		return false, nil
	}
	resp, err := ctx.NSXTClient().ServicesApi.DeleteDhcpServer(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID)
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////

type advancedDHCPPortTask struct{ baseTask }

func NewAdvancedDHCPPortTask() Task {
	return &advancedDHCPPortTask{baseTask{label: "DHCP port (Advanced API)"}}
}

func (t *advancedDHCPPortTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.PortID)
}

func (t *advancedDHCPPortTask) Ensure(ctx EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
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

	if state.AdvancedDHCP.PortID != nil {
		oldPort, resp, err := ctx.NSXTClient().LogicalSwitchingApi.GetLogicalPort(ctx.NSXTClient().Context, *state.AdvancedDHCP.PortID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.PortID = nil
			return t.Ensure(ctx, spec, state)
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
			!containsCommonTags(oldPort.Tags, port.Tags) {
			port.Tags = mergeCommonTags(port.Tags, oldPort.Tags)
			_, resp, err := ctx.NSXTClient().LogicalSwitchingApi.UpdateLogicalPort(ctx.NSXTClient().Context, *state.AdvancedDHCP.PortID, port)
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

	createdPort, resp, err := ctx.NSXTClient().LogicalSwitchingApi.CreateLogicalPort(ctx.NSXTClient().Context, port)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.PortID = &createdPort.Id
	return actionCreated, nil
}

func (t *advancedDHCPPortTask) TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := ctx.NSXTClient().LogicalSwitchingApi.ListLogicalPorts(ctx.NSXTClient().Context, nil)
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

func (t *advancedDHCPPortTask) EnsureDeleted(ctx EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.PortID == nil {
		return false, nil
	}
	localVarOptionals := make(map[string]interface{})
	localVarOptionals["detach"] = true
	resp, err := ctx.NSXTClient().LogicalSwitchingApi.DeleteLogicalPort(ctx.NSXTClient().Context, *state.AdvancedDHCP.PortID, localVarOptionals)
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////

type advancedDHCPIPPoolTask struct{ baseTask }

func NewAdvancedDHCPIPPoolTask() Task {
	return &advancedDHCPIPPoolTask{baseTask{label: "DHCP IP pool (Advanced API)"}}
}

func (t *advancedDHCPIPPoolTask) Reference(state *api.NSXTInfraState) *api.Reference {
	return toReference(state.AdvancedDHCP.IPPoolID)
}

func (t *advancedDHCPIPPoolTask) Ensure(ctx EnsurerContext, spec vinfra.NSXTInfraSpec, state *api.NSXTInfraState) (string, error) {
	cfg, err := newDHCPConfig(spec)
	if err != nil {
		return "", err
	}
	ipPoolRange := manager.IpPoolRange{
		Start: cfg.StartIP,
		End:   cfg.EndIP,
	}
	pool := manager.DhcpIpPool{
		DisplayName:      spec.FullClusterName(),
		Description:      description,
		GatewayIp:        cfg.GatewayIP,
		LeaseTime:        cfg.LeaseTime,
		ErrorThreshold:   98,
		WarningThreshold: 70,
		AllocationRanges: []manager.IpPoolRange{ipPoolRange},
		Tags:             spec.CreateCommonTags(),
	}

	if state.AdvancedDHCP.IPPoolID != nil {
		oldPool, resp, err := ctx.NSXTClient().ServicesApi.ReadDhcpIpPool(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			state.AdvancedDHCP.IPPoolID = nil
			return t.Ensure(ctx, spec, state)
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
			!containsCommonTags(oldPool.Tags, pool.Tags) {
			pool.Tags = mergeCommonTags(pool.Tags, oldPool.Tags)
			_, resp, err := ctx.NSXTClient().ServicesApi.UpdateDhcpIpPool(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID, pool)
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

	createdPool, resp, err := ctx.NSXTClient().ServicesApi.CreateDhcpIpPool(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, pool)
	if err != nil {
		return creatingErr(err)
	}
	if resp.StatusCode != http.StatusCreated {
		return creatingStateCode(resp.StatusCode)
	}
	state.AdvancedDHCP.IPPoolID = &createdPool.Id
	return actionCreated, nil
}

func (t *advancedDHCPIPPoolTask) TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, tags []common.Tag) bool {
	result, resp, err := ctx.NSXTClient().ServicesApi.ListDhcpIpPools(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, nil)
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

func (t *advancedDHCPIPPoolTask) EnsureDeleted(ctx EnsurerContext, state *api.NSXTInfraState) (bool, error) {
	if state.AdvancedDHCP.IPPoolID == nil {
		return false, nil
	}
	resp, err := ctx.NSXTClient().ServicesApi.DeleteDhcpIpPool(ctx.NSXTClient().Context, *state.AdvancedDHCP.ServerID, *state.AdvancedDHCP.IPPoolID)
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////

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

func mergeCommonTags(a []common.Tag, b []common.Tag) []common.Tag {
	result := make([]common.Tag, len(a))
	copy(result, a)
outer:
	for _, tag := range b {
		for _, t := range a {
			if t.Scope == tag.Scope {
				continue outer
			}
		}
		result = append(result, tag)
	}
	return result
}
