// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package secret_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-provider-vsphere/pkg/utils/index"
	secretutil "github.com/gardener/gardener-extension-provider-vsphere/pkg/utils/secret"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

func TestSecretUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extensions Util Secret Suite")
}

var _ = Describe("Secret", func() {

	Context("#IsSecretInUseByShoot", func() {
		const namespace = "namespace"

		var (
			scheme *runtime.Scheme
			client client.Client

			secret        *corev1.Secret
			secretBinding *gardencorev1beta1.SecretBinding
			shoot         *gardencorev1beta1.Shoot
		)

		JustBeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
			}
			secretBinding = &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretbinding",
					Namespace: namespace,
				},
				SecretRef: corev1.SecretReference{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
			}
			shoot = &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.ShootSpec{
					Provider: gardencorev1beta1.Provider{
						Type: "gcp",
					},
					SecretBindingName: ptr.To(secretBinding.Name),
				},
			}

			scheme = runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())
			Expect(gardencorev1beta1.AddToScheme(scheme)).NotTo(HaveOccurred())

			client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					secret,
					secretBinding,
					shoot,
				).
				WithIndex(&gardencorev1beta1.SecretBinding{}, index.SecretRefNamespaceField, index.SecretRefNamespaceIndexerFunc).
				WithIndex(&gardencorev1beta1.Shoot{}, index.SecretBindingNameField, index.SecretBindingNameIndexerFunc).
				Build()
		})

		It("should return false when the Secret is not used", func() {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					secret,
					secretBinding,
				).
				WithIndex(&gardencorev1beta1.SecretBinding{}, index.SecretRefNamespaceField, index.SecretRefNamespaceIndexerFunc).
				WithIndex(&gardencorev1beta1.Shoot{}, index.SecretBindingNameField, index.SecretBindingNameIndexerFunc).
				Build()

			isUsed, err := secretutil.IsSecretInUseByShoot(context.TODO(), client, secret, "gcp")
			Expect(isUsed).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return false when the Secret is in use but the provider does not match", func() {
			isUsed, err := secretutil.IsSecretInUseByShoot(context.TODO(), client, secret, "other")
			Expect(isUsed).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return true when the Secret is in use by Shoot with the given provider", func() {
			isUsed, err := secretutil.IsSecretInUseByShoot(context.TODO(), client, secret, "gcp")
			Expect(isUsed).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the Secret is in use by Shoot from another namespace", func() {
			BeforeEach(func() {
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "another-namespace",
					},
				}
				secretBinding = &gardencorev1beta1.SecretBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secretbinding",
						Namespace: namespace,
					},
					SecretRef: corev1.SecretReference{
						Name:      secret.Name,
						Namespace: secret.Namespace,
					},
				}
			})

			It("should return true", func() {
				isUsed, err := secretutil.IsSecretInUseByShoot(context.TODO(), client, secret, "gcp")
				Expect(isUsed).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
