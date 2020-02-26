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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std"
	vapierrors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/ip_pools"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/realized_state"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/sites/enforcement_points"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/tier_1s"
	t1nat "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/tier_1s/nat"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

const (
	description                  = "created by gardener-extension-provider-vsphere"
	defaultSite                  = "default"
	policyEnforcementPoint       = "default"
	defaultPolicyLocaleServiceID = "default"

	actionCreated   = "created"
	actionUpdated   = "updated"
	actionUnchanged = "unchanged"
	actionFound     = "found"
)

type task interface {
	Label() string
	Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (action string, err error)
	EnsureDeleted(a *ensurer, state *NSXTInfraState) (deleted bool, err error)
	name(spec NSXTInfraSpec) *string
	reference(state *NSXTInfraState) *Reference
}

type baseTask struct {
	label string
}

func (t *baseTask) Label() string {
	return t.label
}

func (t *baseTask) Ensure(_ *ensurer, _ NSXTInfraSpec, _ *NSXTInfraState) (action string, err error) {
	return "", nil
}

func (t *baseTask) EnsureDeleted(_ *ensurer, _ *NSXTInfraState) (deleted bool, err error) {
	return false, nil
}

func (t *baseTask) name(_ NSXTInfraSpec) *string {
	return nil
}

func (t *baseTask) reference(_ *NSXTInfraState) *Reference {
	return nil
}

type lookupTier0GatewayTask struct{ baseTask }

func newLookupTier0GatewayTask() *lookupTier0GatewayTask {
	return &lookupTier0GatewayTask{baseTask{label: "tier-0 gateway lookup"}}
}

func (t *lookupTier0GatewayTask) name(spec NSXTInfraSpec) *string { return &spec.Tier0GatewayName }

func (t *lookupTier0GatewayTask) reference(state *NSXTInfraState) *Reference {
	return state.Tier0GatewayRef
}

func (t *lookupTier0GatewayTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	name := spec.Tier0GatewayName
	client := infra.NewDefaultTier0sClient(a.connector)
	var cursor *string
	total := 0
	count := 0
	for {
		result, err := client.List(cursor, nil, nil, nil, nil, nil)
		if err != nil {
			return "", nicerVAPIError(err)
		}
		for _, item := range result.Results {
			if *item.DisplayName == name {
				// found
				state.Tier0GatewayRef = &Reference{ID: *item.Id, Path: *item.Path}
				return actionFound, nil
			}
		}
		if cursor == nil {
			total = int(*result.ResultCount)
		}
		count += len(result.Results)
		if count >= total {
			return "", fmt.Errorf("not found: %s", name)
		}
		cursor = result.Cursor
	}
}

type lookupEdgeClusterTask struct{ baseTask }

func newLookupEdgeClusterTask() *lookupEdgeClusterTask {
	return &lookupEdgeClusterTask{baseTask{label: "edge cluster lookup"}}
}

func (t *lookupEdgeClusterTask) name(spec NSXTInfraSpec) *string { return &spec.EdgeClusterName }

func (t *lookupEdgeClusterTask) reference(state *NSXTInfraState) *Reference {
	return state.EdgeClusterRef
}

func (t *lookupEdgeClusterTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	name := spec.EdgeClusterName
	client := enforcement_points.NewDefaultEdgeClustersClient(a.connector)
	result, err := client.List(defaultSite, policyEnforcementPoint, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return "", nicerVAPIError(err)
	}
	for _, item := range result.Results {
		if *item.DisplayName == name {
			state.EdgeClusterRef = &Reference{ID: *item.Id, Path: *item.Path}
			return actionFound, nil
		}
	}
	return "", fmt.Errorf("not found: %s", name)
}

type lookupTransportZoneTask struct{ baseTask }

func newLookupTransportZoneTask() *lookupTransportZoneTask {
	return &lookupTransportZoneTask{baseTask{label: "transport zone lookup"}}
}

func (t *lookupTransportZoneTask) name(spec NSXTInfraSpec) *string { return &spec.TransportZoneName }

func (t *lookupTransportZoneTask) reference(state *NSXTInfraState) *Reference {
	return state.TransportZoneRef
}

func (t *lookupTransportZoneTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	name := spec.TransportZoneName
	client := enforcement_points.NewDefaultTransportZonesClient(a.connector)
	result, err := client.List(defaultSite, policyEnforcementPoint, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return "", nicerVAPIError(err)
	}
	for _, item := range result.Results {
		if *item.DisplayName == name {
			state.TransportZoneRef = &Reference{ID: *item.Id, Path: *item.Path}
			return actionFound, nil
		}
	}
	return "", fmt.Errorf("not found: %s", name)
}

