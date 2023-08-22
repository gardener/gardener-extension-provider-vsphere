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

package controlplane

import (
	"context"
	"encoding/json"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/pkg/mock/controller-runtime/manager"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	apisvspherev1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
)

const (
	namespace                        = "shoot--foo--bar"
	genericTokenKubeconfigSecretName = "generic-token-kubeconfig-92e9ae14"
)

var _ = Describe("ValuesProvider", func() {
	var (
		ctrl *gomock.Controller
		c    *mockclient.MockClient
		ctx  context.Context
		mgr  *mockmanager.MockManager

		fakeClient         client.Client
		fakeSecretsManager secretsmanager.Interface

		// Build scheme
		scheme = runtime.NewScheme()
		_      = apisvsphere.AddToScheme(scheme)
		_      = apisvspherev1alpha1.AddToScheme(scheme)

		cpConfig = &apisvsphere.ControlPlaneConfig{
			CloudControllerManager: &apisvsphere.CloudControllerManagerConfig{
				FeatureGates: map[string]bool{
					"CustomResourceValidation": true,
				},
			},
			LoadBalancerClasses: []apisvsphere.CPLoadBalancerClass{{Name: "private"}},
		}

		cp = &extensionsv1alpha1.ControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "control-plane",
				Namespace: namespace,
			},
			Spec: extensionsv1alpha1.ControlPlaneSpec{
				SecretRef: corev1.SecretReference{
					Name:      v1beta1constants.SecretNameCloudProvider,
					Namespace: namespace,
				},
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					ProviderConfig: &runtime.RawExtension{
						Raw: encode(cpConfig),
					},
				},
				InfrastructureProviderStatus: &runtime.RawExtension{
					Raw: encode(&apisvsphere.InfrastructureStatus{
						NSXTInfraState: &apisvsphere.NSXTInfraState{
							SegmentName:  sp("gardener-test-network"),
							AdvancedDHCP: apisvsphere.AdvancedDHCPState{},
						},
					}),
				},
			},
		}

		cidr = "10.250.0.0/19"

		cloudprofile = &gardencorev1beta1.CloudProfile{
			Spec: gardencorev1beta1.CloudProfileSpec{
				MachineImages: []gardencorev1beta1.MachineImage{
					{
						Name: "coreos",
						Versions: []gardencorev1beta1.MachineImageVersion{
							{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "2191.5.0"}},
						},
					},
				},
				ProviderConfig: &runtime.RawExtension{
					Raw: encode(&apisvspherev1alpha1.CloudProfileConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "vsphere.provider.extensions.gardener.cloud/v1alpha1",
							Kind:       "CloudProfileConfig",
						},
						NamePrefix:                    "nameprefix",
						DefaultClassStoragePolicyName: "mypolicy",
						Regions: []apisvspherev1alpha1.RegionSpec{
							{
								Name:               "testregion",
								VsphereHost:        "vsphere.host.internal",
								VsphereInsecureSSL: true,
								NSXTHost:           "nsxt.host.internal",
								NSXTInsecureSSL:    true,
								NSXTRemoteAuth:     true,
								TransportZone:      "tz",
								LogicalTier0Router: "lt0router",
								EdgeCluster:        "edgecluster",
								SNATIPPool:         "snatIpPool",
								Datacenter:         sp("scc01-DC"),
								Datastore:          sp("A800_VMwareB"),
								Zones: []apisvspherev1alpha1.ZoneSpec{
									{
										Name:           "testzone",
										ComputeCluster: sp("scc01w01-DEV"),
									},
								},
							},
						},
						DNSServers: []string{"1.2.3.4"},
						FailureDomainLabels: &apisvspherev1alpha1.FailureDomainLabels{
							Region: "k8s-region",
							Zone:   "k8s-zone",
						},
						Constraints: apisvspherev1alpha1.Constraints{
							LoadBalancerConfig: apisvspherev1alpha1.LoadBalancerConfig{
								Size: "MEDIUM",
								Classes: []apisvspherev1alpha1.LoadBalancerClass{
									{
										Name:       "default",
										IPPoolName: sp("lbpool"),
									},
									{
										Name:              "private",
										IPPoolName:        sp("lbpool2"),
										TCPAppProfileName: sp("tcpprof2"),
									},
								},
							},
						},
						MachineImages: []apisvspherev1alpha1.MachineImages{
							{Name: "coreos",
								Versions: []apisvspherev1alpha1.MachineImageVersion{
									{
										Version: "2191.5.0",
										Path:    "gardener/templates/coreos-2191.5.0",
										GuestID: sp("coreos64Guest"),
									},
								},
							},
						},
					}),
				},
			},
		}

		cluster *extensionscontroller.Cluster

		cpSecretKey = client.ObjectKey{Namespace: namespace, Name: v1beta1constants.SecretNameCloudProvider}
		cpSecret    = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1constants.SecretNameCloudProvider,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"vsphereUsername": []byte("admin"),
				"vspherePassword": []byte("super-secret"),
				"nsxtUsername":    []byte("nsxt-lbadmin"),
				"nsxtPassword":    []byte("nsxt-lbadmin-pw"),
				"nsxtUsernameNE":  []byte("nsxt-ne"),
				"nsxtPasswordNE":  []byte("nsxt-ne-pw"),
			},
		}

		cpcSecretKey = client.ObjectKey{Namespace: namespace, Name: vsphere.CloudProviderConfig}
		cpcSecret    = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vsphere.CloudProviderConfig,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"cloudprovider.conf": []byte{}},
		}

		csiSecretKey = client.ObjectKey{Namespace: namespace, Name: vsphere.SecretCSIVsphereConfig}
		csiSecret    = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vsphere.SecretCSIVsphereConfig,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"csi-vsphere.conf": []byte(`[Global]
cluster-id = "shoot--foo--bar-garden1234"

[VirtualCenter "vsphere.host.internal"]
port = "443"
datacenters = "scc01-DC"
user = "admin"
password = "super-secret"
insecure-flag = "true"
`),
			},
		}

		checksums = map[string]string{
			v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
			vsphere.CloudProviderConfig:              "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
			vsphere.SecretCSIVsphereConfig:           "a93175a6208bed98639833cf08f616d3329884d2558c1b61cde3656f2a57b5be",
		}

		configChartValues = map[string]interface{}{
			"insecureFlag": true,
			"serverPort":   443,
			"serverName":   "vsphere.host.internal",
			"datacenters":  []string{"scc01-DC"},
			"username":     "admin",
			"password":     "super-secret",
			"loadbalancer": map[string]interface{}{
				"size":       "MEDIUM",
				"ipPoolName": "lbpool",
				"classes": []map[string]interface{}{
					{
						"name":              "private",
						"ipPoolName":        "lbpool2",
						"tcpAppProfileName": "tcpprof2",
					},
				},
				"tags": map[string]interface{}{
					"owner": "garden1234",
				},
			},
			"nsxt": map[string]interface{}{
				"password":     "nsxt-lbadmin-pw",
				"host":         "nsxt.host.internal",
				"insecureFlag": true,
				"username":     "nsxt-lbadmin",
				"remoteAuth":   true,
			},
			"labelRegion": "k8s-region",
			"labelZone":   "k8s-zone",
		}

		controlPlaneChartValues = map[string]interface{}{
			"global": map[string]interface{}{
				"genericTokenKubeconfigSecretName": "generic-token-kubeconfig-92e9ae14",
			},
			"vsphere-cloud-controller-manager": map[string]interface{}{
				"replicas":    1,
				"clusterName": "shoot--foo--bar-garden1234",
				"podNetwork":  cidr,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					"checksum/secret-" + vsphere.CloudProviderConfig:              "67234961d8244bf8bd661e1d165036e691b6570a8981a09942df2314644a8b97",
				},
				"podLabels": map[string]interface{}{
					"maintenance.gardener.cloud/restart": "true",
				},
				"featureGates": map[string]bool{
					"CustomResourceValidation": true,
				},
				"tlsCipherSuites": []string{
					"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
					"TLS_RSA_WITH_AES_128_CBC_SHA",
					"TLS_RSA_WITH_AES_256_CBC_SHA",
					"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
				},
				"secrets": map[string]interface{}{
					"server": "cloud-controller-manager-server",
				},
			},
			"csi-vsphere": map[string]interface{}{
				"replicas":       1,
				"serverName":     "vsphere.host.internal",
				"clusterID":      "shoot--foo--bar-garden1234",
				"username":       "admin",
				"password":       "super-secret",
				"serverPort":     443,
				"datacenters":    "scc01-DC",
				"insecureFlag":   "true",
				"resizerEnabled": true,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					"checksum/secret-" + vsphere.SecretCSIVsphereConfig:           "a93175a6208bed98639833cf08f616d3329884d2558c1b61cde3656f2a57b5be",
				},
				"csiSnapshotController": map[string]interface{}{
					"replicas": 1,
				},
				"csiSnapshotValidationWebhook": map[string]interface{}{
					"replicas": 1,
					"secrets": map[string]interface{}{
						"server": "csi-snapshot-validation-server",
					},
					"topologyAwareRoutingEnabled": false,
				},
				"volumesnapshots": map[string]interface{}{
					"enabled": false,
				},
				"labelRegion": "k8s-region",
				"labelZone":   "k8s-zone",
			},
		}

		controlPlaneShootChartValues = map[string]interface{}{
			"csi-vsphere": map[string]interface{}{
				"serverName":   "vsphere.host.internal",
				"clusterID":    "shoot--foo--bar-garden1234",
				"username":     "admin",
				"password":     "super-secret",
				"serverPort":   443,
				"datacenters":  "scc01-DC",
				"insecureFlag": "true",
				"labelRegion":  "k8s-region",
				"labelZone":    "k8s-zone",
				"webhookConfig": map[string]interface{}{
					"url":      "https://" + vsphere.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
					"caBundle": "",
				},
				"pspDisabled":           false,
				"kubernetesServiceHost": "api.foo.test.com",
			},
		}

		logger = log.Log.WithName("test")

		prepareValueProvider = func(cpcAndCsi bool) genericactuator.ValuesProvider {
			// Create mock client
			c = mockclient.NewMockClient(ctrl)
			if cpcAndCsi {
				c.EXPECT().Get(ctx, cpcSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpcSecret))
				c.EXPECT().Get(ctx, csiSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(csiSecret))
			}
			c.EXPECT().Get(ctx, cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))

			// Create valuesProvider
			vp := NewValuesProvider(mgr, logger, "garden1234")

			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = vp.(inject.Client).InjectClient(c)
			Expect(err).NotTo(HaveOccurred())

			return vp
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.TODO()

		cluster = &extensionscontroller.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"generic-token-kubeconfig.secret.gardener.cloud/name": genericTokenKubeconfigSecretName,
				},
			},
			CloudProfile: cloudprofile,
			Seed:         &gardencorev1beta1.Seed{},
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot--foo--bar",
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.ShootSpec{
					Region: "testregion",
					Networking: &gardencorev1beta1.Networking{
						Pods: &cidr,
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.20.0",
					},
					Provider: gardencorev1beta1.Provider{
						ControlPlaneConfig: &runtime.RawExtension{
							Raw: encode(cpConfig),
						},
						Workers: []gardencorev1beta1.Worker{
							{
								Name: "test",
							},
						},
					},
				},
				Status: gardencorev1beta1.ShootStatus{
					AdvertisedAddresses: []gardencorev1beta1.ShootAdvertisedAddress{
						{Name: "internal", URL: "https://api.foo.test.com"},
					},
				},
			},
		}

		mgr := mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetClient().Return(c)
		mgr.EXPECT().GetScheme().Return(scheme)

		fakeClient = fakeclient.NewClientBuilder().Build()
		fakeSecretsManager = fakesecretsmanager.New(fakeClient, namespace)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#GetConfigChartValues", func() {
		It("should return correct config chart values", func() {
			vp := prepareValueProvider(false)

			// Call GetConfigChartValues method and check the result
			values, err := vp.GetConfigChartValues(ctx, cp, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(configChartValues))
		})
	})

	Describe("#GetControlPlaneChartValues", func() {
		BeforeEach(func() {
			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-vsphere-controlplane", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: namespace}})).To(Succeed())
		})

		It("should return correct control plane chart values", func() {
			vp := prepareValueProvider(true)

			c.EXPECT().Delete(context.TODO(), &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "allow-kube-apiserver-to-csi-snapshot-validation", Namespace: cp.Namespace}})

			values, err := vp.GetControlPlaneChartValues(ctx, cp, cluster, fakeSecretsManager, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneChartValues))
		})

		DescribeTable("topologyAwareRoutingEnabled value",
			func(seedSettings *gardencorev1beta1.SeedSettings, shootControlPlane *gardencorev1beta1.ControlPlane, expected bool) {
				cluster.Seed = &gardencorev1beta1.Seed{
					Spec: gardencorev1beta1.SeedSpec{
						Settings: seedSettings,
					},
				}
				cluster.Shoot.Spec.ControlPlane = shootControlPlane

				vp := prepareValueProvider(true)

				c.EXPECT().Delete(context.TODO(), &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "allow-kube-apiserver-to-csi-snapshot-validation", Namespace: cp.Namespace}})

				values, err := vp.GetControlPlaneChartValues(ctx, cp, cluster, fakeSecretsManager, checksums, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).To(HaveKey("csi-vsphere"))
				Expect(values["csi-vsphere"]).To(HaveKeyWithValue("csiSnapshotValidationWebhook", HaveKeyWithValue("topologyAwareRoutingEnabled", expected)))
			},

			Entry("seed setting is nil, shoot control plane is not HA",
				nil,
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is disabled, shoot control plane is not HA",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: false}},
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is enabled, shoot control plane is not HA",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: true}},
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is nil, shoot control plane is HA with failure tolerance type 'zone'",
				nil,
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				false,
			),
			Entry("seed setting is disabled, shoot control plane is HA with failure tolerance type 'zone'",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: false}},
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				false,
			),
			Entry("seed setting is enabled, shoot control plane is HA with failure tolerance type 'zone'",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: true}},
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				true,
			),
		)
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		BeforeEach(func() {
			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-vsphere-controlplane", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: namespace}})).To(Succeed())
		})

		It("should return correct control plane shoot chart values", func() {
			vp := prepareValueProvider(false)

			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, checksums)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneShootChartValues))
		})

		Context("PodSecurityPolicy", func() {
			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is not disabled in the shoot", func() {
				cluster.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
						{
							Name: "PodSecurityPolicy",
						},
					},
				}

				controlPlaneShootChartValues = map[string]interface{}{
					"csi-vsphere": map[string]interface{}{
						"serverName":   "vsphere.host.internal",
						"clusterID":    "shoot--foo--bar-garden1234",
						"username":     "admin",
						"password":     "super-secret",
						"serverPort":   443,
						"datacenters":  "scc01-DC",
						"insecureFlag": "true",
						"labelRegion":  "k8s-region",
						"labelZone":    "k8s-zone",
						"webhookConfig": map[string]interface{}{
							"url":      "https://" + vsphere.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
							"caBundle": "",
						},
						"pspDisabled":           false,
						"kubernetesServiceHost": "api.foo.test.com",
					},
				}

				vp := prepareValueProvider(false)

				// Call GetControlPlaneChartValues method and check the result
				values, err := vp.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, checksums)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).To(Equal(controlPlaneShootChartValues))

			})
			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is disabled in the shoot", func() {
				cluster.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
						{
							Name:     "PodSecurityPolicy",
							Disabled: pointer.Bool(true),
						},
					},
				}

				controlPlaneShootChartValues = map[string]interface{}{
					"csi-vsphere": map[string]interface{}{
						"serverName":   "vsphere.host.internal",
						"clusterID":    "shoot--foo--bar-garden1234",
						"username":     "admin",
						"password":     "super-secret",
						"serverPort":   443,
						"datacenters":  "scc01-DC",
						"insecureFlag": "true",
						"labelRegion":  "k8s-region",
						"labelZone":    "k8s-zone",
						"webhookConfig": map[string]interface{}{
							"url":      "https://" + vsphere.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
							"caBundle": "",
						},
						"pspDisabled":           true,
						"kubernetesServiceHost": "api.foo.test.com",
					},
				}

				vp := prepareValueProvider(false)

				// Call GetControlPlaneChartValues method and check the result
				values, err := vp.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, checksums)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).To(Equal(controlPlaneShootChartValues))
			})
		})
	})

	Describe("#shortenID", func() {
		It("should shorten ID to given max length", func() {
			id1 := "shoot--garden--something12-cf7607c1-1b8a-11e8-8c77-fa163e4902b1"
			id2 := "shoot--garden--something123-cf7607c1-1b8a-11e8-8c77-fa163e4902b1"
			id3 := "shoot--garden--something123-cf7607c1-1b8a-11e8-8c77-fa163e4902b2"
			id4 := "shoot--garden--something1234-cf7607c1-1b8a-11e8-8c77-fa163e4902b1"

			short1 := shortenID(id1, 63)
			short2 := shortenID(id2, 63)
			short3 := shortenID(id3, 63)
			short4 := shortenID(id4, 63)
			Expect(short1).To(Equal(id1))
			Expect(short2).To(Equal("shoot--garden--something123-cf7607c1-1b8a-11e8-8c7-qksvc0j2gs99"))
			Expect(len(short2)).To(Equal(63))
			Expect(short3).To(Equal("shoot--garden--something123-cf7607c1-1b8a-11e8-8c7-qksvc0j2gs9a"))
			Expect(short4).To(Equal("shoot--garden--something1234-cf7607c1-1b8a-11e8-8c-8wzf59wac3mj"))
			Expect(len(short4)).To(Equal(63))
		})
	})

	Describe("#GetControlPlaneShootCRDsChartValues", func() {
		It("should return correct control plane shoot CRDs chart values", func() {
			vp := NewValuesProvider(mgr, logger, "garden1234")

			values, err := vp.GetControlPlaneShootCRDsChartValues(ctx, cp, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"volumesnapshots": map[string]interface{}{"enabled": false},
			}))
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func clientGet(result runtime.Object) interface{} {
	return func(ctx context.Context, key client.ObjectKey, obj runtime.Object, _ ...client.GetOption) error {
		switch obj.(type) {
		case *corev1.Secret:
			*obj.(*corev1.Secret) = *result.(*corev1.Secret)
		}
		return nil
	}
}

func sp(s string) *string {
	return &s
}
