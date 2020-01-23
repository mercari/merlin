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
	NamespaceEvaluatorMetadataName = "namespace-evaluator"
)

// NamespaceEvaluatorSpec defines the desired state of NamespaceEvaluator
type NamespaceEvaluatorSpec struct {
	// IgnoreNamespaces is the list of namespaces (string) to ignore
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Rules is the list of checks to perform for the namespace
	Rules rules.NamespaceRules `json:"rules,omitempty"`
}

// NamespaceEvaluatorStatus defines the observed state of NamespaceEvaluator
type NamespaceEvaluatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// NamespaceEvaluator is the Schema for the namespaceevaluators API
type NamespaceEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceEvaluatorSpec   `json:"spec,omitempty"`
	Status NamespaceEvaluatorStatus `json:"status,omitempty"`
}

func (in *NamespaceEvaluator) IsNamespaceIgnored(namespace string) bool {
	return IsItemInSlice(namespace, in.Spec.IgnoreNamespaces)
}

// +kubebuilder:object:root=true

// NamespaceEvaluatorList contains a list of NamespaceEvaluator
type NamespaceEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceEvaluator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceEvaluator{}, &NamespaceEvaluatorList{})
}
