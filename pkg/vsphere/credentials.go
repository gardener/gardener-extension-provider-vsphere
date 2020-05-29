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

package vsphere

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UserPass struct {
	Username string
	Password string
}

// Credentials contains the necessary vSphere credential information.
type Credentials struct {
	vsphere    *UserPass
	vsphereMCM *UserPass
	vsphereCCM *UserPass
	vsphereCSI *UserPass

	nsxt               *UserPass
	nsxtCCM            *UserPass
	nsxtInfrastructure *UserPass
}

func (c *Credentials) VsphereMCM() UserPass {
	if c.vsphereMCM != nil {
		return *c.vsphereMCM
	}
	return *c.vsphere
}

func (c *Credentials) VsphereCCM() UserPass {
	if c.vsphereCCM != nil {
		return *c.vsphereCCM
	}
	return *c.vsphere
}

func (c *Credentials) VsphereCSI() UserPass {
	if c.vsphereCSI != nil {
		return *c.vsphereCSI
	}
	return *c.vsphere
}

func (c *Credentials) NSXT_CCM() UserPass {
	if c.nsxtCCM != nil {
		return *c.nsxtCCM
	}
	return *c.nsxt
}

func (c *Credentials) NSXT_Infrastructure() UserPass {
	if c.nsxtInfrastructure != nil {
		return *c.nsxtInfrastructure
	}
	return *c.nsxt
}

// GetCredentials computes for a given context and infrastructure the corresponding credentials object.
func GetCredentials(ctx context.Context, c client.Client, secretRef corev1.SecretReference) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, c, &secretRef)
	if err != nil {
		return nil, err
	}
	return ExtractCredentials(secret)
}

func extractUserPass(secret *corev1.Secret, usernameKey, passwordKey string) (*UserPass, error) {
	username, ok := secret.Data[usernameKey]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", usernameKey)
	}

	password, ok := secret.Data[passwordKey]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", passwordKey)
	}

	return &UserPass{Username: string(username), Password: string(password)}, nil
}

// ExtractCredentials generates a credentials object for a given provider secret.
func ExtractCredentials(secret *corev1.Secret) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	vsphere, vsphereErr := extractUserPass(secret, Username, Password)

	mcm, err := extractUserPass(secret, UsernameMCM, PasswordMCM)
	if err != nil && vsphereErr != nil {
		return nil, fmt.Errorf("Need either common or machine controller manager specific vSphere account credentials: %s, %s", vsphereErr, err)
	}
	ccm, err := extractUserPass(secret, UsernameCCM, PasswordCCM)
	if err != nil && vsphereErr != nil {
		return nil, fmt.Errorf("Need either common or cloud controller manager specific vSphere account credentials: %s, %s", vsphereErr, err)
	}
	csi, err := extractUserPass(secret, UsernameCSI, PasswordCSI)
	if err != nil && vsphereErr != nil {
		return nil, fmt.Errorf("Need either common or CSI specific vSphere account credentials: %s, %s", vsphereErr, err)
	}

	nsxt, nsxtErr := extractUserPass(secret, NSXTUsername, NSXTPassword)
	if nsxtErr != nil {
		return nil, nsxtErr
	}
	nsxtCCM, err := extractUserPass(secret, NSXTUsernameCCM, NSXTPasswordCCM)
	if err != nil && nsxtErr != nil {
		return nil, fmt.Errorf("Need either common or cloud controller manager specific NSX-T account credentials: %s, %s", nsxtErr, err)
	}
	nsxtInfra, err := extractUserPass(secret, NSXTUsernameInfrastructure, NSXTPasswordInfrastructure)
	if err != nil && nsxtErr != nil {
		return nil, fmt.Errorf("Need either common or infrastructure specific NSX-T account credentials: %s, %s", nsxtErr, err)
	}

	return &Credentials{
		vsphere:            vsphere,
		vsphereMCM:         mcm,
		vsphereCCM:         ccm,
		vsphereCSI:         csi,
		nsxt:               nsxt,
		nsxtCCM:            nsxtCCM,
		nsxtInfrastructure: nsxtInfra,
	}, nil
}
