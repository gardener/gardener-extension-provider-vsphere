/*
 * Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package withkubernetes

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyServiceAccount applies service account chart and returns token
func ApplyServiceAccount(ctx context.Context, client ctrlClient.Client, config *rest.Config, chartPath string, releaseName, namespace string) error {
	applier, err := kubernetes.NewChartApplierForConfig(config)
	if err != nil {
		return fmt.Errorf("preparing chart applier failed: %w", err)
	}

	err = applier.Apply(ctx, chartPath, namespace, releaseName)
	if err != nil {
		return fmt.Errorf("applying chart %s failed: %w", chartPath, err)
	}
	return nil
}

// DeleteServiceAccount deletes service account chart
func DeleteServiceAccount(ctx context.Context, config *rest.Config, chartPath string, releaseName, namespace string) error {
	applier, err := kubernetes.NewChartApplierForConfig(config)
	if err != nil {
		return fmt.Errorf("preparing chart applier failed: %w", err)
	}

	err = applier.Delete(ctx, chartPath, namespace, releaseName)
	if err != nil {
		return fmt.Errorf("deleting chart %s failed: %w", chartPath, err)
	}
	return nil
}

// GetServiceAccountToken retrieves service account token
func GetServiceAccountToken(ctx context.Context, client ctrlClient.Client, name ctrlClient.ObjectKey) (string, error) {
	obj := &corev1.ServiceAccount{}
	err := client.Get(ctx, name, obj)
	if err != nil {
		return "", fmt.Errorf("geeting service account %s failed: %w", name, err)
	}

	if len(obj.Secrets) == 0 {
		return "", fmt.Errorf("missing service account secret")
	}

	secret := &corev1.Secret{}
	secretName := ctrlClient.ObjectKey{
		Namespace: name.Namespace,
		Name:      obj.Secrets[0].Name,
	}
	err = client.Get(ctx, secretName, secret)
	if err != nil {
		return "", fmt.Errorf("getting service account secret failed: %w", err)
	}

	if secret.Data == nil || secret.Data["token"] == nil {
		return "", fmt.Errorf("getting service account token from secret %s failed", obj.Secrets[0].Name)
	}

	return string(secret.Data["token"]), nil
}
