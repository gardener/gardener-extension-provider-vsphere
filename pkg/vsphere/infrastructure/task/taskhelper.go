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
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std"
	vapierrors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/realized_state"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	api "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
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
	actionExternal  = "external"
)

type baseTask struct {
	label string
}

var _ Task = &baseTask{}

func (t *baseTask) Label() string {
	return t.label
}

func (t *baseTask) Ensure(_ EnsurerContext, _ vinfra.NSXTInfraSpec, _ *api.NSXTInfraState) (action string, err error) {
	return "", nil
}

func (t *baseTask) EnsureDeleted(_ EnsurerContext, _ *api.NSXTInfraState) (deleted bool, err error) {
	return false, nil
}

func (t *baseTask) NameToLog(_ vinfra.NSXTInfraSpec) *string {
	return nil
}

func (t *baseTask) Reference(_ *api.NSXTInfraState) *api.Reference {
	return nil
}

func (t *baseTask) IsExternal(_ *api.NSXTInfraState) bool {
	return false
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New())
}

func getRealizedIPAddress(connector vapiclient.Connector, ipAllocationPath string, timeout time.Duration) (*string, error) {
	client := realized_state.NewRealizedEntitiesClient(connector)

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
		list, err := client.List(ipAllocationPath, nil)
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

func readingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "reading")
}

func updatingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "updating")
}

func creatingErr(err error) (string, error) {
	return "", errors.Wrapf(nicerVAPIError(err), "creating")
}

func toReference(s *string) *api.Reference {
	if s == nil {
		return nil
	}
	return &api.Reference{ID: *s}
}

func TryRecover(ctx EnsurerContext, state *api.NSXTInfraState, rt RecoverableTask, tags []model.Tag) bool {
	return reflectTryRecover(ctx, state, rt, tags)
}

func reflectTryRecover(ctx EnsurerContext, state *api.NSXTInfraState, rt RecoverableTask, tags []model.Tag) bool {
	var cursor *string
	total := 0
	count := 0
	for {
		result, err := rt.ListAll(ctx, state, cursor)
		if err != nil {
			return false
		}
		vresult := reflect.ValueOf(result)
		fieldResults := vresult.FieldByName("Results")
		fieldResultCount := vresult.FieldByName("ResultCount")
		fieldCursor := vresult.FieldByName("Cursor")
		items := fieldResults.Len()
		for i := 0; i < items; i++ {
			vitem := fieldResults.Index(i)
			fieldTags := vitem.FieldByName("Tags")
			itemTags := (fieldTags.Interface()).([]model.Tag)
			if containsTags(itemTags, tags) {
				// found
				id := vitem.FieldByName("Id").Elem().String()
				path := vitem.FieldByName("Path").Elem().String()
				pName := vitem.FieldByName("DisplayName").Interface().(*string)
				pRef := &api.Reference{ID: id, Path: path}
				rt.SetRecoveredReference(state, pRef, pName)
				return true
			}
		}
		if cursor == nil {
			total = int(fieldResultCount.Elem().Int())
		}
		count += items
		if count >= total || items == 0 {
			return false
		}
		s := fieldCursor.Elem().String()
		cursor = &s
	}
}

type dhcpConfig struct {
	GatewayIP         string
	GatewayAddress    string
	Network           string
	DHCPServerAddress string
	DNSServers        []string
	DHCPOptions       map[int][]string
	StartIP           string
	EndIP             string
	SubnetMask        string
	LeaseTime         int64
}

func newDHCPConfig(spec vinfra.NSXTInfraSpec) (*dhcpConfig, error) {
	gatewayIP, err := cidrHost(spec.WorkersNetwork, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "gateway ip")
	}
	gatewayAddr, err := cidrHostAndPrefix(spec.WorkersNetwork, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "gateway address")
	}
	dhcpServerAddress, err := cidrHostAndPrefix(spec.WorkersNetwork, 2)
	if err != nil {
		return nil, errors.Wrapf(err, "DHCP server IP")
	}
	startIP, err := cidrHost(spec.WorkersNetwork, 10)
	if err != nil {
		return nil, errors.Wrapf(err, "start IP of pool")
	}
	endIP, err := cidrHost(spec.WorkersNetwork, -1)
	if err != nil {
		return nil, errors.Wrapf(err, "end IP of pool")
	}
	subnetMask, err := cidrSubnetMask(spec.WorkersNetwork)
	if err != nil {
		return nil, errors.Wrapf(err, "subnet mask of pool")
	}

	return &dhcpConfig{
		GatewayIP:         gatewayIP,
		GatewayAddress:    gatewayAddr,
		Network:           spec.WorkersNetwork,
		DHCPServerAddress: dhcpServerAddress,
		StartIP:           startIP,
		EndIP:             endIP,
		SubnetMask:        subnetMask,
		LeaseTime:         int64(2 * time.Hour.Seconds()),
		DNSServers:        spec.DNSServers,
		DHCPOptions:       spec.DHCPOptions,
	}, nil
}

func LookupIPPoolIDByName(ctx EnsurerContext, name string) (string, string, error) {
	client := infra.NewIpPoolsClient(ctx.Connector())
	var cursor *string
	total := 0
	count := 0
	for {
		result, err := client.List(cursor, nil, nil, nil, nil, nil)
		if err != nil {
			return "", "", nicerVAPIError(err)
		}
		for _, item := range result.Results {
			if *item.DisplayName == name {
				// found
				return *item.Id, *item.Path, nil
			}
		}
		if cursor == nil {
			total = int(*result.ResultCount)
		}
		count += len(result.Results)
		if count >= total {
			return "", "", fmt.Errorf("not found: %s", name)
		}
		cursor = result.Cursor
	}
}

func CheckShootAuthorizationByTags(logger logr.Logger, objectType, name, shootNamespace, gardenID string, tags map[string]string) error {
	gardenIDValue := tags[vinfra.ScopeGarden]
	if len(gardenIDValue) == 0 {
		return fmt.Errorf("shoot %s is not authorized to use the %s %s (missing tag %s)",
			shootNamespace, objectType, name, vinfra.ScopeGarden)
	}
	if gardenID != gardenIDValue {
		return fmt.Errorf("shoot %s is not authorized to use the %s %s (gardenID mismatch: %s != %s)",
			shootNamespace, objectType, name, gardenID, gardenIDValue)
	}
	authorizedShoots := tags[vinfra.ScopeAuthorizedShoots]
	if len(authorizedShoots) == 0 {
		return fmt.Errorf("shoot %s is not authorized to use the %s %s (missing tag %s)",
			shootNamespace, objectType, name, vinfra.ScopeAuthorizedShoots)
	}

	for _, part := range strings.Split(authorizedShoots, ",") {
		if strings.Contains(part, "*") {
			reString := fmt.Sprintf("^%s$", strings.ReplaceAll(part, "*", ".*"))
			re, err := regexp.Compile(reString)
			if err != nil {
				logger.Info("invalid regex in checkShootAuthorizationByTags",
					"objectType", objectType, "name", name, "part", part)
				continue
			}
			if re.MatchString(shootNamespace) {
				return nil // found
			}
		} else if part == shootNamespace {
			return nil // found
		}
	}

	return fmt.Errorf("shoot %s is not authorized to use the %s %s (no match in value of tag %s)",
		shootNamespace, objectType, name, vinfra.ScopeAuthorizedShoots)
}
