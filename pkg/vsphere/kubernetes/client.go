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

package kubernetes

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vsphere2 "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme *runtime.Scheme
var clientCache *cache.LRUExpireCache

func init() {
	scheme = runtime.NewScheme()
	corev1.AddToScheme(scheme)

	clientCache = cache.NewLRUExpireCache(50)
}

func hashMD5(in []byte) string {
	return fmt.Sprintf("%x", md5.Sum(in))
}

// CreateVsphereKubernetesClient creates a kubernetes client
func CreateVsphereKubernetesClient(kubeconfig []byte) (ctrlClient.Client, error) {
	hash := "kubeClient:" + hashMD5(kubeconfig)
	if value, ok := clientCache.Get(hash); ok {
		return value.(ctrlClient.Client), nil
	}

	client, err := createRealVsphereKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientCache.Add(hash, client, 1*time.Hour)
	return client, nil
}

func createRealVsphereKubernetesClient(kubeconfig []byte) (ctrlClient.Client, error) {
	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", func() (*clientcmdapi.Config, error) {
		return clientcmd.Load([]byte(kubeconfig))
	})
	if err != nil {
		return nil, fmt.Errorf("build config from kubeconfig failed: %w", err)
	}

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	client, err := ctrlClient.New(
		rest.AddUserAgent(config, "machine-controller-manager-provider-vsphere"),
		ctrlClient.Options{
			Scheme: scheme,
		})

	return client, err
}

// GetVsphereAPISession gets a vsphere-api-session from cache or creates a new one
func GetVsphereAPISession(region vsphere.K8sRegionSpec, userpass vsphere2.UserPass) (*VsphereAPIClient, error) {
	key := fmt.Sprintf("%s\t%s\t%s", region.VsphereHost, userpass.Username, userpass.Password)
	hash := "VsphereAPIClient:" + hashMD5([]byte(key))
	if value, ok := clientCache.Get(hash); ok {
		return value.(*VsphereAPIClient), nil
	}

	apiClient, err := newVsphereAPIClient(region, userpass.Username, userpass.Password)
	if err != nil {
		return nil, err
	}

	clientCache.Add(hash, apiClient, 1*time.Hour)
	return apiClient, nil
}
