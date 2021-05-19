// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"context"
	"fmt"
	"net/http"

	vspherevalidation "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/validation"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"

	"github.com/gardener/gardener/extensions/pkg/util"
	secretutil "github.com/gardener/gardener/extensions/pkg/util/secret"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Shoot validates shoots
type Secret struct {
	client    client.Client
	apiReader client.Reader
	decoder   runtime.Decoder
	Logger    logr.Logger
}

// Handle implements Handler.Handle
func (s *Secret) Handle(ctx context.Context, req admission.Request) admission.Response {
	var (
		secret    = &corev1.Secret{}
		oldSecret = &corev1.Secret{}
	)

	if err := util.Decode(s.decoder, req.Object.Raw, secret); err != nil {
		s.Logger.Error(err, "failed to decode resource as secret", "kind", req.Kind, "namespace", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if len(req.OldObject.Raw) != 0 {
		if err := util.Decode(s.decoder, req.OldObject.Raw, oldSecret); err != nil {
			s.Logger.Error(err, "failed to decode old resource as secret", "kind", req.Kind, "namespace", req.Namespace, "name", req.Name)
			return admission.Errored(http.StatusBadRequest, err)
		}

		if equality.Semantic.DeepEqual(secret.Data, oldSecret.Data) {
			return admission.Allowed("no changes in the secret data")
		}
	}

	isInUse, err := secretutil.IsSecretInUseByShoot(ctx, s.client, secret, vsphere.Type)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if !isInUse {
		return admission.Allowed(fmt.Sprintf("secert not used by any %q shoot cluster", vsphere.Type))
	}

	if err := vspherevalidation.ValidateCloudProviderSecret(secret); err != nil {
		return admission.Denied(fmt.Sprintf("invalid secret: %v", err))
	}

	return admission.Allowed("valid secret")
}

// InjectClient injects the client.
func (s *Secret) InjectClient(c client.Client) error {
	s.client = c
	return nil
}

// InjectAPIReader injects the given apiReader into the validator.
func (s *Secret) InjectAPIReader(apiReader client.Reader) error {
	s.apiReader = apiReader
	return nil
}

// InjectScheme injects the scheme.
func (s *Secret) InjectScheme(_ *runtime.Scheme) error {
	s.decoder = scheme.Codecs.UniversalDeserializer()
	return nil
}
