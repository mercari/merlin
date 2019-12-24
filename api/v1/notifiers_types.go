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
	. "github.com/kouzoh/merlin/notifiers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NotifiersMetadataName = "notifiers"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NotifiersSpec defines the desired state of Notifiers
type NotifiersSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Slack is the notifier for slack
	Slack Slack `json:"slack,omitempty"`
	// TODO: add a wrapper to validate and send to all notifiers
}

// NotifiersStatus defines the observed state of Notifiers
type NotifiersStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// Notifiers is the Schema for the notifiers API
type Notifiers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifiersSpec   `json:"spec,omitempty"`
	Status NotifiersStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NotifiersList contains a list of Notifiers
type NotifiersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifiers `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notifiers{}, &NotifiersList{})
}
