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
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gutils "github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/chart"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"

	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	apishelper "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/utils"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/helpers"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/ensurer"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"
)

var controlPlaneSecrets = &secrets.Secrets{
	CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
		v1beta1constants.SecretNameCACluster: {
			Name:       v1beta1constants.SecretNameCACluster,
			CommonName: "kubernetes",
			CertType:   secrets.CACert,
		},
	},
	SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
		return []secrets.ConfigInterface{
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.CloudControllerManagerName,
					CommonName:   "system:serviceaccount:kube-system:cloud-controller-manager",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       vsphere.CloudControllerManagerServerName,
					CommonName: vsphere.CloudControllerManagerName,
					DNSNames:   kutil.DNSNamesForService(vsphere.CloudControllerManagerName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1beta1constants.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.CSIAttacherName,
					CommonName:   vsphere.UsernamePrefix + vsphere.CSIAttacherName,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.CSIProvisionerName,
					CommonName:   vsphere.UsernamePrefix + vsphere.CSIProvisionerName,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       vsphere.CSISnapshotterName,
					CommonName: vsphere.UsernamePrefix + vsphere.CSISnapshotterName,
					CertType:   secrets.ClientCert,
					SigningCA:  cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.VsphereCSIControllerName,
					CommonName:   vsphere.UsernamePrefix + vsphere.VsphereCSIControllerName,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.VsphereCSISyncerName,
					CommonName:   vsphere.UsernamePrefix + vsphere.VsphereCSISyncerName,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.CSIResizerName,
					CommonName:   vsphere.UsernamePrefix + vsphere.CSIResizerName,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       vsphere.CSISnapshotControllerName,
					CommonName: vsphere.UsernamePrefix + vsphere.CSISnapshotControllerName,
					CertType:   secrets.ClientCert,
					SigningCA:  cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1beta1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
		}
	},
}

var configChart = &chart.Chart{
	Name: "cloud-provider-config",
	Path: filepath.Join(vsphere.InternalChartsPath, "cloud-provider-config"),
	Objects: []*chart.Object{
		{Type: &corev1.Secret{}, Name: vsphere.CloudProviderConfig},
	},
}

var controlPlaneChart = &chart.Chart{
	Name: "seed-controlplane",
	Path: filepath.Join(vsphere.InternalChartsPath, "seed-controlplane"),
	SubCharts: []*chart.Chart{
		{
			Name:   "vsphere-cloud-controller-manager",
			Images: []string{vsphere.CloudControllerImageName},
			Objects: []*chart.Object{
				{Type: &corev1.Service{}, Name: vsphere.CloudControllerManagerName},
				{Type: &appsv1.Deployment{}, Name: vsphere.CloudControllerManagerName},
				{Type: &corev1.ConfigMap{}, Name: vsphere.CloudControllerManagerName + "-observability-config"},
				{Type: &autoscalingv1beta2.VerticalPodAutoscaler{}, Name: vsphere.CloudControllerManagerName + "-vpa"},
			},
		},
		{
			Name: "csi-vsphere",
			Images: []string{
				vsphere.CSIAttacherImageName,
				vsphere.CSIProvisionerImageName,
				vsphere.CSIDriverControllerImageName,
				vsphere.CSIDriverSyncerImageName,
				vsphere.CSIResizerImageName,
				vsphere.CSISnapshotterImageName,
				vsphere.LivenessProbeImageName,
				vsphere.CSISnapshotControllerImageName},
			Objects: []*chart.Object{
				// csi-driver-controller
				{Type: &corev1.Secret{}, Name: vsphere.SecretCSIVsphereConfig},
				{Type: &corev1.Service{}, Name: vsphere.VsphereCSIControllerName},
				{Type: &appsv1.Deployment{}, Name: vsphere.VsphereCSIControllerName},
				{Type: &corev1.ConfigMap{}, Name: vsphere.VsphereCSIControllerName + "-observability-config"},
				{Type: &autoscalingv1beta2.VerticalPodAutoscaler{}, Name: vsphere.VsphereCSIControllerName + "-vpa"},
				// csi-snapshot-controller
				{Type: &appsv1.Deployment{}, Name: vsphere.CSISnapshotControllerName},
				{Type: &autoscalingv1beta2.VerticalPodAutoscaler{}, Name: vsphere.CSISnapshotControllerName + "-vpa"},
			},
		},
	},
}

