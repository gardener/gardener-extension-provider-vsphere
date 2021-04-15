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

package infrastructure

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	vsphereinstall "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/install"
	vspherev1alpha1 "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1"
	controllerinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/ensurer"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	vapi_errors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	vapiclient "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/ip_pools"
	t1nat "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/tier_1s/nat"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	nsxtHost     = flag.String("nsxt-host", "", "NSX-T manager host")
	nsxtUsername = flag.String("nsxt-username", "admin", "NSX-T manager user name")
	nsxtPassword = flag.String("nsxt-password", "", "NSX-T manager password")

	edgeClusterName   = flag.String("nsxt-edge-cluster", "", "NSX-T edge cluster name")
	transportZoneName = flag.String("nsxt-transport-zone", "", "NSX-T transport zone name")
	tier0GatewayName  = flag.String("nsxt-t0-gateway", "", "NSX-T T0 gateway name")
	snatIPPoolName    = flag.String("nsxt-snat-ip-pool", "", "NSX-T SNAT IP pool name")
)

func validateFlags() {
	if len(*nsxtHost) == 0 {
		panic("--nsxt-host flag is not specified")
	}
	if len(*nsxtUsername) == 0 {
		panic("--nsxt-username flag is not specified")
	}
	if len(*nsxtPassword) == 0 {
		panic("--nsxt-password flag is not specified")
	}
	if len(*edgeClusterName) == 0 {
		panic("--nsxt-edge-cluster flag is not specified")
	}
	if len(*transportZoneName) == 0 {
		panic("--nsxt-transport-zone flag is not specified")
	}
	if len(*tier0GatewayName) == 0 {
		panic("--nsxt-t0-gateway flag is not specified")
	}
	if len(*snatIPPoolName) == 0 {
		panic("--nsxt-snat-ip-pool flag is not specified")
	}
}

func getNSXTConfig() (*infrastructure.NSXTConfig, error) {
	cfg := &infrastructure.NSXTConfig{}
	cfg.Host = *nsxtHost
	cfg.User = *nsxtUsername
	cfg.Password = *nsxtPassword
	cfg.InsecureFlag = true
	return cfg, nil
}

func getNSXTInfraSpec() (*infrastructure.NSXTInfraSpec, error) {
	spec := &infrastructure.NSXTInfraSpec{
		EdgeClusterName:          *edgeClusterName,
		TransportZoneName:        *transportZoneName,
		Tier0GatewayName:         *tier0GatewayName,
		SNATIPPoolName:           *snatIPPoolName,
		GardenID:                 "1234-5678-integration-test",
		GardenName:               "gardener-test",
		ClusterName:              "dummy",
		WorkersNetwork:           "10.251.0.0/19",
		DNSServers:               []string{"8.8.8.8"},
		ExternalTier1GatewayPath: nil,
	}
	return spec, nil
}