type lookupSNATIPPoolTask struct{ baseTask }

func newLookupSNATIPPoolTask() *lookupSNATIPPoolTask {
	return &lookupSNATIPPoolTask{baseTask{label: "SNAT IP pool lookup"}}
}

func (t *lookupSNATIPPoolTask) name(spec NSXTInfraSpec) *string { return &spec.SNATIPPoolName }

func (t *lookupSNATIPPoolTask) reference(state *NSXTInfraState) *Reference { return state.SNATIPPoolRef }

func (t *lookupSNATIPPoolTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	name := spec.SNATIPPoolName
	client := infra.NewDefaultIpPoolsClient(a.connector)
	var cursor *string
	total := 0
	count := 0
	for {
		result, err := client.List(cursor, nil, nil, nil, nil, nil)
		if err != nil {
			return "", nicerVAPIError(err)
		}
		for _, item := range result.Results {
			if *item.DisplayName == name {
				// found
				state.SNATIPPoolRef = &Reference{ID: *item.Id, Path: *item.Path}
				return actionFound, nil
			}
		}
		if cursor == nil {
			total = int(*result.ResultCount)
		}
		count += len(result.Results)
		if count >= total {
			return "", fmt.Errorf("not found: %s", name)
		}
		cursor = result.Cursor
	}
}

type tier1GatewayTask struct{ baseTask }

func newTier1GatewayTask() *tier1GatewayTask {
	return &tier1GatewayTask{baseTask{label: "tier-1 gateway"}}
}

func (t *tier1GatewayTask) reference(state *NSXTInfraState) *Reference { return state.Tier1GatewayRef }