var controlPlaneShootChart = &chart.Chart{
	Name: "shoot-system-components",
	Path: filepath.Join(vsphere.InternalChartsPath, "shoot-system-components"),
	SubCharts: []*chart.Chart{
		{
			Name: "vsphere-cloud-controller-manager",
			Objects: []*chart.Object{
				{Type: &corev1.ServiceAccount{}, Name: "cloud-controller-manager"},
				{Type: &rbacv1.ClusterRole{}, Name: "system:cloud-controller-manager"},
				{Type: &rbacv1.RoleBinding{}, Name: "system:cloud-controller-manager:apiserver-authentication-reader"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:cloud-controller-manager"},
			},
		},
		{
			Name: "csi-vsphere",
			Images: []string{
				vsphere.CSINodeDriverRegistrarImageName,
				vsphere.CSIDriverNodeImageName,
				vsphere.LivenessProbeImageName,
			},
			Objects: []*chart.Object{
				// csi-driver
				{Type: &appsv1.DaemonSet{}, Name: vsphere.CSINodeName},
				//{Type: &storagev1beta1.CSIDriver{}, Name: "csi.vsphere.vmware.com"},
				{Type: &corev1.ServiceAccount{}, Name: vsphere.CSIDriverName + "-node"},
				{Type: &corev1.Secret{}, Name: vsphere.SecretCSIVsphereConfig},
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSIDriverName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIDriverName},
				{Type: &policyv1beta1.PodSecurityPolicy{}, Name: strings.Replace(vsphere.UsernamePrefix+vsphere.CSIDriverName, ":", ".", -1)},
				// csi-provisioner
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSIProvisionerName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIProvisionerName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.CSIProvisionerName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIProvisionerName},
				// csi-attacher
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSIAttacherName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIAttacherName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.CSIAttacherName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIAttacherName},
				// csi-resizer
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSIResizerName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIResizerName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.CSIResizerName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSIResizerName},
				// csi-snapshotter
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotterName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotterName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotterName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotterName},
				// csi-snapshot-controller
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotControllerName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotControllerName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotControllerName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.CSISnapshotControllerName},
				// csi-syncer
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncerName},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncerName},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncerName},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncerName},
			},
		},
	},
}

var controlPlaneShootCRDsChart = &chart.Chart{
	Name: "shoot-crds",
	Path: filepath.Join(vsphere.InternalChartsPath, "shoot-crds"),
	SubCharts: []*chart.Chart{
		{
			Name: "volumesnapshots",
			Objects: []*chart.Object{
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshotclasses.snapshot.storage.k8s.io"},
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshotcontents.snapshot.storage.k8s.io"},
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshots.snapshot.storage.k8s.io"},
			},
		},
	},
}

var storageClassChart = &chart.Chart{
	Name: "shoot-storageclasses",
	Path: filepath.Join(vsphere.InternalChartsPath, "shoot-storageclasses"),
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(logger logr.Logger, gardenID string) genericactuator.ValuesProvider {
	return &valuesProvider{
		logger:   logger.WithName("vsphere-values-provider"),
		gardenID: gardenID,
	}
}

// valuesProvider is a ValuesProvider that provides vSphere-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	genericactuator.NoopValuesProvider
	common.ClientContext
	logger   logr.Logger
	gardenID string
}

// GetConfigChartValues returns the values for the config chart applied by the generic actuator.
func (vp *valuesProvider) GetConfigChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	cpConfig, err := helper.GetControlPlaneConfig(cluster)
	if err != nil {
		return nil, err
	}

	// Get credentials
	credentials, err := vsphere.GetCredentials(ctx, vp.Client(), cp.Spec.SecretRef)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get vSphere credentials from secret '%s/%s'", cp.Spec.SecretRef.Namespace, cp.Spec.SecretRef.Name)
	}

	// Get config chart values
	return vp.getConfigChartValues(cp, cpConfig, cluster, credentials)
}

