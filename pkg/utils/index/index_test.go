// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package index_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/utils/index"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

func TestIndex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extensions Util Index Suite")
}

var _ = Describe("Index", func() {
	Context("#SecretRefNamespaceIndexerFunc", func() {
		It("should return empty slice for non SecretBinding", func() {
			actual := index.SecretRefNamespaceIndexerFunc(&corev1.Secret{})
			Expect(actual).To(Equal([]string{}))
		})

		It("should return secretRef.namespace for SecretBinding", func() {
			secretBinding := &gardencorev1beta1.SecretBinding{
				SecretRef: corev1.SecretReference{
					Namespace: "garden-dev",
				},
			}

			actual := index.SecretRefNamespaceIndexerFunc(secretBinding)
			Expect(actual).To(Equal([]string{"garden-dev"}))
		})
	})

	Context("#SecretBindingNameIndexerFunc", func() {
		It("should return empty slice for non Shoot", func() {
			actual := index.SecretBindingNameIndexerFunc(&corev1.Pod{})
			Expect(actual).To(BeEmpty())
		})

		It("should return empty slice for nil secretBindingName", func() {
			shoot := &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{},
			}

			actual := index.SecretBindingNameIndexerFunc(shoot)
			Expect(actual).To(BeEmpty())
		})

		It("should return spec.secretBindingName for Shoot", func() {
			shoot := &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					SecretBindingName: ptr.To("foo"),
				},
			}

			actual := index.SecretBindingNameIndexerFunc(shoot)
			Expect(actual).To(Equal([]string{"foo"}))
		})
	})
})
