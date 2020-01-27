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
	"github.com/kouzoh/merlin/rules"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodEvaluatorMetadataName = "pod-evaluator"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodEvaluatorSpec defines the desired state of PodEvaluator
type PodEvaluatorSpec struct {
	// Restarts is the number of restarts limit before we send out a notification
	Restarts int32 `json:"crashes,omitempty"`
	// IgnoreNamespaces is the list of namespaces (string) to ignore
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Rules is the list of checks to perform for the namespace
	Rules rules.PodRules `json:"rules,omitempty"`
}

// PodEvaluatorStatus defines the observed state of PodEvaluator
type PodEvaluatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PodEvaluator is the Schema for the podevaluators API
type PodEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodEvaluatorSpec   `json:"spec,omitempty"`
	Status PodEvaluatorStatus `json:"status,omitempty"`
}

func (in *PodEvaluator) IsNamespaceIgnored(namespace string) bool {
	return IsItemInSlice(namespace, in.Spec.IgnoreNamespaces)
}

// +kubebuilder:object:root=true

// PodEvaluatorList contains a list of PodEvaluator
type PodEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodEvaluator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodEvaluator{}, &PodEvaluatorList{})
}
