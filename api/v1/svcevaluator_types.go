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
	SVCEvaluatorMetadataName = "svc-evaluator"
)

type SVCEvaluatorSpec struct {
	// IgnoreNamespaces is the list of namespaces (string) to ignore
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Rules is the list of checks to perform for the namespace
	Rules rules.ServiceRules `json:"rules,omitempty"`
}

type SVCEvaluatorStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

type SVCEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SVCEvaluatorSpec   `json:"spec,omitempty"`
	Status SVCEvaluatorStatus `json:"status,omitempty"`
}

func (in *SVCEvaluator) IsNamespaceIgnored(namespace string) bool {
	return IsItemInSlice(namespace, in.Spec.IgnoreNamespaces)
}

// +kubebuilder:object:root=true

type SVCEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SVCEvaluator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SVCEvaluator{}, &SVCEvaluatorList{})
}