// GetControlPlaneChartValues returns the values for the control plane chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	cpConfig, err := helper.GetControlPlaneConfig(cluster)
	if err != nil {
		return nil, err
	}

	// Get credentials
	credentials, err := vsphere.GetCredentials(ctx, vp.Client(), cp.Spec.SecretRef)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get vSphere credentials from secret '%s/%s'", cp.Spec.SecretRef.Namespace, cp.Spec.SecretRef.Name)
	}

	secretCloudProviderConfig := &corev1.Secret{}
	if err := vp.Client().Get(ctx, kutil.Key(cp.Namespace, vsphere.CloudProviderConfig), secretCloudProviderConfig); err == nil {
		checksums[vsphere.CloudProviderConfig] = gutils.ComputeChecksum(secretCloudProviderConfig.Data)
	}

	secretCSIVsphereConfig := &corev1.Secret{}
	if err := vp.Client().Get(ctx, kutil.Key(cp.Namespace, vsphere.SecretCSIVsphereConfig), secretCSIVsphereConfig); err == nil {
		checksums[vsphere.SecretCSIVsphereConfig] = gutils.ComputeChecksum(secretCSIVsphereConfig.Data)
	}

	// TODO: Remove this code in next version. Delete old config
	if err := vp.deleteCCMMonitoringConfig(ctx, cp.Namespace); err != nil {
		return nil, err
	}

	// TODO: Remove this code in a future version again.
	if err := vp.deleteLegacyCloudProviderConfigMap(ctx, cp.Namespace); err != nil {
		return nil, err
	}

	// Get control plane chart values
	return vp.getControlPlaneChartValues(cpConfig, cp, cluster, credentials, checksums, scaledDown)
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
) (map[string]interface{}, error) {
	// Get credentials
	credentials, err := vsphere.GetCredentials(ctx, vp.Client(), cp.Spec.SecretRef)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get vSphere credentials from secret '%s/%s'", cp.Spec.SecretRef.Namespace, cp.Spec.SecretRef.Name)
	}

	// Get control plane shoot chart values
	return vp.getControlPlaneShootChartValues(cp, cluster, credentials)
}

// GetControlPlaneShootCRDsChartValues returns the values for the control plane shoot CRDs chart applied by the generic actuator.
// Currently the provider extension does not specify a control plane shoot CRDs chart. That's why we simply return empty values.
func (vp *valuesProvider) GetControlPlaneShootCRDsChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	return map[string]interface{}{
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"volumesnapshots": map[string]interface{}{
			"enabled": false, // not supported in vsphere-csi-driver v2.3.0
		},
	}, nil
}

// GetStorageClassesChartValues returns the values for the shoot storageclasses chart applied by the generic actuator.
func (vp *valuesProvider) GetStorageClassesChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {

	cloudProfileConfig, err := helper.GetCloudProfileConfig(cluster)
	if err != nil {
		return nil, err
	}

	volumeBindingMode := "Immediate"
	if cloudProfileConfig.FailureDomainLabels != nil {
		// can only be used if topology tags are set
		volumeBindingMode = "WaitForFirstConsumer"
	}
	allowVolumeExpansion := cloudProfileConfig.CSIResizerDisabled == nil || !*cloudProfileConfig.CSIResizerDisabled

	return map[string]interface{}{
		"storagePolicyName":    cloudProfileConfig.DefaultClassStoragePolicyName,
		"volumeBindingMode":    volumeBindingMode,
		"allowVolumeExpansion": allowVolumeExpansion,
	}, nil
}

func splitServerNameAndPort(host string) (name string, port int, err error) {
	parts := strings.Split(host, ":")
	if len(parts) == 1 {
		name = host
		port = 443
	} else if len(parts) == 2 {
		name = parts[0]
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, errors.Wrapf(err, "invalid port for vSphere host: host=%s,port=%s", host, parts[1])
		}
	} else {
		return "", 0, fmt.Errorf("invalid vSphere host: %s (too many parts %v)", host, parts)
	}

	return
}