func (t *tier1GatewayTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	client := infra.NewDefaultTier1sClient(a.connector)

	tier1 := model.Tier1{
		DisplayName:  strptr(spec.FullClusterName()),
		Description:  strptr(description),
		FailoverMode: strptr(model.Tier1_FAILOVER_MODE_PREEMPTIVE),
		Tags:         spec.createTags(),
		RouteAdvertisementTypes: []string{
			model.Tier1_ROUTE_ADVERTISEMENT_TYPES_STATIC_ROUTES,
			model.Tier1_ROUTE_ADVERTISEMENT_TYPES_NAT,
			model.Tier1_ROUTE_ADVERTISEMENT_TYPES_LB_VIP,
			model.Tier1_ROUTE_ADVERTISEMENT_TYPES_LB_SNAT,
		},
		Tier0Path: &state.Tier0GatewayRef.Path,
	}

	if state.Tier1GatewayRef != nil {
		oldTier1, err := client.Get(state.Tier1GatewayRef.ID)
		if isNotFoundError(err) {
			state.Tier1GatewayRef = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if *oldTier1.DisplayName != *tier1.DisplayName ||
			oldTier1.FailoverMode == nil ||
			*oldTier1.FailoverMode != *tier1.FailoverMode ||
			oldTier1.Tier0Path == nil ||
			*oldTier1.Tier0Path != *tier1.Tier0Path ||
			!equalStrings(oldTier1.RouteAdvertisementTypes, tier1.RouteAdvertisementTypes) ||
			!equalTags(oldTier1.Tags, tier1.Tags) {
			err := client.Patch(state.Tier1GatewayRef.ID, tier1)
			if err != nil {
				return updatingErr(err)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	id := generateID("tier1gw")
	createdObj, err := client.Update(id, tier1)
	if err != nil {
		return creatingErr(err)
	}
	state.Tier1GatewayRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return actionCreated, nil
}

func (t *tier1GatewayTask) EnsureDeleted(a *ensurer, state *NSXTInfraState) (bool, error) {
	client := infra.NewDefaultTier1sClient(a.connector)
	if state.Tier1GatewayRef == nil {
		return false, nil
	}
	err := client.Delete(state.Tier1GatewayRef.ID)
	if err != nil {
		return false, nicerVAPIError(err)
	}
	state.Tier1GatewayRef = nil
	return true, nil
}

type tier1GatewayLocaleServiceTask struct{ baseTask }

func newTier1GatewayLocaleServiceTask() *tier1GatewayLocaleServiceTask {
	return &tier1GatewayLocaleServiceTask{baseTask{label: "tier-1 gateway local service"}}
}

func (t *tier1GatewayLocaleServiceTask) reference(state *NSXTInfraState) *Reference {
	return state.LocaleServiceRef
}

func (t *tier1GatewayLocaleServiceTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	client := tier_1s.NewDefaultLocaleServicesClient(a.connector)

	obj := model.LocaleServices{
		DisplayName:     strptr(spec.FullClusterName()),
		Description:     strptr(description),
		EdgeClusterPath: &state.EdgeClusterRef.Path,
		Tags:            spec.createTags(),
	}

	if state.LocaleServiceRef != nil {
		oldTier1, err := client.Get(state.LocaleServiceRef.ID, defaultPolicyLocaleServiceID)
		if isNotFoundError(err) {
			state.Tier1GatewayRef = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if *oldTier1.DisplayName != *obj.DisplayName ||
			oldTier1.EdgeClusterPath == nil ||
			*oldTier1.EdgeClusterPath != *obj.EdgeClusterPath ||
			!equalTags(oldTier1.Tags, obj.Tags) {
			err := client.Patch(state.LocaleServiceRef.ID, defaultPolicyLocaleServiceID, obj)
			if err != nil {
				return updatingErr(err)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}
	// The default ID of the locale service will be the Tier1 ID
	id := state.Tier1GatewayRef.ID
	err := client.Patch(id, defaultPolicyLocaleServiceID, obj)
	if err != nil {
		return creatingErr(err)
	}
	state.LocaleServiceRef = &Reference{ID: id, Path: ""}
	return actionCreated, nil
}

func (t *tier1GatewayLocaleServiceTask) EnsureDeleted(a *ensurer, state *NSXTInfraState) (bool, error) {
	client := tier_1s.NewDefaultLocaleServicesClient(a.connector)
	if state.LocaleServiceRef == nil {
		return false, nil
	}
	err := client.Delete(state.LocaleServiceRef.ID, defaultPolicyLocaleServiceID)
	if err != nil {
		return false, nicerVAPIError(err)
	}
	state.LocaleServiceRef = nil
	return true, nil
}

type segmentTask struct{ baseTask }

func newSegmentTask() *segmentTask {
	return &segmentTask{baseTask{label: "segment"}}
}

func (t *segmentTask) reference(state *NSXTInfraState) *Reference { return state.SegmentRef }

func (t *segmentTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	client := infra.NewDefaultSegmentsClient(a.connector)

	gatewayAddr, err := cidrHostAndPrefix(spec.WorkersNetwork, 1)
	if err != nil {
		return "", errors.Wrapf(err, "gateway address")
	}
	subnet := model.SegmentSubnet{
		GatewayAddress: strptr(gatewayAddr),
	}
	displayName := spec.FullClusterName() + "-" + RandomString(8)
	segment := model.Segment{
		DisplayName:       strptr(displayName),
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
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if !strings.HasPrefix(*oldSegment.DisplayName, spec.FullClusterName()) ||
			oldSegment.ConnectivityPath == nil ||
			*oldSegment.ConnectivityPath != *segment.ConnectivityPath ||
			oldSegment.TransportZonePath == nil ||
			*oldSegment.TransportZonePath != *segment.TransportZonePath ||
			len(oldSegment.Subnets) != 1 ||
			oldSegment.Subnets[0].GatewayAddress == nil ||
			*oldSegment.Subnets[0].GatewayAddress != *segment.Subnets[0].GatewayAddress ||
			!equalTags(oldSegment.Tags, segment.Tags) {
			err := client.Patch(state.SegmentRef.ID, segment)
			if err != nil {
				return updatingErr(err)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	id := generateID("segment")
	createdObj, err := client.Update(id, segment)
	if err != nil {
		return creatingErr(err)
	}
	state.SegmentRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	state.SegmentName = createdObj.DisplayName
	return actionCreated, nil
}

func (t *segmentTask) EnsureDeleted(a *ensurer, state *NSXTInfraState) (bool, error) {
	client := infra.NewDefaultSegmentsClient(a.connector)
	if state.SegmentRef == nil {
		return false, nil
	}
	err := client.Delete(state.SegmentRef.ID)
	if err != nil {
		return false, nicerVAPIError(err)
	}
	state.SegmentRef = nil
	state.SegmentName = nil
	return true, nil
}

type snatIPAddressAllocationTask struct{ baseTask }

func newSNATIPAddressAllocationTask() *snatIPAddressAllocationTask {
	return &snatIPAddressAllocationTask{baseTask{label: "SNAT IP address allocation"}}
}

func (t *snatIPAddressAllocationTask) reference(state *NSXTInfraState) *Reference {
	return state.SNATIPAddressAllocRef
}

func (t *snatIPAddressAllocationTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
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
			return actionUnchanged, nil
		}
		if !isNotFoundError(err) {
			return readingErr(err)
		}
	}

	id := generateID("snatippool")
	createdObj, err := client.Update(state.SNATIPPoolRef.ID, id, allocation)
	if err != nil {
		return creatingErr(err)
	}
	state.SNATIPAddressAllocRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return actionCreated, nil
}

func (t *snatIPAddressAllocationTask) EnsureDeleted(a *ensurer, state *NSXTInfraState) (bool, error) {
	client := ip_pools.NewDefaultIpAllocationsClient(a.connector)
	if state.SNATIPAddressAllocRef == nil {
		return false, nil
	}
	err := client.Delete(state.SNATIPPoolRef.ID, state.SNATIPAddressAllocRef.ID)
	if err != nil {
		return false, err
	}
	state.SNATIPAddressAllocRef = nil
	state.SNATIPAddress = nil
	return true, nil
}

type snatIPAddressRealizationTask struct{ baseTask }

func newSNATIPAddressRealizationTask() *snatIPAddressRealizationTask {
	return &snatIPAddressRealizationTask{baseTask{label: "SNAT IP address realization"}}
}

func (t *snatIPAddressRealizationTask) reference(state *NSXTInfraState) *Reference {
	return toReference(state.SNATIPAddress)
}

func (t *snatIPAddressRealizationTask) Ensure(a *ensurer, _ NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	ipAddress, err := getRealizedIPAddress(a.connector, state.SNATIPAddressAllocRef.Path, 15*time.Second)
	if err != nil {
		return "", err
	}
	state.SNATIPAddress = ipAddress
	return actionFound, nil
}

type snatRuleTask struct{ baseTask }

func newSNATRuleTask() *snatRuleTask {
	return &snatRuleTask{baseTask{label: "SNAT rule"}}
}

func (t *snatRuleTask) reference(state *NSXTInfraState) *Reference { return state.SNATRuleRef }

func (t *snatRuleTask) Ensure(a *ensurer, spec NSXTInfraSpec, state *NSXTInfraState) (string, error) {
	client := t1nat.NewDefaultNatRulesClient(a.connector)

	rule := model.PolicyNatRule{
		DisplayName:    strptr(spec.FullClusterName()),
		Description:    strptr(description),
		Action:         model.PolicyNatRule_ACTION_SNAT,
		Enabled:        boolptr(true),
		Logging:        boolptr(true),
		SequenceNumber: int64ptr(100),
		Tags:           spec.createTags(),

		SourceNetwork:     strptr(spec.WorkersNetwork),
		TranslatedNetwork: strptr(fmt.Sprintf("%s/32", *state.SNATIPAddress)),
	}

	if state.SNATRuleRef != nil {
		oldRule, err := client.Get(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID)
		if isNotFoundError(err) {
			state.SNATRuleRef = nil
			return t.Ensure(a, spec, state)
		}
		if err != nil {
			return readingErr(err)
		}
		if *oldRule.DisplayName != *rule.DisplayName ||
			oldRule.Action != rule.Action ||
			oldRule.Enabled == nil ||
			*oldRule.Enabled != *rule.Enabled ||
			oldRule.Logging == nil ||
			*oldRule.Logging != *rule.Logging ||
			oldRule.SequenceNumber == nil ||
			*oldRule.SequenceNumber != *rule.SequenceNumber ||
			oldRule.SourceNetwork == nil ||
			*oldRule.SourceNetwork != *rule.SourceNetwork ||
			oldRule.TranslatedNetwork == nil ||
			*oldRule.TranslatedNetwork != *rule.TranslatedNetwork ||
			oldRule.DestinationNetwork != nil ||
			!equalTags(oldRule.Tags, rule.Tags) {
			err := client.Patch(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID, rule)
			if err != nil {
				return updatingErr(err)
			}
			return actionUpdated, nil
		}
		return actionUnchanged, nil
	}

	id := generateID("snatrule")
	createdObj, err := client.Update(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, id, rule)
	if err != nil {
		return creatingErr(err)
	}
	state.SNATRuleRef = &Reference{ID: *createdObj.Id, Path: *createdObj.Path}
	return actionCreated, nil
}

func (t *snatRuleTask) EnsureDeleted(a *ensurer, state *NSXTInfraState) (bool, error) {
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

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New())
}

func getRealizedIPAddress(connector vapiclient.Connector, ipAllocationPath string, timeout time.Duration) (*string, error) {
	client := realized_state.NewDefaultRealizedEntitiesClient(connector)

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

func readingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "reading")
}

func updatingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "updating")
}

func creatingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "creating")
}

func toReference(s *string) *Reference {
	if s == nil {
		return nil
	}
	return &Reference{ID: *s}
}
