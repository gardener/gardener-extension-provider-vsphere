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

	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	apisvspherev1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	namespace = "shoot--foo--bar"
)

var _ = Describe("ValuesProvider", func() {
	var (
		ctrl *gomock.Controller

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

		cidr    = "10.250.0.0/19"
		cluster = &extensionscontroller.Cluster{
			CloudProfile: &gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: "coreos",
							Versions: []gardencorev1beta1.ExpirableVersion{
								{
									Version: "2191.5.0",
								},
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
			},
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot--foo--bar",
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.ShootSpec{
					Region: "testregion",
					Networking: gardencorev1beta1.Networking{
						Pods: &cidr,
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.14.0",
					},
					Provider: gardencorev1beta1.Provider{
						ControlPlaneConfig: &runtime.RawExtension{
							Raw: encode(cpConfig),
						},
					},
				},
			},
		}

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
			vsphere.CloudControllerManagerName:       "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
			vsphere.CloudControllerManagerServerName: "6dff2a2e6f14444b66d8e4a351c049f7e89ee24ba3eaab95dbec40ba6bdebb52",
			vsphere.CSIAttacherName:                  "2da58ad61c401a2af779a909d22fb42eed93a1524cbfdab974ceedb413fcb914",
			vsphere.CSIProvisionerName:               "f75b42d40ab501428c383dfb2336cb1fc892bbee1fc1d739675171e4acc4d911",
			vsphere.CSIResizerName:                   "a77e663ba1af340fb3dd7f6f8a1be47c7aa9e658198695480641e6b934c0b9ed",
			vsphere.SecretCSIVsphereConfig:           "a93175a6208bed98639833cf08f616d3329884d2558c1b61cde3656f2a57b5be",
			vsphere.VsphereCSIController:             "6666666666",
			vsphere.VsphereCSISyncer:                 "7777777777",
		}

		configChartValues = map[string]interface{}{
			"insecureFlag": true,
			"serverPort":   443,
			"serverName":   "vsphere.host.internal",
			"datacenters":  "scc01-DC",
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
				"passwordNE":   "nsxt-ne-pw",
				"host":         "nsxt.host.internal",
				"insecureFlag": true,
				"username":     "nsxt-lbadmin",
				"usernameNE":   "nsxt-ne",
				"remoteAuth":   true,
			},
			"labelRegion": "k8s-region",
			"labelZone":   "k8s-zone",
		}

		controlPlaneChartValues = map[string]interface{}{
			"vsphere-cloud-controller-manager": map[string]interface{}{
				"replicas":          1,
				"kubernetesVersion": "1.14.0",
				"clusterName":       "shoot--foo--bar-garden1234",
				"podNetwork":        cidr,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-" + vsphere.CloudControllerManagerName:       "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
					"checksum/secret-" + vsphere.CloudControllerManagerServerName: "6dff2a2e6f14444b66d8e4a351c049f7e89ee24ba3eaab95dbec40ba6bdebb52",
					"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					"checksum/configmap-" + vsphere.CloudProviderConfig:           "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
				},
				"podLabels": map[string]interface{}{
					"maintenance.gardener.cloud/restart": "true",
				},
				"featureGates": map[string]bool{
					"CustomResourceValidation": true,
				},
			},
			"csi-vsphere": map[string]interface{}{
				"replicas":          1,
				"kubernetesVersion": "1.14.0",
				"serverName":        "vsphere.host.internal",
				"clusterID":         "shoot--foo--bar-garden1234",
				"username":          "admin",
				"password":          "super-secret",
				"serverPort":        443,
				"datacenters":       "scc01-DC",
				"insecureFlag":      "true",
				"resizerEnabled":    true,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-" + vsphere.CSIProvisionerName:               "f75b42d40ab501428c383dfb2336cb1fc892bbee1fc1d739675171e4acc4d911",
					"checksum/secret-" + vsphere.CSIAttacherName:                  "2da58ad61c401a2af779a909d22fb42eed93a1524cbfdab974ceedb413fcb914",
					"checksum/secret-" + vsphere.CSIResizerName:                   "a77e663ba1af340fb3dd7f6f8a1be47c7aa9e658198695480641e6b934c0b9ed",
					"checksum/secret-" + vsphere.VsphereCSIController:             "6666666666",
					"checksum/secret-" + vsphere.VsphereCSISyncer:                 "7777777777",
					"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					"checksum/secret-" + vsphere.SecretCSIVsphereConfig:           "a93175a6208bed98639833cf08f616d3329884d2558c1b61cde3656f2a57b5be",
				},
				"labelRegion": "k8s-region",
				"labelZone":   "k8s-zone",
			},
		}

		controlPlaneShootChartValues = map[string]interface{}{
			"csi-vsphere": map[string]interface{}{
				"serverName":        "vsphere.host.internal",
				"clusterID":         "shoot--foo--bar-garden1234",
				"username":          "admin",
				"password":          "super-secret",
				"serverPort":        443,
				"datacenters":       "scc01-DC",
				"insecureFlag":      "true",
				"kubernetesVersion": "1.14.0",
				"labelRegion":       "k8s-region",
				"labelZone":         "k8s-zone",
			},
		}

		logger = log.Log.WithName("test")

		prepareValueProvider = func(csi bool) genericactuator.ValuesProvider {
			// Create mock client
			client := mockclient.NewMockClient(ctrl)
			if csi {
				client.EXPECT().Get(context.TODO(), csiSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(csiSecret))
			}
			client.EXPECT().Get(context.TODO(), cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))

			// Create valuesProvider
			vp := NewValuesProvider(logger, "garden1234")
			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = vp.(inject.Client).InjectClient(client)
			Expect(err).NotTo(HaveOccurred())

			return vp
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#GetConfigChartValues", func() {
		It("should return correct config chart values", func() {
			vp := prepareValueProvider(false)

			// Call GetConfigChartValues method and check the result
			values, err := vp.GetConfigChartValues(context.TODO(), cp, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(configChartValues))
		})
	})

	Describe("#GetControlPlaneChartValues", func() {
		It("should return correct control plane chart values", func() {
			vp := prepareValueProvider(true)

			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values["csi-vsphere"]).To(Equal(controlPlaneChartValues["csi-vsphere"]))
			Expect(values["vsphere-cloud-controller-manager"]).To(Equal(controlPlaneChartValues["vsphere-cloud-controller-manager"]))
			Expect(values).To(Equal(controlPlaneChartValues))
		})
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		It("should return correct control plane shoot chart values", func() {
			vp := prepareValueProvider(false)

			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, checksums)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneShootChartValues))
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
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func clientGet(result runtime.Object) interface{} {
	return func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
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