var _ = Describe("Infrastructure tests", func() {

	var (
		ctx    = context.Background()
		logger *logrus.Entry

		testEnv   *envtest.Environment
		mgrCancel context.CancelFunc
		c         client.Client

		decoder       runtime.Decoder
		vsphereClient task.EnsurerContext
		nsxtConfig    *infrastructure.NSXTConfig
		nsxtInfraSpec *infrastructure.NSXTInfraSpec

		internalChartsPath string
	)

	BeforeSuite(func() {
		flag.Parse()
		validateFlags()

		internalChartsPath = vsphere.InternalChartsPath
		repoRoot := filepath.Join("..", "..", "..")
		vsphere.InternalChartsPath = filepath.Join(repoRoot, vsphere.InternalChartsPath)

		// enable manager logs
		logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

		log := logrus.New()
		log.SetOutput(GinkgoWriter)
		logger = logrus.NewEntry(log)

		logger.Infof("NSX-T host: %s", *nsxtHost)
		logger.Infof("NSX-T username: %s", *nsxtUsername)
		logger.Infof("NSX-T T0-Gateway: %s", *tier0GatewayName)
		logger.Infof("NSX-T SNAT IP pool name: %s", *snatIPPoolName)
		By("starting test environment")
		testEnv = &envtest.Environment{
			UseExistingCluster: pointer.BoolPtr(true),
			CRDInstallOptions: envtest.CRDInstallOptions{
				Paths: []string{
					filepath.Join(repoRoot, "example", "20-crd-cluster.yaml"),
					filepath.Join(repoRoot, "example", "20-crd-infrastructure.yaml"),
				},
			},
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		By("parse flags")
		flag.Parse()
		validateFlags()

		nsxtConfig, err = getNSXTConfig()
		Expect(err).NotTo(HaveOccurred())
		nsxtInfraSpec, err = getNSXTInfraSpec()
		Expect(err).NotTo(HaveOccurred())

		By("setup manager")
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())

		Expect(extensionsv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
		Expect(vsphereinstall.AddToScheme(mgr.GetScheme())).To(Succeed())

		opts := controllerinfra.AddOptions{
			GardenId: nsxtInfraSpec.GardenID,
		}
		Expect(controllerinfra.AddToManagerWithOptions(mgr, opts)).To(Succeed())

		var mgrContext context.Context
		mgrContext, mgrCancel = context.WithCancel(ctx)

		By("start manager")
		go func() {
			err := mgr.Start(mgrContext)
			Expect(err).NotTo(HaveOccurred())
		}()

		c = mgr.GetClient()
		Expect(c).NotTo(BeNil())

		decoder = serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()
	})

	BeforeEach(func() {
		// new namespace for each test
		namespace, err := generateNamespaceName()
		Expect(err).NotTo(HaveOccurred())
		nsxtInfraSpec.ClusterName = namespace

		shootCtx := &ensurer.ShootContext{ShootNamespace: nsxtInfraSpec.ClusterName, GardenID: nsxtInfraSpec.GardenID}
		infraEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(logf.Log, nsxtConfig, shootCtx)
		Expect(err).NotTo(HaveOccurred())
		vsphereClient = infraEnsurer.(task.EnsurerContext)
		Expect(vsphereClient).NotTo(BeNil())
	})

	AfterSuite(func() {
		defer func() {
			By("stopping manager")
			mgrCancel()
		}()

		By("running cleanup actions")
		framework.RunCleanupActions()

		By("stopping test environment")
		Expect(testEnv.Stop()).To(Succeed())

		vsphere.InternalChartsPath = internalChartsPath
	})

	Context("with infrastructure creating own T1 gateway", func() {
		AfterEach(func() {
			framework.RunCleanupActions()
		})

		It("should successfully create and delete", func() {
			namespace := nsxtInfraSpec.ClusterName
			providerConfig := newProviderConfig("", "")
			cloudProfileConfig := newCloudProfileConfig(nsxtConfig, nsxtInfraSpec)

			err := runTest(ctx, logger, c, namespace, providerConfig, decoder, nsxtConfig, vsphereClient, cloudProfileConfig, nsxtInfraSpec.WorkersNetwork)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with infrastructure that uses existing T1 gateway", func() {
		AfterEach(func() {
			framework.RunCleanupActions()
		})

		It("should successfully create and delete", func() {
			namespace := nsxtInfraSpec.ClusterName

			t1Ref, lbSvcRef, err := prepareNewT1GatewayAndLBService(logger, namespace, *nsxtInfraSpec, vsphereClient)
			// ensure deleting resources even on errors
			var cleanupHandle framework.CleanupActionHandle
			cleanupHandle = framework.AddCleanupAction(func() {
				err := teardownT1GatewayAndLBService(logger, t1Ref, lbSvcRef, vsphereClient)
				Expect(err).NotTo(HaveOccurred())

				framework.RemoveCleanupAction(cleanupHandle)
			})
			Expect(err).NotTo(HaveOccurred())

			providerConfig := newProviderConfig(t1Ref.Path, lbSvcRef.Path)
			cloudProfileConfig := newCloudProfileConfig(nsxtConfig, nsxtInfraSpec)

			err = runTest(ctx, logger, c, namespace, providerConfig, decoder, nsxtConfig, vsphereClient, cloudProfileConfig, nsxtInfraSpec.WorkersNetwork)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func runTest(
	ctx context.Context,
	logger *logrus.Entry,
	c client.Client,
	namespaceName string,
	providerConfig *vspherev1alpha1.InfrastructureConfig,
	decoder runtime.Decoder,
	nsxtConfig *infrastructure.NSXTConfig,
	vsphereClient task.EnsurerContext,
	cloudProfileConfig *vspherev1alpha1.CloudProfileConfig,
	workerCIDR string,
) error {
	var (
		namespace      *corev1.Namespace
		cluster        *extensionsv1alpha1.Cluster
		infra          *extensionsv1alpha1.Infrastructure
		providerStatus *vspherev1alpha1.InfrastructureStatus
	)

	var cleanupHandle framework.CleanupActionHandle
	cleanupHandle = framework.AddCleanupAction(func() {
		By("delete infrastructure")
		Expect(client.IgnoreNotFound(c.Delete(ctx, infra))).To(Succeed())

		By("wait until infrastructure is deleted")
		err := extensions.WaitUntilExtensionCRDeleted(
			ctx,
			c,
			logger,
			func() extensionsv1alpha1.Object { return &extensionsv1alpha1.Infrastructure{} },
			"Infrastructure",
			infra.Namespace,
			infra.Name,
			10*time.Second,
			16*time.Minute,
		)
		Expect(err).NotTo(HaveOccurred())

		By("verify infrastructure deletion")
		verifyDeletion(ctx, vsphereClient.Connector(), providerStatus)

		Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
		Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())

		framework.RemoveCleanupAction(cleanupHandle)
	})

	By("create namespace for test execution")
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	if err := c.Create(ctx, namespace); err != nil {
		return err
	}

	cloudProfileConfigJSON, err := json.Marshal(&cloudProfileConfig)
	if err != nil {
		return err
	}

	cloudprofile := gardenerv1beta1.CloudProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gardenerv1beta1.SchemeGroupVersion.String(),
			Kind:       "CloudProfile",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: gardenerv1beta1.CloudProfileSpec{
			ProviderConfig: &runtime.RawExtension{
				Raw: cloudProfileConfigJSON,
			},
			MachineImages: []gardenerv1beta1.MachineImage{
				{
					Name: "gardenlinux",
					Versions: []gardenerv1beta1.MachineImageVersion{
						{ExpirableVersion: gardenerv1beta1.ExpirableVersion{Version: "27.1.0"}},
					},
				},
			},
		},
	}

	cloudProfileJSON, err := json.Marshal(&cloudprofile)
	if err != nil {
		return err
	}

	providerConfigJSON, err := json.Marshal(providerConfig)
	if err != nil {
		return err
	}

	nameParts := strings.Split(namespaceName, "--")
	shoot := gardenerv1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gardenerv1beta1.SchemeGroupVersion.String(),
			Kind:       "Shoot",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nameParts[len(nameParts)-1],
		},
		Spec: gardenerv1beta1.ShootSpec{
			Networking: gardenerv1beta1.Networking{
				Nodes: &workerCIDR,
			},
			Provider: gardenerv1beta1.Provider{
				InfrastructureConfig: &runtime.RawExtension{
					Raw: providerConfigJSON,
				},
			},
		},
	}

	shootJSON, err := json.Marshal(&shoot)
	if err != nil {
		return err
	}

	By("create cluster")
	cluster = &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{
				Raw: cloudProfileJSON,
			},
			Shoot: runtime.RawExtension{
				Raw: shootJSON,
			},
		},
	}
	if err := c.Create(ctx, cluster); err != nil {
		return err
	}

	By("deploy cloudprovider secret into namespace")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudprovider",
			Namespace: namespaceName,
		},
		Data: map[string][]byte{
			vsphere.NSXTUsername: []byte(nsxtConfig.User),
			vsphere.NSXTPassword: []byte(nsxtConfig.Password),
			vsphere.Username:     []byte(""),
			vsphere.Password:     []byte(""),
		},
	}
	if err := c.Create(ctx, secret); err != nil {
		return err
	}

	By("create infrastructure")
	infra, err = newInfrastructure(namespaceName, cloudProfileConfig.Regions[0].Name, providerConfig)
	if err != nil {
		return err
	}

	if err := c.Create(ctx, infra); err != nil {
		return err
	}

	By("wait until infrastructure is created")
	if err := extensions.WaitUntilExtensionCRReady(
		ctx,
		c,
		logger,
		func() client.Object { return &extensionsv1alpha1.Infrastructure{} },
		"Infrastucture",
		infra.Namespace,
		infra.Name,
		10*time.Second,
		30*time.Second,
		16*time.Minute,
		nil,
	); err != nil {
		return err
	}

	By("decode infrastucture status")
	if err := c.Get(ctx, client.ObjectKey{Namespace: infra.Namespace, Name: infra.Name}, infra); err != nil {
		return err
	}

	providerStatus = &vspherev1alpha1.InfrastructureStatus{}
	if _, _, err := decoder.Decode(infra.Status.ProviderStatus.Raw, nil, providerStatus); err != nil {
		return err
	}

	By("verify infrastructure creation")
	verifyCreation(providerStatus)

	return nil
}

func newProviderConfig(t1gwPath, lbSvcPath string) *vspherev1alpha1.InfrastructureConfig {
	config := &vspherev1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: vspherev1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureConfig",
		},
	}
	if t1gwPath != "" && lbSvcPath != "" {
		config.Networks = &vspherev1alpha1.Networks{
			Tier1GatewayPath:        t1gwPath,
			LoadBalancerServicePath: lbSvcPath,
		}
	}
	return config
}

