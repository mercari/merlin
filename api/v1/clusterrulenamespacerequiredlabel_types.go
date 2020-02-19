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
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRuleNamespaceRequiredLabelSpec defines the desired state of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Label is the required label for this namespace, specified key, value, and a match
	Label RequiredLabel `json:"label"`
}

// ClusterRuleNamespaceRequiredLabelStatus defines the observed state of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleNamespaceRequiredLabel is the Schema for the clusterrulenamespacerequiredlabels API
type ClusterRuleNamespaceRequiredLabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleNamespaceRequiredLabelSpec   `json:"spec,omitempty"`
	Status ClusterRuleNamespaceRequiredLabelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRuleNamespaceRequiredLabelList contains a list of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleNamespaceRequiredLabel `json:"items"`
}

func (r ClusterRuleNamespaceRequiredLabel) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}) *EvaluationResult {
	l.Info("Evaluating HPA replica percentage", "name", r.Name)
	evaluationResult := &EvaluationResult{}
	namespace, ok := resource.(corev1.Namespace)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to type %s", corev1.Namespace{}.Kind)
		return evaluationResult
	}
	issue, err := r.Spec.Label.Validate(namespace.GetLabels())
	if err != nil {
		evaluationResult.Err = err
		return evaluationResult
	}

	if issue.Label != "" {
		issue.Notification = r.Spec.Notification
		evaluationResult.Issues = append(evaluationResult.Issues, issue)
	}

	return evaluationResult
}

func init() {
	SchemeBuilder.Register(&ClusterRuleNamespaceRequiredLabel{}, &ClusterRuleNamespaceRequiredLabelList{})
}
