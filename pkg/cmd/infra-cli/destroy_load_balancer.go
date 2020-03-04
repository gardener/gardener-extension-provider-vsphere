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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cloud-provider-vsphere/pkg/cloudprovider/vsphere/loadbalancer"
	"k8s.io/cloud-provider-vsphere/pkg/cloudprovider/vsphere/loadbalancer/config"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

func DestroyLoadBalancers(cfg *infrastructure.NSXTConfig, clusterName, ipPoolName, owner string) error {
	lbCfg := &config.LBConfig{
		LoadBalancer: config.LoadBalancerConfig{
			LoadBalancerClassConfig: config.LoadBalancerClassConfig{
				IPPoolName: ipPoolName,
			},
			Size: "SMALL",
			AdditionalTags: map[string]string{
				loadbalancer.ScopeOwner: owner,
			},
		},
		LoadBalancerClasses: map[string]*config.LoadBalancerClassConfig{},
		NSXT: config.NsxtConfig{
			User:         cfg.User,
			Password:     cfg.Password,
			Host:         cfg.Host,
			InsecureFlag: cfg.InsecureFlag,
		},
	}
	lbProvider, err := loadbalancer.NewLBProvider(lbCfg)
	if err != nil {
		return errors.Wrapf(err, "NewLBProvider failed")
	}

	return lbProvider.CleanupServices(clusterName, map[types.NamespacedName]corev1.Service{})
}
