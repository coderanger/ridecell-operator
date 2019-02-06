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

package components

import (
	"encoding/json"
	"fmt"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type iamUserComponent struct {
	iamAPI iamiface.IAMAPI
}

func NewIAMUser() *iamUserComponent {
	sess := session.Must(session.NewSession())
	iamService := iam.New(sess)
	return &iamUserComponent{iamAPI: iamService}
}

func (comp *iamUserComponent) InjectIAMAPI(iamapi iamiface.IAMAPI) {
	comp.iamAPI = iamapi
}

func (_ *iamUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *iamUserComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *iamUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.IAMUser)

	// Try to get our user, if it can't be found create it
	var user *iam.User
	getUserOutput, err := comp.iamAPI.GetUser(&iam.GetUserInput{UserName: aws.String(instance.Spec.UserName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != iam.ErrCodeNoSuchEntityException {
				return components.Result{}, errors.Wrapf(aerr, "iam_user: failed to get user")
			}
			// If user does not exist create it
			createUserOutput, err := comp.iamAPI.CreateUser(&iam.CreateUserInput{
				UserName:            aws.String(instance.Spec.UserName),
				PermissionsBoundary: aws.String(instance.Spec.PermissionsBoundaryArn),
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: failed to create user")
			}
			user = createUserOutput.User
		}
	} else {
		// If getUser did not return an error
		user = getUserOutput.User
	}

	// Get inline user policy names
	listUserPoliciesOutput, err := comp.iamAPI.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: user.UserName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_user: failed to list inline user policies")
	}

	userPolicies := map[string]string{}
	for _, userPolicyName := range listUserPoliciesOutput.PolicyNames {
		// Not actually in use at the moment.
		getUserPolicy, err := comp.iamAPI.GetUserPolicy(&iam.GetUserPolicyInput{
			PolicyName: userPolicyName,
			UserName:   user.UserName,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to get user policy")
		}
		userPolicies[aws.StringValue(getUserPolicy.PolicyName)] = aws.StringValue(getUserPolicy.PolicyDocument)
	}

	// If there is an inline policy that is not in the spec delete it
	for _, userPolicyName := range userPolicies {
		_, ok := instance.Spec.InlinePolicies[userPolicyName]
		if !ok {
			_, err = comp.iamAPI.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
				PolicyName: aws.String(userPolicyName),
				UserName:   user.UserName,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: failed to delete user policy")
			}
		}
	}

	// Update our user policies
	for policyName, rawPolicy := range instance.Spec.InlinePolicies {
		policyBytes, err := json.Marshal(rawPolicy)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to marshal policy into json")
		}
		inputUserPolicyDocument := string(policyBytes)
		_, err = comp.iamAPI.PutUserPolicy(&iam.PutUserPolicyInput{
			PolicyDocument: aws.String(inputUserPolicyDocument),
			PolicyName:     aws.String(policyName),
			UserName:       user.UserName,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to put user policy")
		}
	}

	fetchAccessKey := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s-access-key", instance.Name), Namespace: instance.Namespace}, fetchAccessKey)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to get access-key secret")
		}
		fetchAccessKey = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-access-key", instance.Name), Namespace: instance.Namespace}}
	}

	_, ok0 := fetchAccessKey.Data["access_key_id"]
	_, ok1 := fetchAccessKey.Data["secret_access_key"]

	if !ok0 || !ok1 {
		// Find any access keys related attached to this user
		existingAccessKeys, err := comp.iamAPI.ListAccessKeys(&iam.ListAccessKeysInput{UserName: user.UserName})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to list access keys")
		}
		// Delete access keys if they exist
		for _, accessKeyMeta := range existingAccessKeys.AccessKeyMetadata {
			_, err := comp.iamAPI.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				AccessKeyId: accessKeyMeta.AccessKeyId,
				UserName:    user.UserName,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: failed to delete access keys")
			}
		}

		// Make new access key and put it in a secret
		createAccessKeyOutput, err := comp.iamAPI.CreateAccessKey(&iam.CreateAccessKeyInput{UserName: user.UserName})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to create new access key")
		}
		fetchAccessKey.Data = make(map[string][]byte)
		fetchAccessKey.Data["access_key_id"] = []byte(aws.StringValue(createAccessKeyOutput.AccessKey.AccessKeyId))
		fetchAccessKey.Data["secret_access_key"] = []byte(aws.StringValue(createAccessKeyOutput.AccessKey.SecretAccessKey))

		_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, fetchAccessKey, func(existingObj runtime.Object) error {
			existing := existingObj.(*corev1.Secret)
			// Sync important fields.
			err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
			if err != nil {
				return errors.Wrapf(err, "iam_user: Failed to set controller reference")
			}
			existing.Labels = fetchAccessKey.Labels
			existing.Annotations = fetchAccessKey.Annotations
			existing.Type = fetchAccessKey.Type
			existing.Data = fetchAccessKey.Data
			return nil
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to create or update secret")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.IAMUser)
		instance.Status.Status = awsv1beta1.StatusReady
		instance.Status.Message = "User exists and has secret"
		return nil
	}}, nil
}