func newCloudProfileConfig(cfg *infrastructure.NSXTConfig, spec *infrastructure.NSXTInfraSpec) *vspherev1alpha1.CloudProfileConfig {
	return &vspherev1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: vspherev1alpha1.SchemeGroupVersion.String(),
			Kind:       "CloudProfileConfig",
		},
		NamePrefix: spec.GardenName,
		Folder:     "gardener",
		Regions: []vspherev1alpha1.RegionSpec{
			{
				Name:               "infrastructure-test-region",
				NSXTHost:           cfg.Host,
				NSXTInsecureSSL:    cfg.InsecureFlag,
				NSXTRemoteAuth:     false,
				VsphereHost:        "vsphere.dummy",
				TransportZone:      spec.TransportZoneName,
				LogicalTier0Router: spec.Tier0GatewayName,
				EdgeCluster:        spec.EdgeClusterName,
				SNATIPPool:         spec.SNATIPPoolName,
				Datacenter:         str("DummyDC"),
				Zones: []vspherev1alpha1.ZoneSpec{
					{
						Name:         "infrastructure-test-region-a",
						Datastore:    str("DummyDatastore"),
						ResourcePool: str("DummyResourcePool"),
					},
				},
			},
		},
		DefaultClassStoragePolicyName: "test",
		FailureDomainLabels:           nil,
		DNSServers:                    spec.DNSServers,
		MachineImages: []vspherev1alpha1.MachineImages{
			{
				Name: "gardenlinux",
				Versions: []vspherev1alpha1.MachineImageVersion{
					{
						Version: "27.1.0",
						Path:    "/gardener/template/gardenlinux-27.1.0",
					},
				},
			},
		},
		Constraints: vspherev1alpha1.Constraints{
			LoadBalancerConfig: vspherev1alpha1.LoadBalancerConfig{
				Size: "SMALL",
				Classes: []vspherev1alpha1.LoadBalancerClass{
					{
						Name:       "default",
						IPPoolName: &spec.SNATIPPoolName, // SNAT IP pool here only used for LB cleanup testing
					},
				},
			},
		},
		CSIResizerDisabled: nil,
		MachineTypeOptions: nil,
	}
}

