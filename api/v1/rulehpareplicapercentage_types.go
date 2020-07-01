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

// RuleHPAReplicaPercentageSpec defines the desired state of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageSpec struct {
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Selector selects name or matched labels for a resource to apply this rule
	Selector Selector `json:"selector"`
	// Percent is the threshold of percentage for a HPA current replica divided by max replica to be considered as an issue.
	Percent int32 `json:"percent"`
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentageList contains a list of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleHPAReplicaPercentage `json:"items"`
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentage is the Schema for the rulehpareplicapercentage API
type RuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RuleHPAReplicaPercentageSpec `json:"spec,omitempty"`
}

func init() {
	SchemeBuilder.Register(&RuleHPAReplicaPercentage{}, &RuleHPAReplicaPercentageList{})
}
