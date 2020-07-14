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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gardener/controller-manager-library/pkg/utils"
	apisvsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/helper"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure/task"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gutils "github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/chart"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
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
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
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
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
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
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.VsphereCSIController,
					CommonName:   vsphere.UsernamePrefix + vsphere.VsphereCSIController,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         vsphere.VsphereCSISyncer,
					CommonName:   vsphere.UsernamePrefix + vsphere.VsphereCSISyncer,
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
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
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
		}
	},
}

var configChart = &chart.Chart{
	Name: "cloud-provider-config",
	Path: filepath.Join(vsphere.InternalChartsPath, "cloud-provider-config"),
	Objects: []*chart.Object{
		{Type: &corev1.ConfigMap{}, Name: vsphere.CloudProviderConfig},
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
				{Type: &corev1.ConfigMap{}, Name: vsphere.CloudControllerManagerName + "-monitoring-config"},
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
				vsphere.LivenessProbeImageName},
			Objects: []*chart.Object{
				{Type: &corev1.Secret{}, Name: vsphere.SecretCSIVsphereConfig},
				{Type: &appsv1.Deployment{}, Name: vsphere.VsphereCSIController},
				{Type: &autoscalingv1beta2.VerticalPodAutoscaler{}, Name: vsphere.VsphereCSIController + "-vpa"},
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
				// csi-syncer
				{Type: &rbacv1.ClusterRole{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncer},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncer},
				{Type: &rbacv1.Role{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncer},
				{Type: &rbacv1.RoleBinding{}, Name: vsphere.UsernamePrefix + vsphere.VsphereCSISyncer},
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

	secretCSIVsphereConfig := &corev1.Secret{}
	if err := vp.Client().Get(ctx, kutil.Key(cp.Namespace, vsphere.SecretCSIVsphereConfig), secretCSIVsphereConfig); err == nil {
		checksums[vsphere.SecretCSIVsphereConfig] = gutils.ComputeChecksum(secretCSIVsphereConfig.Data)
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

	return map[string]interface{}{
		"storagePolicyName": cloudProfileConfig.DefaultClassStoragePolicyName,
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

	defaultClass, loadBalancersClasses, err := validation.OverwriteLoadBalancerClasses(cloudProfileConfig.Constraints.LoadBalancerConfig.Classes, cpConfig)
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
		"datacenters":  strings.Join(helper.CollectDatacenters(region), ","),
		"username":     credentials.VsphereCCM().Username,
		"password":     credentials.VsphereCCM().Password,
		"loadbalancer": loadBalancer,
		"nsxt": map[string]interface{}{
			"host":         region.NSXTHost,
			"insecureFlag": region.VsphereInsecureSSL,
			"username":     credentials.NSXT_LBAdmin().Username,
			"password":     credentials.NSXT_LBAdmin().Password,
			"remoteAuth":   region.NSXTRemoteAuth,
		},
	}

	if credentials.NSXT_LBAdmin().Username != credentials.NSXT_NetworkEngineer().Username {
		values["nsxt"].(map[string]interface{})["usernameNE"] = credentials.NSXT_NetworkEngineer().Username
		values["nsxt"].(map[string]interface{})["passwordNE"] = credentials.NSXT_NetworkEngineer().Password
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

	clusterID := cp.Namespace + "-" + vp.gardenID
	csiResizerEnabled := false
	// for v2.0.0
	// csiResizerEnabled := cloudProfileConfig.CSIResizerDisabled == nil || !*cloudProfileConfig.CSIResizerDisabled
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
				"checksum/configmap-" + vsphere.CloudProviderConfig:           checksums[vsphere.CloudProviderConfig],
			},
			"podLabels": map[string]interface{}{
				v1beta1constants.LabelPodMaintenanceRestart: "true",
			},
		},
		"csi-vsphere": map[string]interface{}{
			"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"serverName":        serverName,
			"clusterID":         clusterID,
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
				"checksum/secret-" + vsphere.VsphereCSIController:             checksums[vsphere.VsphereCSIController],
				"checksum/secret-" + vsphere.VsphereCSISyncer:                 checksums[vsphere.VsphereCSISyncer],
				"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
				"checksum/secret-" + vsphere.SecretCSIVsphereConfig:           checksums[vsphere.SecretCSIVsphereConfig],
			},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["vsphere-cloud-controller-manager"].(map[string]interface{})["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
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

	clusterID := cp.Namespace + "-" + vp.gardenID
	values := map[string]interface{}{
		"csi-vsphere": map[string]interface{}{
			"serverName":        serverName,
			"clusterID":         clusterID,
			"username":          credentials.VsphereCSI().Username,
			"password":          credentials.VsphereCSI().Password,
			"serverPort":        port,
			"datacenters":       strings.Join(helper.CollectDatacenters(region), ","),
			"insecureFlag":      insecureFlag,
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		},
	}

	return values, nil
}
