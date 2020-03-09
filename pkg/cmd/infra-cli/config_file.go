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
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardencoreinstall "github.com/gardener/gardener/pkg/apis/core/install"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

type configFileBuilder struct {
	kubeconfig         string
	cloudProfileName   string
	region             string
	ctx                context.Context
	client             client.Client
	cloudProfileConfig *vsphere.CloudProfileConfig
}

func BuildConfigFile(kubeconfig, cloudProfileName, region string) (*infrastructure.NSXTConfig, error) {
	gardencoreinstall.Install(scheme.Scheme)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	c, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	builder := configFileBuilder{
		kubeconfig:       kubeconfig,
		cloudProfileName: cloudProfileName,
		region:           region,
		ctx:              context.TODO(),
		client:           c,
	}

	return builder.build()
}

func (b *configFileBuilder) build() (*infrastructure.NSXTConfig, error) {
	err := b.loadCloudProfileConfig()
	if err != nil {
		return nil, err
	}

	cfg := &infrastructure.NSXTConfig{}
	err = b.fillHost(cfg)
	if err != nil {
		return nil, err
	}

	err = b.fillUserPassword(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (b *configFileBuilder) loadCloudProfileConfig() error {
	cloudProfile := &v1beta1.CloudProfile{}
	key := client.ObjectKey{Name: b.cloudProfileName}
	err := b.client.Get(b.ctx, key, cloudProfile)
	if err != nil {
		return errors.Wrapf(err, "cloud profile %s cannot be retrieved", b.cloudProfileName)
	}
	if cloudProfile.Spec.Type != "vsphere" {
		return errors.Wrapf(err, "cloud profile %s has wrong type %s ('vsphere' expected)", b.cloudProfileName, cloudProfile.Spec.Type)
	}
	if cloudProfile.Spec.ProviderConfig == nil {
		return fmt.Errorf("cloud profile %s: missing provider config", b.cloudProfileName)
	}
	b.cloudProfileConfig, err = helper.GetCloudProfileConfigFromProfile(cloudProfile)
	return err
}

func (b *configFileBuilder) fillHost(cfg *infrastructure.NSXTConfig) error {
	if b.region == "" {
		if len(b.cloudProfileConfig.Regions) != 1 {
			return fmt.Errorf("cloud profile has multiple regions, please specify region")
		}
		b.region = b.cloudProfileConfig.Regions[0].Name
	}

	for _, r := range b.cloudProfileConfig.Regions {
		if r.Name == b.region {
			cfg.Host = r.NSXTHost
			cfg.InsecureFlag = r.NSXTInsecureSSL
			return nil
		}
	}
	return fmt.Errorf("region %s not found in cloud profile %s", b.region, b.cloudProfileName)
}

func (b *configFileBuilder) fillUserPassword(cfg *infrastructure.NSXTConfig) error {
	req, err := labels.NewRequirement("cloudprofile.garden.sapcloud.io/name", selection.Equals, []string{b.cloudProfileName})
	if err != nil {
		return err
	}
	labelsSelector := client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(*req),
	}
	var namespace client.InNamespace = "garden"
	secretBindings := &v1beta1.SecretBindingList{}
	err = b.client.List(b.ctx, secretBindings, labelsSelector, namespace)
	if err != nil {
		return errors.Wrapf(err, "list secret bindings failed")
	}

	if len(secretBindings.Items) != 1 {
		return fmt.Errorf("listing secret bindings returned %d != 1 items", len(secretBindings.Items))
	}
	secretKey := client.ObjectKey{
		Namespace: secretBindings.Items[0].SecretRef.Namespace,
		Name:      secretBindings.Items[0].SecretRef.Name,
	}
	secret := &corev1.Secret{}
	err = b.client.Get(b.ctx, secretKey, secret)
	if err != nil {
		return errors.Wrapf(err, "get secret %s failed", secretKey)
	}

	cfg.User, err = extractStringValue(secret, "nsxtUsername")
	if err != nil {
		return err
	}
	cfg.Password, err = extractStringValue(secret, "nsxtPassword")
	return err
}

func extractStringValue(secret *corev1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("missing secret data %s", key)
	}
	return string(value), nil
}
