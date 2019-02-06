/*
Copyright 2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package components_test

import (
	"context"
	"fmt"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	encryptedsecretcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/encryptedsecret/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockKMSClient struct {
	kmsiface.KMSAPI
}

var _ = Describe("encryptedsecret Component", func() {

	It("is reconcilable", func() {
		// Adding so coveralls may complain less
		comp := encryptedsecretcomponents.NewEncryptedSecret()
		Expect(comp.IsReconcilable(ctx)).To(BeTrue())
	})

	It("runs basic reconcile", func() {
		comp := encryptedsecretcomponents.NewEncryptedSecret()
		mockKMS := &mockKMSClient{}
		comp.InjectKMSAPI(mockKMS)

		instance.Data = map[string]string{
			"TEST_VALUE0": "dGVzdDA=",
			"TEST_VALUE1": "dGVzdDE=",
			"TEST_VALUE2": "dGVzdDI=",
			"test_value3": "VEVTVDM=",
		}

		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(fetchSecret.Data["TEST_VALUE0"])).To(Equal("kmstest0"))
		Expect(string(fetchSecret.Data["TEST_VALUE1"])).To(Equal("kmstest1"))
		Expect(string(fetchSecret.Data["TEST_VALUE2"])).To(Equal("kmstest2"))
		Expect(string(fetchSecret.Data["test_value3"])).To(Equal("kmsTEST3"))
	})

	It("updates an existing secret", func() {
		comp := encryptedsecretcomponents.NewEncryptedSecret()
		mockKMS := &mockKMSClient{}
		comp.InjectKMSAPI(mockKMS)

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
			Data: map[string][]byte{
				"test_value": []byte("test"),
			},
		}
		err := ctx.Create(context.TODO(), newSecret)
		Expect(err).ToNot(HaveOccurred())
		// Overwrite that secret with new one
		instance.Data = map[string]string{"new_value": "dGVzdDE="}
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err = ctx.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		_, ok := fetchSecret.Data["test_value"]
		Expect(ok).To(BeFalse())

		val, ok := fetchSecret.Data["new_value"]
		Expect(ok).To(BeTrue())
		Expect(string(val)).To(Equal("kmstest1"))
	})

})

func (m *mockKMSClient) Decrypt(input *kms.DecryptInput) (*kms.DecryptOutput, error) {
	if len(input.CiphertextBlob) < 0 {
		return &kms.DecryptOutput{}, awserr.New(kms.ErrCodeInvalidCiphertextException, "awsmock_decrypt: Invalid cipher text", errors.New(""))
	}
	return &kms.DecryptOutput{Plaintext: []byte(fmt.Sprintf("kms%s", string(input.CiphertextBlob)))}, nil
}