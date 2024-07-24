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

package worker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	mockkubernetes "github.com/gardener/gardener/pkg/client/kubernetes/mock"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	vspherev1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	. "github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/worker"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
)

// TODO martin adapt/fix test

var _ = Describe("Machines", func() {
	var (
		ctrl           *gomock.Controller
		c              *mockclient.MockClient
		statusWriter   *mockclient.MockStatusWriter
		chartApplier   *mockkubernetes.MockChartApplier
		workerDelegate genericworkeractuator.WorkerDelegate
		scheme         *runtime.Scheme
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		c = mockclient.NewMockClient(ctrl)
		statusWriter = mockclient.NewMockStatusWriter(ctrl)
		chartApplier = mockkubernetes.NewMockChartApplier(ctrl)

		scheme = runtime.NewScheme()
		_ = api.AddToScheme(scheme)
		_ = apiv1alpha1.AddToScheme(scheme)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("workerDelegate", func() {
		BeforeEach(func() {
			workerDelegate, _ = NewWorkerDelegate(nil, scheme, nil, "", nil, nil)
		})

		Describe("#GenerateMachineDeployments, #DeployMachineClasses", func() {
			var (
				namespace        string
				cloudProfileName string

				host         string
				username     string
				password     string
				nsxtUsername string
				nsxtPassword string
				insecureSSL  bool
				region       string

				machineImageName    string
				machineImageVersion string
				machineImagePath    string

				machineType   string
				networkName   string
				datacenter    string
				resourcePool  string
				datastore     string
				resourcePool2 string
				datastore2    string
				folder        string
				sshKey        string
				userData      = []byte("some-user-data")

				namePool1           string
				minPool1            int32
				maxPool1            int32
				maxSurgePool1       intstr.IntOrString
				maxUnavailablePool1 intstr.IntOrString

				namePool2           string
				minPool2            int32
				maxPool2            int32
				maxSurgePool2       intstr.IntOrString
				maxUnavailablePool2 intstr.IntOrString

				zone1 string
				zone2 string

				machineConfiguration *machinev1alpha1.MachineConfiguration

				switch1 string
				switch2 string

				workerPoolHash1 string
				workerPoolHash2 string

				shootVersionMajorMinor string
				shootVersion           string
				cluster                *extensionscontroller.Cluster
				w                      *extensionsv1alpha1.Worker
			)

			BeforeEach(func() {
				namespace = "shoot--foobar--vsphere"
				cloudProfileName = "vsphere"

				region = "testregion"
				host = "vsphere.host.internal"
				username = "myuser"
				password = "mypassword"
				insecureSSL = true
				nsxtUsername = "nsxtuser"
				nsxtPassword = "nsxtpassword"

				machineImageName = "my-os"
				machineImageVersion = "123"
				machineImagePath = "templates/my-template"

				machineType = "mt1"
				datacenter = "my-dc"
				resourcePool = "my-pool"
				datastore = "my-ds"
				resourcePool2 = "my-pool2"
				datastore2 = "my-ds2"
				folder = "my-folder"
				networkName = "mynetwork"
				sshKey = "aaabbbcccddd"
				userData = []byte("some-user-data")

				namePool1 = "pool-1"
				minPool1 = 5
				maxPool1 = 10
				maxSurgePool1 = intstr.FromInt(3)
				maxUnavailablePool1 = intstr.FromInt(2)

				namePool2 = "pool-2"
				minPool2 = 30
				maxPool2 = 45
				maxSurgePool2 = intstr.FromInt(10)
				maxUnavailablePool2 = intstr.FromInt(15)

				zone1 = "testregion-a"
				zone2 = "testregion-b"

				machineConfiguration = &machinev1alpha1.MachineConfiguration{}

				switch1 = "switch1"
				switch2 = "switch2"

				shootVersionMajorMinor = "1.2"
				shootVersion = shootVersionMajorMinor + ".3"

				images := []apiv1alpha1.MachineImages{
					{
						Name: machineImageName,
						Versions: []apiv1alpha1.MachineImageVersion{
							{
								Version: machineImageVersion,
								Path:    machineImagePath,
							},
						},
					},
				}
				cluster = createCluster(cloudProfileName, shootVersion, images)

				w = &extensionsv1alpha1.Worker{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
					},
					Spec: extensionsv1alpha1.WorkerSpec{
						SecretRef: corev1.SecretReference{
							Name:      "secret",
							Namespace: namespace,
						},
						Region: region,
						InfrastructureProviderStatus: &runtime.RawExtension{
							Raw: encode(&apiv1alpha1.InfrastructureStatus{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "vsphere.provider.extensions.gardener.cloud/v1alpha1",
									Kind:       "InfrastructureStatus",
								},
								VsphereConfig: apiv1alpha1.VsphereConfig{
									Folder: folder,
									Region: region,
									ZoneConfigs: map[string]apiv1alpha1.ZoneConfig{
										"testregion-a": {
											Datacenter:   datacenter,
											Datastore:    datastore,
											ResourcePool: resourcePool,
											SwitchUUID:   switch1,
										},
										"testregion-b": {
											Datacenter:   datacenter,
											Datastore:    datastore2,
											ResourcePool: resourcePool2,
											SwitchUUID:   switch2,
										},
									},
								},
								NSXTInfraState: &apiv1alpha1.NSXTInfraState{
									SegmentName:  &networkName,
									AdvancedDHCP: apiv1alpha1.AdvancedDHCPState{},
								},
							}),
						},
						Pools: []extensionsv1alpha1.WorkerPool{
							{
								Name:           namePool1,
								Minimum:        minPool1,
								Maximum:        maxPool1,
								MaxSurge:       maxSurgePool1,
								MaxUnavailable: maxUnavailablePool1,
								MachineType:    machineType,
								MachineImage: extensionsv1alpha1.MachineImage{
									Name:    machineImageName,
									Version: machineImageVersion,
								},
								UserData: userData,
								Zones: []string{
									zone1,
									zone2,
								},
							},
							{
								Name:           namePool2,
								Minimum:        minPool2,
								Maximum:        maxPool2,
								MaxSurge:       maxSurgePool2,
								MaxUnavailable: maxUnavailablePool2,
								MachineType:    machineType,
								MachineImage: extensionsv1alpha1.MachineImage{
									Name:    machineImageName,
									Version: machineImageVersion,
								},
								UserData: userData,
								Zones: []string{
									zone1,
									zone2,
								},
							},
						},
						SSHPublicKey: []byte(sshKey),
					},
				}

				workerPoolHash1, _ = worker.WorkerPoolHash(w.Spec.Pools[0], cluster)
				workerPoolHash2, _ = worker.WorkerPoolHash(w.Spec.Pools[1], cluster)

				workerDelegate, _ = NewWorkerDelegate(c, scheme, chartApplier, "", w, cluster)
			})

			It("should return the expected machine deployments", func() {
				expectGetSecretCallToWork(c, username, password, nsxtUsername, nsxtPassword)

				// Test workerDelegate.DeployMachineClasses()
				defaultMachineClass := map[string]interface{}{
					"region":       region,
					"resourcePool": resourcePool,
					"datacenter":   datacenter,
					"datastore":    datastore,
					"folder":       folder,
					"network":      networkName,
					"memory":       4096,
					"numCpus":      2,
					"systemDisk": map[string]interface{}{
						"size": 20,
					},
					"templateVM": machineImagePath,
					"sshKeys":    []string{sshKey},
					"tags": map[string]string{
						"mcm.gardener.cloud/cluster": namespace,
						"mcm.gardener.cloud/role":    "node",
					},
					"secret": map[string]interface{}{
						"cloudConfig": string(userData),
					},
					"credentialsSecretRef": map[string]interface{}{
						"name":      w.Spec.SecretRef.Name,
						"namespace": w.Spec.SecretRef.Namespace,
					},
				}

				machineClassNamePool1Zone1 := fmt.Sprintf("%s-%s-z1", namespace, namePool1)
				machineClassNamePool1Zone2 := fmt.Sprintf("%s-%s-z2", namespace, namePool1)
				machineClassNamePool2Zone1 := fmt.Sprintf("%s-%s-z1", namespace, namePool2)
				machineClassNamePool2Zone2 := fmt.Sprintf("%s-%s-z2", namespace, namePool2)

				machineClassPool1Zone1 := prepareMachineClass(defaultMachineClass, machineClassNamePool1Zone1, resourcePool, datastore, workerPoolHash1, switch1, host, username, password, insecureSSL)
				machineClassPool1Zone2 := prepareMachineClass(defaultMachineClass, machineClassNamePool1Zone2, resourcePool2, datastore2, workerPoolHash1, switch2, host, username, password, insecureSSL)
				machineClassPool2Zone1 := prepareMachineClass(defaultMachineClass, machineClassNamePool2Zone1, resourcePool, datastore, workerPoolHash2, switch1, host, username, password, insecureSSL)
				machineClassPool2Zone2 := prepareMachineClass(defaultMachineClass, machineClassNamePool2Zone2, resourcePool2, datastore2, workerPoolHash2, switch2, host, username, password, insecureSSL)

				machineClassWithHashPool1Zone1 := machineClassPool1Zone1["name"].(string)
				machineClassWithHashPool1Zone2 := machineClassPool1Zone2["name"].(string)
				machineClassWithHashPool2Zone1 := machineClassPool2Zone1["name"].(string)
				machineClassWithHashPool2Zone2 := machineClassPool2Zone2["name"].(string)

				chartApplier.
					EXPECT().
					ApplyFromEmbeddedFS(
						context.TODO(),
						filepath.Join(vsphere.InternalChartsPath, "machineclass"),
						namespace,
						"machineclass",
						kubernetes.Values(map[string]interface{}{"machineClasses": []map[string]interface{}{
							machineClassPool1Zone1,
							machineClassPool1Zone2,
							machineClassPool2Zone1,
							machineClassPool2Zone2,
						}}),
					).
					Return(nil)

				err := workerDelegate.DeployMachineClasses(context.TODO())
				Expect(err).NotTo(HaveOccurred())

				// Test workerDelegate.UpdateMachineDeployments()
				expectedImages := &apiv1alpha1.WorkerStatus{
					TypeMeta: metav1.TypeMeta{
						APIVersion: vspherev1alpha1.SchemeGroupVersion.String(),
						Kind:       "WorkerStatus",
					},
					MachineImages: []vspherev1alpha1.MachineImage{
						{
							Name:    machineImageName,
							Version: machineImageVersion,
							Path:    machineImagePath,
						},
					},
				}

				workerWithExpectedImages := w.DeepCopy()
				workerWithExpectedImages.Status.ProviderStatus = &runtime.RawExtension{
					Object: expectedImages,
				}

				ctx := context.TODO()
				c.EXPECT().Status().Return(statusWriter)
				statusWriter.EXPECT().Patch(ctx, workerWithExpectedImages, gomock.Any()).Return(nil)

				err = workerDelegate.UpdateMachineImagesStatus(ctx)
				Expect(err).NotTo(HaveOccurred())

				labelsZone1 := map[string]string{vsphere.CSITopologyRegionKey: region, vsphere.CSITopologyZoneKey: zone1}
				labelsZone2 := map[string]string{vsphere.CSITopologyRegionKey: region, vsphere.CSITopologyZoneKey: zone2}
				// Test workerDelegate.GenerateMachineDeployments()
				machineDeployments := worker.MachineDeployments{
					{
						Name:                 machineClassNamePool1Zone1,
						ClassName:            machineClassWithHashPool1Zone1,
						SecretName:           machineClassWithHashPool1Zone1,
						Minimum:              worker.DistributeOverZones(0, minPool1, 2),
						Maximum:              worker.DistributeOverZones(0, maxPool1, 2),
						MaxSurge:             worker.DistributePositiveIntOrPercent(0, maxSurgePool1, 2, maxPool1),
						MaxUnavailable:       worker.DistributePositiveIntOrPercent(0, maxUnavailablePool1, 2, minPool1),
						Labels:               labelsZone1,
						MachineConfiguration: machineConfiguration,
					},
					{
						Name:                 machineClassNamePool1Zone2,
						ClassName:            machineClassWithHashPool1Zone2,
						SecretName:           machineClassWithHashPool1Zone2,
						Minimum:              worker.DistributeOverZones(1, minPool1, 2),
						Maximum:              worker.DistributeOverZones(1, maxPool1, 2),
						MaxSurge:             worker.DistributePositiveIntOrPercent(1, maxSurgePool1, 2, maxPool1),
						MaxUnavailable:       worker.DistributePositiveIntOrPercent(1, maxUnavailablePool1, 2, minPool1),
						Labels:               labelsZone2,
						MachineConfiguration: machineConfiguration,
					},
					{
						Name:                 machineClassNamePool2Zone1,
						ClassName:            machineClassWithHashPool2Zone1,
						SecretName:           machineClassWithHashPool2Zone1,
						Minimum:              worker.DistributeOverZones(0, minPool2, 2),
						Maximum:              worker.DistributeOverZones(0, maxPool2, 2),
						MaxSurge:             worker.DistributePositiveIntOrPercent(0, maxSurgePool2, 2, maxPool1),
						MaxUnavailable:       worker.DistributePositiveIntOrPercent(0, maxUnavailablePool2, 2, minPool1),
						Labels:               labelsZone1,
						MachineConfiguration: machineConfiguration,
					},
					{
						Name:                 machineClassNamePool2Zone2,
						ClassName:            machineClassWithHashPool2Zone2,
						SecretName:           machineClassWithHashPool2Zone2,
						Minimum:              worker.DistributeOverZones(1, minPool2, 2),
						Maximum:              worker.DistributeOverZones(1, maxPool2, 2),
						MaxSurge:             worker.DistributePositiveIntOrPercent(1, maxSurgePool2, 2, maxPool1),
						MaxUnavailable:       worker.DistributePositiveIntOrPercent(1, maxUnavailablePool2, 2, minPool1),
						Labels:               labelsZone2,
						MachineConfiguration: machineConfiguration,
					},
				}

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(machineDeployments))
			})

			It("should fail because the secret cannot be read", func() {
				c.EXPECT().
					Get(context.TODO(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
					Return(fmt.Errorf("error"))

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the version is invalid", func() {
				expectGetSecretCallToWork(c, username, password, nsxtUsername, nsxtPassword)

				cluster.Shoot.Spec.Kubernetes.Version = "invalid"
				workerDelegate, _ = NewWorkerDelegate(c, scheme, chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the infrastructure status cannot be decoded", func() {
				expectGetSecretCallToWork(c, username, password, nsxtUsername, nsxtPassword)

				w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Raw: []byte(`invalid`)}

				workerDelegate, _ = NewWorkerDelegate(c, scheme, chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the machine image cannot be found", func() {
				expectGetSecretCallToWork(c, username, password, nsxtUsername, nsxtPassword)

				invalidImages := []apiv1alpha1.MachineImages{
					{
						Name: "xxname",
						Versions: []apiv1alpha1.MachineImageVersion{
							{
								Version: "xxversion",
								Path:    "xxpath",
							},
						},
					},
				}
				clusterWithoutImages := createCluster(cloudProfileName, shootVersion, invalidImages)

				workerDelegate, _ = NewWorkerDelegate(c, scheme, chartApplier, "", w, clusterWithoutImages)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should set expected machineControllerManager settings on machine deployment", func() {
				expectGetSecretCallToWork(c, username, password, nsxtUsername, nsxtPassword)

				testDrainTimeout := metav1.Duration{Duration: 10 * time.Minute}
				testHealthTimeout := metav1.Duration{Duration: 20 * time.Minute}
				testCreationTimeout := metav1.Duration{Duration: 30 * time.Minute}
				testMaxEvictRetries := int32(30)
				testNodeConditions := []string{"ReadonlyFilesystem", "KernelDeadlock", "DiskPressure"}
				w.Spec.Pools[0].MachineControllerManagerSettings = &gardencorev1beta1.MachineControllerManagerSettings{
					MachineDrainTimeout:    &testDrainTimeout,
					MachineCreationTimeout: &testCreationTimeout,
					MachineHealthTimeout:   &testHealthTimeout,
					MaxEvictRetries:        &testMaxEvictRetries,
					NodeConditions:         testNodeConditions,
				}

				workerDelegate, _ = NewWorkerDelegate(c, scheme, chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				resultSettings := result[0].MachineConfiguration
				resultNodeConditions := strings.Join(testNodeConditions, ",")

				Expect(err).NotTo(HaveOccurred())
				Expect(resultSettings.MachineDrainTimeout).To(Equal(&testDrainTimeout))
				Expect(resultSettings.MachineCreationTimeout).To(Equal(&testCreationTimeout))
				Expect(resultSettings.MachineHealthTimeout).To(Equal(&testHealthTimeout))
				Expect(resultSettings.MaxEvictRetries).To(Equal(&testMaxEvictRetries))
				Expect(resultSettings.NodeConditions).To(Equal(&resultNodeConditions))
			})
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func expectGetSecretCallToWork(c *mockclient.MockClient, username, password string, nsxtUsername, nsxtPassword string) {
	c.EXPECT().
		Get(context.TODO(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, secret *corev1.Secret, _ ...client.GetOption) error {
			secret.Data = map[string][]byte{
				vsphere.Username:     []byte(username),
				vsphere.Password:     []byte(password),
				vsphere.NSXTUsername: []byte(nsxtUsername),
				vsphere.NSXTPassword: []byte(nsxtPassword),
			}
			return nil
		})
}

func createCluster(cloudProfileName, shootVersion string, images []apiv1alpha1.MachineImages) *extensionscontroller.Cluster {
	cloudProfileConfig := &apiv1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
			Kind:       "CloudProfileConfig",
		},
		NamePrefix:                    "nameprefix",
		DefaultClassStoragePolicyName: "mypolicy",
		DNSServers:                    []string{"1.2.3.4"},
		Regions: []apiv1alpha1.RegionSpec{
			{
				Name:               "testregion",
				VsphereHost:        "vsphere.host.internal",
				VsphereInsecureSSL: true,
				NSXTHost:           "nsxt.host.internal",
				NSXTInsecureSSL:    true,
				TransportZone:      "tz",
				LogicalTier0Router: "lt0router",
				EdgeCluster:        "edgecluster",
				SNATIPPool:         "snatIpPool",
				Datacenter:         sp("scc01-DC"),
				Datastore:          sp("A800_VMwareB"),
				Zones: []apiv1alpha1.ZoneSpec{
					{
						Name:           "testregion-a",
						ComputeCluster: sp("scc01w01-DEV-A"),
					},
					{
						Name:           "testregion-b",
						ComputeCluster: sp("scc01w01-DEV-B"),
					},
				},
			},
		},
		Constraints: apiv1alpha1.Constraints{
			LoadBalancerConfig: apiv1alpha1.LoadBalancerConfig{
				Size: "SMALL",
				Classes: []apiv1alpha1.LoadBalancerClass{
					{
						Name:       "default",
						IPPoolName: sp("lbpool"),
					},
				},
			},
		},
		MachineImages: images,
	}
	cloudProfileConfigJSON, _ := json.Marshal(cloudProfileConfig)
	cluster := &extensionscontroller.Cluster{
		CloudProfile: &gardencorev1beta1.CloudProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: cloudProfileName,
			},
			Spec: gardencorev1beta1.CloudProfileSpec{
				ProviderConfig: &runtime.RawExtension{
					Raw: cloudProfileConfigJSON,
				},
				Regions: []gardencorev1beta1.Region{
					{
						Name: "testregion",
						Zones: []gardencorev1beta1.AvailabilityZone{
							{Name: "testregion-a"},
							{Name: "testregion-b"},
						},
					},
				},
				MachineTypes: []gardencorev1beta1.MachineType{
					{
						Name:   "mt1",
						Memory: resource.MustParse("4096Mi"),
						CPU:    resource.MustParse("2"),
					},
				},
			},
		},
		Shoot: &gardencorev1beta1.Shoot{
			Spec: gardencorev1beta1.ShootSpec{
				Region: "testregion",
				Kubernetes: gardencorev1beta1.Kubernetes{
					Version: shootVersion,
				},
			},
		},
	}

	specImages := []gardencorev1beta1.MachineImage{}
	for _, image := range images {
		specImages = append(specImages, gardencorev1beta1.MachineImage{
			Name: image.Name,
			Versions: []gardencorev1beta1.MachineImageVersion{
				{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: image.Versions[0].Version}},
			},
		})
	}
	cluster.CloudProfile.Spec.MachineImages = specImages

	return cluster
}

func prepareMachineClass(defaultMachineClass map[string]interface{}, machineClassName, resourcePool, datastore, workerPoolHash, switchUUID, host, username, password string, insecureSSL bool) map[string]interface{} {
	out := make(map[string]interface{}, len(defaultMachineClass)+10)

	for k, v := range defaultMachineClass {
		out[k] = v
	}

	out["resourcePool"] = resourcePool
	out["datastore"] = datastore
	out["name"] = fmt.Sprintf("%s-%s", machineClassName, workerPoolHash)
	out["switchUuid"] = switchUUID
	out["secret"].(map[string]interface{})[vsphere.Host] = host
	out["secret"].(map[string]interface{})[vsphere.Username] = username
	out["secret"].(map[string]interface{})[vsphere.Password] = password
	out["secret"].(map[string]interface{})[vsphere.InsecureSSL] = strconv.FormatBool(insecureSSL)

	return out
}

func sp(s string) *string {
	return &s
}
