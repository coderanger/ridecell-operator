/*
Copyright 2020 Ridecell, Inc.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IAMRoleSpec defines the desired state of IAMRole
type IAMRoleSpec struct {
	RoleName                 string            `json:"roleName,omitempty"`
	InlinePolicies           map[string]string `json:"inlinePolicies,omitempty"`
	AssumeRolePolicyDocument string            `json:"assumeRolePolicyDocument,omitempty"`
	PermissionsBoundaryArn   string            `json:"permissionsBoundaryArn,omitempty"`
}

// IAMRoleStatus defines the observed state of IAMRole
type IAMRoleStatus struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	RoleName string `json:"roleName"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMRole is the Schema for the IAMRoles API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type IAMRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMRoleSpec   `json:"spec,omitempty"`
	Status IAMRoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMRoleList contains a list of IAMRole
type IAMRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IAMRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IAMRole{}, &IAMRoleList{})
}