// getConfigChartValues collects and returns the configuration chart values.
func (vp *valuesProvider) getConfigChartValues(
	cp *extensionsv1alpha1.ControlPlane,
	cpConfig *apisvsphere.ControlPlaneConfig,
	cluster *extensionscontroller.Cluster,
	credentials *vsphere.Credentials,
) (map[string]interface{}, error) {

	cloudProfileConfig, err := helper.GetCloudProfileConfig(cluster)
	if err != nil {
		return nil, err
	}

	infraConfig, err := helper.GetInfrastructureConfig(cluster)
	if err != nil {
		return nil, err
	}

	region := helper.FindRegion(cluster.Shoot.Spec.Region, cloudProfileConfig)
	if region == nil {
		return nil, fmt.Errorf("region %q not found in cloud profile config", cluster.Shoot.Spec.Region)
	}

	serverName, port, err := splitServerNameAndPort(region.VsphereHost)
	if err != nil {
		return nil, err
	}

	infraStatus, err := helper.GetInfrastructureStatus(cp.Name, cp.Spec.InfrastructureProviderStatus)
	if err != nil {
		return nil, err
	}

	checkFunc := vp.checkAuthorizationOfOverwrittenIPPoolName(cluster, cloudProfileConfig, credentials)
	defaultClass, loadBalancersClasses, err := validation.OverwriteLoadBalancerClasses(
		cloudProfileConfig.Constraints.LoadBalancerConfig.Classes, cpConfig, checkFunc)
	if err != nil {
		return nil, err
	}
	loadBalancersClassesMap := []map[string]interface{}{}
	for _, cpClass := range loadBalancersClasses {
		lbClass := map[string]interface{}{
			"name": cpClass.Name,
		}
		if !utils.IsEmptyString(cpClass.IPPoolName) {
			lbClass["ipPoolName"] = *cpClass.IPPoolName
		}
		if !utils.IsEmptyString(cpClass.TCPAppProfileName) {
			lbClass["tcpAppProfileName"] = *cpClass.TCPAppProfileName
		}
		if !utils.IsEmptyString(cpClass.UDPAppProfileName) {
			lbClass["udpAppProfileName"] = *cpClass.UDPAppProfileName
		}
		loadBalancersClassesMap = append(loadBalancersClassesMap, lbClass)
	}

	lbSize := cloudProfileConfig.Constraints.LoadBalancerConfig.Size
	if cpConfig.LoadBalancerSize != nil && *cpConfig.LoadBalancerSize != "" {
		lbSize = *cpConfig.LoadBalancerSize
	}
	loadBalancer := map[string]interface{}{
		"ipPoolName": *defaultClass.IPPoolName,
		"size":       lbSize,
		"classes":    loadBalancersClassesMap,
		"tags":       map[string]interface{}{"owner": vp.gardenID},
	}
	if !utils.IsEmptyString(defaultClass.TCPAppProfileName) {
		loadBalancer["tcpAppProfileName"] = *defaultClass.TCPAppProfileName
	}
	if !utils.IsEmptyString(defaultClass.UDPAppProfileName) {
		loadBalancer["udpAppProfileName"] = *defaultClass.UDPAppProfileName
	}
	if infraStatus.NSXTInfraState != nil && infraStatus.NSXTInfraState.Tier1GatewayRef != nil {
		loadBalancer["tier1GatewayPath"] = infraStatus.NSXTInfraState.Tier1GatewayRef.Path
	}
	if infraConfig.Networks != nil {
		loadBalancer["lbServiceId"] = task.IdFromPath(infraConfig.Networks.LoadBalancerServicePath)
	}

	// Collect config chart values
	values := map[string]interface{}{
		"serverName":   serverName,
		"serverPort":   port,
		"insecureFlag": region.VsphereInsecureSSL,
		"datacenters":  helper.CollectDatacenters(region),
		"username":     credentials.VsphereCCM().Username,
		"password":     credentials.VsphereCCM().Password,
		"loadbalancer": loadBalancer,
		"nsxt": map[string]interface{}{
			"host":         region.NSXTHost,
			"insecureFlag": region.NSXTInsecureSSL,
			"username":     credentials.NSXT().Username,
			"password":     credentials.NSXT().Password,
			"remoteAuth":   region.NSXTRemoteAuth,
		},
	}

	if !utils.IsEmptyString(region.CaFile) {
		values["caFile"] = *region.CaFile
	}
	if !utils.IsEmptyString(region.Thumbprint) {
		values["thumbprint"] = *region.Thumbprint
	}
	if cloudProfileConfig.FailureDomainLabels != nil {
		values["labelRegion"] = cloudProfileConfig.FailureDomainLabels.Region
		values["labelZone"] = cloudProfileConfig.FailureDomainLabels.Zone
	}

	return values, nil
}

