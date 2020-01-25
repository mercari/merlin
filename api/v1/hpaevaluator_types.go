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
	HPAEvaluatorMetadataName = "hpa-evaluator"
)

type HPAEvaluatorSpec struct {
	// IgnoreNamespaces is the list of namespaces (string) to ignore
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	//Rules are the rules to check for the evaluator
	Rules rules.HPARules `json:"rules,omitempty"`
}

type HPAEvaluatorStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

//  HPAEvaluator is the Schema for the hpaevaluators API
type HPAEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HPAEvaluatorSpec   `json:"spec,omitempty"`
	Status HPAEvaluatorStatus `json:"status,omitempty"`
}

func (in *HPAEvaluator) IsNamespaceIgnored(namespace string) bool {
	return IsItemInSlice(namespace, in.Spec.IgnoreNamespaces)
}

// +kubebuilder:object:root=true

type HPAEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HPAEvaluator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HPAEvaluator{}, &HPAEvaluatorList{})
}
