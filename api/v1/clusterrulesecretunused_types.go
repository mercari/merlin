/*

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRuleSecretUnusedSpec defines the desired state of ClusterRuleSecretUnused
type ClusterRuleSecretUnusedSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// InitialDelaySeconds is the delay time before the check is being run
	InitialDelaySeconds int64 `json:"initialDelaySeconds,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRuleSecretUnusedList contains a list of ClusterRuleSecretUnused
type ClusterRuleSecretUnusedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleSecretUnused `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleSecretUnused is the Schema for the clusterrulesecretunuseds API
type ClusterRuleSecretUnused struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterRuleSecretUnusedSpec `json:"spec,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ClusterRuleSecretUnused{}, &ClusterRuleSecretUnusedList{})
}