func (vp *valuesProvider) checkAuthorizationOfOverwrittenIPPoolName(cluster *extensionscontroller.Cluster,
	cloudProfileConfig *apisvsphere.CloudProfileConfig, credentials *vsphere.Credentials) func(ipPoolName string) error {

	wrap := func(err error) error {
		return errors.Wrap(err, "checkAuthorizationOfOverwrittenIPPoolName failed")
	}
	return func(ipPoolName string) error {
		regionName := cluster.Shoot.Spec.Region
		region := apishelper.FindRegion(regionName, cloudProfileConfig)
		if region == nil {
			return wrap(fmt.Errorf("region %q not found in cloud profile", regionName))
		}
		nsxtConfig := helpers.NewNSXTConfig(credentials, region)
		shootCtx := &ensurer.ShootContext{ShootNamespace: cluster.ObjectMeta.Name, GardenID: vp.gardenID}
		infraEnsurer, err := ensurer.NewNSXTInfrastructureEnsurer(vp.logger, nsxtConfig, shootCtx)
		if err != nil {
			return wrap(err)
		}

		tags, err := infraEnsurer.GetIPPoolTags(ipPoolName)
		if err != nil {
			return wrap(err)
		}

		return infraEnsurer.CheckShootAuthorizationByTags("IP pool", ipPoolName, tags)
	}
}