func newInfrastructure(namespace, region string, providerConfig *vspherev1alpha1.InfrastructureConfig) (*extensionsv1alpha1.Infrastructure, error) {
	const sshPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDcSZKq0lM9w+ElLp9I9jFvqEFbOV1+iOBX7WEe66GvPLOWl9ul03ecjhOf06+FhPsWFac1yaxo2xj+SJ+FVZ3DdSn4fjTpS9NGyQVPInSZveetRw0TV0rbYCFBTJuVqUFu6yPEgdcWq8dlUjLqnRNwlelHRcJeBfACBZDLNSxjj0oUz7ANRNCEne1ecySwuJUAz3IlNLPXFexRT0alV7Nl9hmJke3dD73nbeGbQtwvtu8GNFEoO4Eu3xOCKsLw6ILLo4FBiFcYQOZqvYZgCb4ncKM52bnABagG54upgBMZBRzOJvWp0ol+jK3Em7Vb6ufDTTVNiQY78U6BAlNZ8Xg+LUVeyk1C6vWjzAQf02eRvMdfnRCFvmwUpzbHWaVMsQm8gf3AgnTUuDR0ev1nQH/5892wZA86uLYW/wLiiSbvQsqtY1jSn9BAGFGdhXgWLAkGsd/E1vOT+vDcor6/6KjHBm0rG697A3TDBRkbXQ/1oFxcM9m17RteCaXuTiAYWMqGKDoJvTMDc4L+Uvy544pEfbOH39zfkIYE76WLAFPFsUWX6lXFjQrX3O7vEV73bCHoJnwzaNd03PSdJOw+LCzrTmxVezwli3F9wUDiBRB0HkQxIXQmncc1HSecCKALkogIK+1e1OumoWh6gPdkF4PlTMUxRitrwPWSaiUIlPfCpQ== your_email@example.com"

	providerConfigJSON, err := json.Marshal(&providerConfig)
	if err != nil {
		return nil, err
	}

	return &extensionsv1alpha1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "infrastructure",
			Namespace: namespace,
		},
		Spec: extensionsv1alpha1.InfrastructureSpec{
			DefaultSpec: extensionsv1alpha1.DefaultSpec{
				Type: vsphere.Type,
				ProviderConfig: &runtime.RawExtension{
					Raw: providerConfigJSON,
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      "cloudprovider",
				Namespace: namespace,
			},
			Region:       region,
			SSHPublicKey: []byte(sshPublicKey),
		},
	}, nil
}

