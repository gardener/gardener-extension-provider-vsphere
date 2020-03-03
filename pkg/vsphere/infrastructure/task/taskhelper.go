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
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std"
	vapierrors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
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

func (t *baseTask) Name(_ vinfra.NSXTInfraSpec) *string {
	return nil
}

func (t *baseTask) Reference(_ *api.NSXTInfraState) *api.Reference {
	return nil
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

func toReference(s *string) *api.Reference {
	if s == nil {
		return nil
	}
	return &api.Reference{ID: *s}
}
