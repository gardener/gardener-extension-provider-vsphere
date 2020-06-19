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

package vsphere

import (
	"path/filepath"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

const (
	// Name is the name of the vSphere provider controller.
	Name = "provider-vsphere"

	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// MCMProviderVsphereImageName is the namne of the vSphere provider plugin image.
	MCMProviderVsphereImageName = "machine-controller-manager-provider-vsphere"
	// CloudControllerImageName is the name of the external vSphere CloudProvider image.
	CloudControllerImageName = "vsphere-cloud-controller-manager"

	// CSIAttacherImageName is the name of the CSI attacher image.
	CSIAttacherImageName = "csi-attacher"
	// CSINodeDriverRegistrarImageName is the name of the CSI driver registrar image.
	CSINodeDriverRegistrarImageName = "csi-node-driver-registrar"
	// CSIProvisionerImageName is the name of the CSI provisioner image.
	CSIProvisionerImageName = "csi-provisioner"
	// CSIDriverControllerImageName is the name of the CSI driver controller plugin image.
	CSIDriverControllerImageName = "vsphere-csi-driver-controller"
	// CSIDriverNodeImageName is the name of the CSI driver node plugin image.
	CSIDriverNodeImageName = "vsphere-csi-driver-node"
	// CSIDriverSyncerImageName is the name of the vSphere CSI Syncer image.
	CSIDriverSyncerImageName = "vsphere-csi-driver-syncer"
	// CSIResizerImageName is the name of the csi-resizer image.
	CSIResizerImageName = "csi-resizer"
	// LivenessProbeImageName is the name of the liveness-probe image.
	LivenessProbeImageName = "liveness-probe"

	// Host is a constant for the key in a cloud provider secret holding the VSphere host name
	Host = "vsphereHost"
	// Username is a constant for the key in a cloud provider secret holding the VSphere user name (optional, for all components)
	Username = "vsphereUsername"
	// Password is a constant for the key in a cloud provider secret holding the VSphere password (optional, for all components)
	Password = "vspherePassword"
	// Username is a constant for the key in a cloud provider secret holding the VSphere user name (specific for MachineControllerManager)
	UsernameMCM = "vsphereUsernameMCM"
	// Password is a constant for the key in a cloud provider secret holding the VSphere password (specific for MachineControllerManager)
	PasswordMCM = "vspherePasswordMCM"
	// Username is a constant for the key in a cloud provider secret holding the VSphere user name (specific for CloudControllerManager)
	UsernameCCM = "vsphereUsernameCCM"
	// Password is a constant for the key in a cloud provider secret holding the VSphere password (specific for CloudControllerManager)
	PasswordCCM = "vspherePasswordCCM"
	// Username is a constant for the key in a cloud provider secret holding the VSphere user name (specific for CSI)
	UsernameCSI = "vsphereUsernameCSI"
	// Password is a constant for the key in a cloud provider secret holding the VSphere password (specific for CSI)
	PasswordCSI = "vspherePasswordCSI"
	// InsecureSSL is a constant for the key in a cloud provider secret holding the boolean flag to allow insecure HTTPS connections to the VSphere host
	InsecureSSL = "vsphereInsecureSSL"

	// NSXTUsername is a constant for the key in a cloud provider secret holding the NSX-T user name with role 'Enterprise Admin' (optional, for all components)
	NSXTUsername = "nsxtUsername"
	// Password is a constant for the key in a cloud provider secret holding the NSX-T password for user with role 'Enterprise Admin'
	NSXTPassword = "nsxtPassword"
	// NSXTUsernameLBAdmin is a constant for the key in a cloud provider secret holding the NSX-T user name with role 'LB Admin' (needed for CloudControllerManager)
	NSXTUsernameLBAdmin = "nsxtUsernameLBAdmin"
	// NSXTPasswordLBAdmin is a constant for the key in a cloud provider secret holding the NSX-T password for user with role 'LB Admin'
	NSXTPasswordLBAdmin = "nsxtPasswordLBAdmin"
	// NSXTUsernameNE is a constant for the key in a cloud provider secret holding the NSX-T user name with role 'Network Engineer' (needed for infrastructure and IP pools address allocation in CloudControllerManager)
	NSXTUsernameNE = "nsxtUsernameNE"
	// NSXTPasswordNE is a constant for the key in a cloud provider secret holding the NSX-T password for user with role 'Network Engineer'
	NSXTPasswordNE = "nsxtPasswordNE"

	// CloudProviderConfig is the name of the configmap containing the cloud provider config.
	CloudProviderConfig = "cloud-provider-config"
	// CloudProviderConfigMapKey is the key storing the cloud provider config as value in the cloud provider configmap.
	CloudProviderConfigMapKey = "cloudprovider.conf"
	// SecretCSIVsphereConfig is a constant for the secret containing the CSI vSphere config.
	SecretCSIVsphereConfig = "csi-vsphere-config"
	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"
	// MachineControllerManagerVpaName is the name of the VerticalPodAutoscaler of the machine-controller-manager deployment.
	MachineControllerManagerVpaName = "machine-controller-manager-vpa"
	// MachineControllerManagerMonitoringConfigName is the name of the ConfigMap containing monitoring stack configurations for machine-controller-manager.
	MachineControllerManagerMonitoringConfigName = "machine-controller-manager-monitoring-config"

	// CloudControllerManagerName is the constant for the name of the CloudController deployed by the control plane controller.
	CloudControllerManagerName = "cloud-controller-manager"
	// CloudControllerManagerServerName is the constant for the name of the CloudController deployed by the control plane controller.
	CloudControllerManagerServerName = "cloud-controller-manager-server"
	// CSIProvisionerName is a constant for the name of the csi-provisioner component.
	CSIProvisionerName = "csi-provisioner"
	// CSIAttacherName is a constant for the name of the csi-attacher component.
	CSIAttacherName = "csi-attacher"
	// CSIResizerName is a constant for the name of the csi-resizer component.
	CSIResizerName = "csi-resizer"
	// VsphereCSIController is a constant for the name of the vsphere-csi-controller component.
	VsphereCSIController = "vsphere-csi-controller"
	// VsphereCSISyncer is a constant for the name of the vsphere-csi-syncer component.
	VsphereCSISyncer = "csi-syncer"
	// CSINodeName is a constant for the chart name for a CSI node deployment in the shoot.
	CSINodeName = "vsphere-csi-node"
	// CSIDriverName is a constant for the name of the csi-driver component.
	CSIDriverName = "csi-driver"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")

	// UsernamePrefix is a constant for the username prefix of components deployed by OpenStack.
	UsernamePrefix = extensionsv1alpha1.SchemeGroupVersion.Group + ":" + Name + ":"
)