func getTLSCipherSuites(kubeVersion *semver.Version) []string {
	// the following suites are not supported by the deployed cloud-controller-manager
	// TODO: This can be removed as soon as the cloud-controller-manager was updated to support the TLS suites.
	unsupportedSuites := sets.NewString(
		"TLS_AES_128_GCM_SHA256",
		"TLS_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	)

	var ciphers []string
	for _, cipher := range kutil.TLSCipherSuites(kubeVersion) {
		if unsupportedSuites.Has(cipher) {
			continue
		}
		ciphers = append(ciphers, cipher)
	}

	return ciphers
}

// getControlPlaneChartValues collects and returns the control plane chart values.
func (vp *valuesProvider) getControlPlaneChartValues(
	cpConfig *apisvsphere.ControlPlaneConfig,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	credentials *vsphere.Credentials,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {

	cloudProfileConfig, err := helper.GetCloudProfileConfig(cluster)
	if err != nil {
		return nil, err
	}

	region := helper.FindRegion(cluster.Shoot.Spec.Region, cloudProfileConfig)
	if region == nil {
		return nil, fmt.Errorf("region %q not found in cloud profile config", cluster.Shoot.Spec.Region)
	}

	serverName, port, err := splitServerNameAndPort(region.VsphereHost)
	if err != nil {
		return nil, err
	}

	kubeVersion, err := semver.NewVersion(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return nil, err
	}
	clusterID, csiClusterID := vp.calcClusterIDs(cp)
	csiResizerEnabled := cloudProfileConfig.CSIResizerDisabled == nil || !*cloudProfileConfig.CSIResizerDisabled
	values := map[string]interface{}{
		"vsphere-cloud-controller-manager": map[string]interface{}{
			"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"clusterName":       clusterID,
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"podNetwork":        extensionscontroller.GetPodNetwork(cluster),
			"podAnnotations": map[string]interface{}{
				"checksum/secret-" + vsphere.CloudControllerManagerName:       checksums[vsphere.CloudControllerManagerName],
				"checksum/secret-" + vsphere.CloudControllerManagerServerName: checksums[vsphere.CloudControllerManagerServerName],
				"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
				"checksum/secret-" + vsphere.CloudProviderConfig:              checksums[vsphere.CloudProviderConfig],
			},
			"podLabels": map[string]interface{}{
				v1beta1constants.LabelPodMaintenanceRestart: "true",
			},
			"tlsCipherSuites": getTLSCipherSuites(kubeVersion),
		},
		"csi-vsphere": map[string]interface{}{
			"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"serverName":        serverName,
			"clusterID":         csiClusterID,
			"username":          credentials.VsphereCSI().Username,
			"password":          credentials.VsphereCSI().Password,
			"serverPort":        port,
			"datacenters":       strings.Join(helper.CollectDatacenters(region), ","),
			"insecureFlag":      fmt.Sprintf("%t", region.VsphereInsecureSSL),
			"resizerEnabled":    csiResizerEnabled,
			"podAnnotations": map[string]interface{}{
				"checksum/secret-" + vsphere.CSIProvisionerName:               checksums[vsphere.CSIProvisionerName],
				"checksum/secret-" + vsphere.CSIAttacherName:                  checksums[vsphere.CSIAttacherName],
				"checksum/secret-" + vsphere.CSIResizerName:                   checksums[vsphere.CSIResizerName],
				"checksum/secret-" + vsphere.CSISnapshotterName:               checksums[vsphere.CSISnapshotterName],
				"checksum/secret-" + vsphere.VsphereCSIControllerName:         checksums[vsphere.VsphereCSIControllerName],
				"checksum/secret-" + vsphere.VsphereCSISyncerName:             checksums[vsphere.VsphereCSISyncerName],
				"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
				"checksum/secret-" + vsphere.SecretCSIVsphereConfig:           checksums[vsphere.SecretCSIVsphereConfig],
			},
			"csiSnapshotController": map[string]interface{}{
				"replicas": extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
				"podAnnotations": map[string]interface{}{
					"checksum/secret-" + vsphere.CSISnapshotControllerName: checksums[vsphere.CSISnapshotControllerName],
				},
			},
			"volumesnapshots": map[string]interface{}{
				"enabled": false, // not supported in vsphere-csi-driver v2.3.0
			},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["vsphere-cloud-controller-manager"].(map[string]interface{})["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	if cloudProfileConfig.FailureDomainLabels != nil {
		values["csi-vsphere"].(map[string]interface{})["labelRegion"] = cloudProfileConfig.FailureDomainLabels.Region
		values["csi-vsphere"].(map[string]interface{})["labelZone"] = cloudProfileConfig.FailureDomainLabels.Zone
	}

	return values, nil
}

// getControlPlaneShootChartValues collects and returns the control plane shoot chart values.
func (vp *valuesProvider) getControlPlaneShootChartValues(
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	credentials *vsphere.Credentials,
) (map[string]interface{}, error) {

	cloudProfileConfig, err := helper.GetCloudProfileConfig(cluster)
	if err != nil {
		return nil, err
	}

	region := helper.FindRegion(cluster.Shoot.Spec.Region, cloudProfileConfig)
	if region == nil {
		return nil, fmt.Errorf("region %q not found in cloud profile config", cluster.Shoot.Spec.Region)
	}

	insecureFlag := "false"
	if region.VsphereInsecureSSL {
		insecureFlag = "true"
	}

	serverName, port, err := splitServerNameAndPort(region.VsphereHost)
	if err != nil {
		return nil, err
	}

	_, csiClusterID := vp.calcClusterIDs(cp)
	values := map[string]interface{}{
		"csi-vsphere": map[string]interface{}{
			"serverName":        serverName,
			"clusterID":         csiClusterID,
			"username":          credentials.VsphereCSI().Username,
			"password":          credentials.VsphereCSI().Password,
			"serverPort":        port,
			"datacenters":       strings.Join(helper.CollectDatacenters(region), ","),
			"insecureFlag":      insecureFlag,
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		},
	}

	if cloudProfileConfig.FailureDomainLabels != nil {
		values["csi-vsphere"].(map[string]interface{})["labelRegion"] = cloudProfileConfig.FailureDomainLabels.Region
		values["csi-vsphere"].(map[string]interface{})["labelZone"] = cloudProfileConfig.FailureDomainLabels.Zone
	}

	return values, nil
}

func (vp *valuesProvider) calcClusterIDs(cp *extensionsv1alpha1.ControlPlane) (clusterID string, csiClusterID string) {
	clusterID = cp.Namespace + "-" + vp.gardenID
	csiClusterID = shortenID(clusterID, 63)
	return
}

func shortenID(id string, maxlen int) string {
	if maxlen < 16 {
		panic("maxlen < 16 for shortenID")
	}
	if len(id) <= maxlen {
		return id
	}

	hash := fnv.New64()
	_, _ = hash.Write([]byte(id))
	hashstr := strconv.FormatUint(hash.Sum64(), 36)
	return fmt.Sprintf("%s-%s", id[:62-len(hashstr)], hashstr)
}
