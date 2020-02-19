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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// RuleHPAReplicaPercentageStatus defines the observed state of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageStatus struct {
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentage is the Schema for the rulehpareplicapercentage API
type RuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleHPAReplicaPercentageSpec   `json:"spec,omitempty"`
	Status RuleHPAReplicaPercentageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentageList contains a list of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleHPAReplicaPercentage `json:"items"`
}

func (r RuleHPAReplicaPercentage) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}) *EvaluationResult {
	l.Info("Evaluating HPA replica percentage", "name", r.Name)
	evaluationResult := &EvaluationResult{}
	hpa, ok := resource.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to hpa type")
		return evaluationResult
	}

	l.Info("percentage", "p", r.Spec.Percent)
	if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(r.Spec.Percent)/100.0 {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Label:          IssueLabelHighReplicaPercent,
			DefaultMessage: fmt.Sprintf("HPA current replica percentage is higher than %v", r.Spec.Percent),
			Notification:   r.Spec.Notification,
		})
	}
	return evaluationResult
}

func init() {
	SchemeBuilder.Register(&RuleHPAReplicaPercentage{}, &RuleHPAReplicaPercentageList{})
}
