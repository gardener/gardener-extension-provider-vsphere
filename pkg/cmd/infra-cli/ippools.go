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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	nsxt "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/manager"

	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/ip_pools"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/ensurer"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"
)

func CreateIPPool(logger logr.Logger, cfg *infrastructure.NSXTConfig, ipPoolName, ipPoolRanges, ipPoolCidr string, advancedAPI bool) error {
	infrastructureEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(logger, cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "creating ensurer failed")
	}

	ensurerContext, ok := infrastructureEnsurer.(task.EnsurerContext)
	if !ok {
		return fmt.Errorf("Cannot acces EnsurerContext")
	}

	if advancedAPI {
		return createIPPoolAdvanced(logger, ensurerContext.NSXTClient(), ipPoolName, ipPoolRanges, ipPoolCidr)
	}
	return createIPPoolPolicy(logger, ensurerContext.Connector(), ipPoolName, ipPoolRanges, ipPoolCidr)
}

func createIPPoolAdvanced(logger logr.Logger, nsxClient *nsxt.APIClient, ipPoolName, ipPoolRanges, ipPoolCidr string) error {
	logger.Info("Creating IP Pool (advanced)", "name", ipPoolName)

	description := "created by gardener-extension-provider-vsphere infra-cli"
	subnets, err := getSubnetsFromRanges(ipPoolRanges, ipPoolCidr)
	if err != nil {
		return err
	}
	ipPool := manager.IpPool{
		DisplayName: ipPoolName,
		Description: description,
		Subnets:     subnets,
		Tags:        []common.Tag{{Scope: "gardener", Tag: "true"}},
	}

	_, resp, err := nsxClient.PoolManagementApi.CreateIpPool(nsxClient.Context, ipPool)

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("CreateIpPool returned unexpected status: %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}

func getSubnetsFromRanges(ipPoolRanges, ipPoolCidr string) ([]manager.IpPoolSubnet, error) {
	ranges := strings.Split(ipPoolRanges, ",")
	var allocationRanges []manager.IpPoolRange
	for _, allocRange := range ranges {
		parts := strings.Split(allocRange, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid range: %s", allocRange)
		}
		elem := manager.IpPoolRange{
			Start: parts[0],
			End:   parts[1],
		}
		allocationRanges = append(allocationRanges, elem)
	}

	return []manager.IpPoolSubnet{
		{
			Cidr:             ipPoolCidr,
			AllocationRanges: allocationRanges,
		},
	}, nil
}

func createIPPoolPolicy(logger logr.Logger, connector vapiclient.Connector, ipPoolName, ipPoolRanges, ipPoolCidr string) error {
	id := ipPoolName
	logger.Info("Creating IP Pool (policy)", "id", id, "name", ipPoolName)

	client := infra.NewDefaultIpPoolsClient(connector)
	_, err := client.Get(id)
	if err == nil {
		return fmt.Errorf("pool with id %s already existing", id)
	}

	description := "created by gardener-extension-provider-vsphere infra-cli"
	obj := model.IpAddressPool{
		DisplayName: &ipPoolName,
		Description: &description,
		Tags:        []model.Tag{{Scope: strptr("gardener"), Tag: strptr("true")}},
		Id:          &id,
	}

	createdObj, err := client.Update(id, obj)
	if err != nil {
		return errors.Wrap(err, "creating IP pool failed")
	}

	subnetClient := ip_pools.NewDefaultIpSubnetsClient(connector)
	subnetID, subnetParam, err := ipAddressPoolStaticSubnetToStructValue(createdObj.Path, ipPoolName, ipPoolRanges, ipPoolCidr)
	if err != nil {
		return errors.Wrap(err, "creating IP pool subnet struct value failed")
	}
	err = subnetClient.Patch(id, subnetID, subnetParam)
	if err != nil {
		return errors.Wrap(err, "creating IP pool subnet failed")
	}
	return nil
}

func ipAddressPoolStaticSubnetToStructValue(poolPath *string, ipPoolName, ipPoolRanges, ipPoolCidr string) (string, *data.StructValue, error) {
	ranges := strings.Split(ipPoolRanges, ",")
	subnetID := newUUID()
	obj := model.IpAddressPoolStaticSubnet{
		DisplayName:  strptr(ipPoolName + "-subnet"),
		Id:           &subnetID,
		ResourceType: "IpAddressPoolStaticSubnet",
		Cidr:         &ipPoolCidr,
		ParentPath:   poolPath,
	}

	var poolRanges []model.IpPoolRange
	for _, allocRange := range ranges {
		parts := strings.Split(allocRange, "-")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("Invalid range: %s", allocRange)
		}
		ipRange := model.IpPoolRange{
			Start: &parts[0],
			End:   &parts[1],
		}
		poolRanges = append(poolRanges, ipRange)
	}
	obj.AllocationRanges = poolRanges

	converter := bindings.NewTypeConverter()
	converter.SetMode(bindings.REST)

	dataValue, errs := converter.ConvertToVapi(obj, model.IpAddressPoolStaticSubnetBindingType())
	if errs != nil {
		return "", nil, fmt.Errorf("Error converting Static Subnet: %v", errs[0])
	}

	return subnetID, dataValue.(*data.StructValue), nil
}

func DeleteIPPool(logger logr.Logger, cfg *infrastructure.NSXTConfig, ipPoolName string, advancedAPI bool) error {
	infrastructureEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(logger, cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "creating ensurer failed")
	}

	ensurerContext, ok := infrastructureEnsurer.(task.EnsurerContext)
	if !ok {
		return fmt.Errorf("Cannot acces EnsurerContext")
	}

	if advancedAPI {
		return deleteIPPoolAdvanced(logger, ensurerContext.NSXTClient(), ipPoolName)
	}
	return deleteIPPoolPolicy(logger, ensurerContext.Connector(), ipPoolName)
}

