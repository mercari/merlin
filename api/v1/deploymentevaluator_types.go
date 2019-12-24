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

const (
	DeploymentEvaluatorMetadataName = "deployment-evaluator"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentEvaluatorSpec defines the desired state of DeploymentEvaluator
type DeploymentEvaluatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of DeploymentEvaluator. Edit DeploymentEvaluator_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// DeploymentEvaluatorStatus defines the observed state of DeploymentEvaluator
type DeploymentEvaluatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// DeploymentEvaluator is the Schema for the deploymentevaluators API
type DeploymentEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentEvaluatorSpec   `json:"spec,omitempty"`
	Status DeploymentEvaluatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentEvaluatorList contains a list of DeploymentEvaluator
type DeploymentEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentEvaluator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentEvaluator{}, &DeploymentEvaluatorList{})
}
