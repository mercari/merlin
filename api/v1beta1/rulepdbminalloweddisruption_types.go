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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RulePDBMinAllowedDisruptionSpec defines the desired state of RulePDBMinAllowedDisruption
type RulePDBMinAllowedDisruptionSpec struct {
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Selector selects name or matched labels for a resource to apply this rule
	Selector Selector `json:"selector"`
	// MinAllowedDisruption is the minimal allowed disruption for this rule, should be an integer, default to 1
	MinAllowedDisruption int `json:"minAllowedDisruption,omitempty"`
}

// +kubebuilder:object:root=true

// RulePDBMinAllowedDisruptionList contains a list of RulePDBMinAllowedDisruption
type RulePDBMinAllowedDisruptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RulePDBMinAllowedDisruption `json:"items"`
}

// +kubebuilder:object:root=true

// RulePDBMinAllowedDisruption is the Schema for the rulepdbminalloweddisruptions API
type RulePDBMinAllowedDisruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RulePDBMinAllowedDisruptionSpec `json:"spec,omitempty"`
}

func init() {
	SchemeBuilder.Register(&RulePDBMinAllowedDisruption{}, &RulePDBMinAllowedDisruptionList{})
}