func deleteIPPoolAdvanced(logger logr.Logger, nsxClient *nsxt.APIClient, ipPoolName string) error {
	logger.Info("Deleting IP Pool (advanced)", "name", ipPoolName)

	list, resp, err := nsxClient.PoolManagementApi.ListIpPools(nsxClient.Context, nil)

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ListIpPools returned unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	for _, pool := range list.Results {
		if pool.DisplayName == ipPoolName {
			resp, err = nsxClient.PoolManagementApi.DeleteIpPool(nsxClient.Context, pool.Id, nil)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("DeleteIpPool returned unexpected status: %d %s", resp.StatusCode, resp.Status)
			}
			return nil
		}
	}
	logger.Info("IP pool not found", "name", ipPoolName)
	return nil
}

func deleteIPPoolPolicy(logger logr.Logger, connector vapiclient.Connector, ipPoolName string) error {
	logger.Info("Deleting IP Pool (policy)", "name", ipPoolName)

	client := infra.NewDefaultIpPoolsClient(connector)
	list, err := client.List(nil, nil, nil, nil, nil, nil)

	if err != nil {
		return err
	}

	for _, pool := range list.Results {
		if pool.DisplayName != nil && *pool.DisplayName == ipPoolName {
			subnetClient := ip_pools.NewDefaultIpSubnetsClient(connector)
			subnets, err := subnetClient.List(*pool.Id, nil, nil, nil, nil, nil, nil)
			if err != nil {
				return errors.Wrap(err, "listing subnets failed")
			}
			for _, subnet := range subnets.Results {
				dataval, err := subnet.Field("id")
				if err != nil {
					return errors.Wrap(err, "subnet id data value failed")
				}
				if dataval.Type() != data.STRING {
					return fmt.Errorf("unexpected type for id data value: %s", dataval.Type())
				}
				subnetID := dataval.(*data.StringValue).Value()
				err = subnetClient.Delete(*pool.Id, subnetID)
				if err != nil {
					return errors.Wrapf(err, "deleting subnet %s failed", subnetID)
				}
			}
			return client.Delete(*pool.Id)
		}
	}
	logger.Info("IP pool %s not found", ipPoolName)
	return nil
}

func strptr(s string) *string {
	return &s
}

func newUUID() string {
	newUUID, _ := uuid.NewRandom()
	return newUUID.String()
}
