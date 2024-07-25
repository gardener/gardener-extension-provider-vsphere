/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 *
 */

package worker

import (
	"context"
	"fmt"
	"path/filepath"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils/chart"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
)

var (
	mcmChart = &chart.Chart{
		Name:   vsphere.MachineControllerManagerName,
		Path:   filepath.Join(vsphere.InternalChartsPath, vsphere.MachineControllerManagerName, "seed"),
		Images: []string{vsphere.MachineControllerManagerImageName, vsphere.MCMProviderVsphereImageName},
		Objects: []*chart.Object{
			{Type: &appsv1.Deployment{}, Name: vsphere.MachineControllerManagerName},
			{Type: &corev1.Service{}, Name: vsphere.MachineControllerManagerName},
			{Type: &corev1.ServiceAccount{}, Name: vsphere.MachineControllerManagerName},
			{Type: &corev1.Secret{}, Name: vsphere.MachineControllerManagerName},
			{Type: extensionscontroller.GetVerticalPodAutoscalerObject(), Name: vsphere.MachineControllerManagerVpaName},
			{Type: &corev1.ConfigMap{}, Name: vsphere.MachineControllerManagerMonitoringConfigName},
		},
	}

	mcmShootChart = &chart.Chart{
		Name: vsphere.MachineControllerManagerName,
		Path: filepath.Join(vsphere.InternalChartsPath, vsphere.MachineControllerManagerName, "shoot"),
		Objects: []*chart.Object{
			{Type: &rbacv1.ClusterRole{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", vsphere.Name, vsphere.MachineControllerManagerName)},
			{Type: &rbacv1.ClusterRoleBinding{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", vsphere.Name, vsphere.MachineControllerManagerName)},
		},
	}
)

func (w *workerDelegate) GetMachineControllerManagerChartValues(ctx context.Context) (map[string]interface{}, error) {
	namespace := &corev1.Namespace{}
	namespaceKey := client.ObjectKey{
		Namespace: w.worker.Namespace,
		Name:      "",
	}

	if err := w.client.Get(ctx, namespaceKey, namespace); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"providerName": vsphere.Name,
		"namespace": map[string]interface{}{
			"uid": namespace.UID,
		},
		"podLabels": map[string]interface{}{
			v1beta1constants.LabelPodMaintenanceRestart: "true",
		},
	}, nil
}

func (w *workerDelegate) GetMachineControllerManagerShootChartValues(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"providerName": vsphere.Name,
	}, nil
}

// GetMachineControllerManagerCloudCredentials should return the IaaS credentials
// with the secret keys used by the machine-controller-manager.
func (w *workerDelegate) GetMachineControllerManagerCloudCredentials(ctx context.Context) (map[string][]byte, error) {
	return w.generateMachineClassSecretData(ctx)
}