func generateNamespaceName() (string, error) {
	suffix, err := gardenerutils.GenerateRandomStringFromCharset(5, "0123456789abcdefghijklmnopqrstuvwxyz")
	if err != nil {
		return "", err
	}

	return "vsphere--infra-it--" + suffix, nil
}

func prepareNewT1GatewayAndLBService(logger *logrus.Entry, technicalShootName string, spec infrastructure.NSXTInfraSpec,
	ensurerCtx task.EnsurerContext) (t1Ref *apisvsphere.Reference, lbRef *apisvsphere.Reference, err error) {
	logger.Infof("Creating Tier1 gateway and LB service...")

	state := apisvsphere.NSXTInfraState{}

	taskT0 := task.NewLookupTier0GatewayTask()
	action, err := taskT0.Ensure(ensurerCtx, spec, &state)
	if err != nil {
		return
	}
	logger.Infof("T0 Gateway lookup: %s", action)

	taskEC := task.NewLookupEdgeClusterTask()
	action, err = taskEC.Ensure(ensurerCtx, spec, &state)
	if err != nil {
		return
	}
	logger.Infof("Edge Cluster lookup: %s", action)

	taskT1 := task.NewTier1GatewayTask()
	action, err = taskT1.Ensure(ensurerCtx, spec, &state)
	if err != nil {
		return
	}
	t1Ref = state.Tier1GatewayRef
	logger.Infof("T1 Gateway '%s': %s", state.Tier1GatewayRef.Path, action)

	// update tags for permissions
	client := infra.NewDefaultTier1sClient(ensurerCtx.Connector())
	tier1, err := client.Get(t1Ref.ID)
	if err != nil {
		return
	}
	authTag := model.Tag{
		Scope: str(infrastructure.ScopeAuthorizedShoots),
		Tag:   str(technicalShootName),
	}
	newTags := []model.Tag{authTag}
	for _, tag := range tier1.Tags {
		if *tag.Scope == infrastructure.ScopeGarden {
			newTags = append(newTags, tag)
		}
	}
	tier1.Tags = newTags
	_, err = client.Update(t1Ref.ID, tier1)
	if err != nil {
		return
	}
	logger.Infof("T1 Gateway '%s': tags updated", state.Tier1GatewayRef.Path)

	taskT1Locale := task.NewTier1GatewayLocaleServiceTask()
	action, err = taskT1Locale.Ensure(ensurerCtx, spec, &state)
	if err != nil {
		return
	}
	logger.Infof("T1 Gateway Locale Service: %s", action)

	lbClient := infra.NewDefaultLbServicesClient(ensurerCtx.Connector())
	lbID := "infrastructure-test-" + uuid.New().String()
	lbService := model.LBService{
		Description:      str("infrastructure-test for " + technicalShootName),
		DisplayName:      str("infrastructure-test-" + technicalShootName),
		Tags:             []model.Tag{authTag},
		Size:             str("SMALL"),
		Enabled:          boolptr(true),
		ConnectivityPath: str(state.Tier1GatewayRef.Path),
	}
	service, err := lbClient.Update(lbID, lbService)
	if err != nil {
		return
	}
	logger.Infof("LB service '%s': created", *service.Path)
	lbRef = &apisvsphere.Reference{
		ID:   *service.Id,
		Path: *service.Path,
	}

	return
}

func teardownT1GatewayAndLBService(logger *logrus.Entry, t1Ref, lbRef *apisvsphere.Reference, ensurerCtx task.EnsurerContext) error {
	logger.Infof("Deleting Tier1 gateway and LB service...")

	errmsg := ""
	if lbRef != nil {
		client := infra.NewDefaultLbServicesClient(ensurerCtx.Connector())
		err := client.Delete(lbRef.ID, boolptr(true))
		if err != nil {
			errmsg += fmt.Sprintf("deleting LB service failed with %s, ", err)
		}
		logger.Infof("LB service '%s': deleted", lbRef.Path)
	}

	if t1Ref != nil {
		state := apisvsphere.NSXTInfraState{
			Tier1GatewayRef:  t1Ref,
			LocaleServiceRef: &apisvsphere.Reference{ID: t1Ref.ID, Path: ""},
		}

		taskT1Locale := task.NewTier1GatewayLocaleServiceTask()
		deleted, err := taskT1Locale.EnsureDeleted(ensurerCtx, &state)
		if err != nil {
			errmsg += fmt.Sprintf("deleting T1 gateway locale service failed with %s, ", err)
		}
		if deleted {
			logger.Infof("T1 Gateway Locale Service deleted")
		}

		taskT1 := task.NewTier1GatewayTask()
		deleted, err = taskT1.EnsureDeleted(ensurerCtx, &state)
		if err != nil {
			errmsg += fmt.Sprintf("deleting T1 gateway failed with %s, ", err)
		}
		if deleted {
			logger.Infof("T1 Gateway '%s': deleted", t1Ref.Path)
		}
	}

	if errmsg != "" {
		return fmt.Errorf(errmsg)
	}
	return nil
}

func verifyCreation(infraStatus *vspherev1alpha1.InfrastructureStatus) {
	Expect(infraStatus.CreationStarted).NotTo(BeNil())
	Expect(*infraStatus.CreationStarted).To(Equal(true))

	state := infraStatus.NSXTInfraState
	Expect(state).NotTo(BeNil())

	if state.ExternalTier1Gateway == nil || !*state.ExternalTier1Gateway {
		// tier1 gateway exists
		Expect(state.Tier1GatewayRef).NotTo(BeNil())
	}

	// network segment exists
	Expect(state.SegmentName).NotTo(BeNil())
	Expect(state.SegmentRef).NotTo(BeNil())

	// SNAT IP address has been allocated
	Expect(state.SNATIPAddress).NotTo(BeNil())

	// SNAT rule exists
	Expect(state.SNATRuleRef).NotTo(BeNil())

	// DHCPServerConfig rule exists
	Expect(state.DHCPServerConfigRef).NotTo(BeNil())
}

func verifyDeletion(
	ctx context.Context,
	nsxtClientConnector vapiclient.Connector,
	oldInfraStatus *vspherev1alpha1.InfrastructureStatus,
) {
	if oldInfraStatus.NSXTInfraState == nil {
		return
	}
	state := oldInfraStatus.NSXTInfraState

	isNotFoundError := func(err error) bool {
		if _, ok := err.(vapi_errors.NotFound); ok {
			return true
		}

		return false
	}

	// SNAT IP address doesn't exist
	if state.SNATIPAddressAllocRef != nil {
		client := ip_pools.NewDefaultIpAllocationsClient(nsxtClientConnector)
		_, err := client.Get(state.SNATIPPoolRef.ID, state.SNATIPAddressAllocRef.ID)
		Expect(err).To(HaveOccurred())
		Expect(isNotFoundError(err)).To(BeTrue())
	}

	// SNAT rule doesn't exist
	if state.SNATRuleRef != nil {
		client := t1nat.NewDefaultNatRulesClient(nsxtClientConnector)
		_, err := client.Get(state.Tier1GatewayRef.ID, model.PolicyNat_NAT_TYPE_USER, state.SNATRuleRef.ID)
		Expect(err).To(HaveOccurred())
		Expect(isNotFoundError(err)).To(BeTrue())
	}

	// DHCPServerConfig doesn't exist
	if state.DHCPServerConfigRef != nil {
		client := infra.NewDefaultDhcpServerConfigsClient(nsxtClientConnector)
		_, err := client.Get(state.DHCPServerConfigRef.ID)
		Expect(err).To(HaveOccurred())
		Expect(isNotFoundError(err)).To(BeTrue())
	}

	// network segment doesn't exist
	if state.SegmentRef != nil {
		client := infra.NewDefaultSegmentsClient(nsxtClientConnector)
		_, err := client.Get(state.SegmentRef.ID)
		Expect(err).To(HaveOccurred())
		Expect(isNotFoundError(err)).To(BeTrue())
	}

	// network segment doesn't exist
	if (state.ExternalTier1Gateway == nil || !*state.ExternalTier1Gateway) && state.Tier1GatewayRef != nil {
		client := infra.NewDefaultTier1sClient(nsxtClientConnector)
		_, err := client.Get(state.Tier1GatewayRef.ID)
		Expect(err).To(HaveOccurred())
		Expect(isNotFoundError(err)).To(BeTrue())
	}
}

func str(s string) *string {
	return &s
}

func boolptr(b bool) *bool {
	return &b
}
