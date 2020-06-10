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
	"k8s.io/apimachinery/pkg/types"
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

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentageList contains a list of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleHPAReplicaPercentage `json:"items"`
}

func (r RuleHPAReplicaPercentageList) ListItems() []Rule {
	var items []Rule
	for _, i := range r.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RuleHPAReplicaPercentage is the Schema for the rulehpareplicapercentage API
type RuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleHPAReplicaPercentageSpec `json:"spec,omitempty"`
	Status RuleStatus                   `json:"status,omitempty"`
}

func (r RuleHPAReplicaPercentage) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	l.Info("evaluating", GetStructName(hpa), hpa.Name)

	if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(r.Spec.Percent)/100.0 {
		isViolated = true
		message = fmt.Sprintf("HPA percentage is >= %v%%", r.Spec.Percent)
	} else {
		message = fmt.Sprintf("HPA percentage is within threshold (< %v%%)", r.Spec.Percent)
	}
	return
}

func (r RuleHPAReplicaPercentage) GetStatus() RuleStatus {
	return r.Status
}

func (r RuleHPAReplicaPercentage) List() RuleList {
	return &RuleHPAReplicaPercentageList{}
}

func (r RuleHPAReplicaPercentage) IsNamespaceIgnored(namespace string) bool {
	return false
}

func (r RuleHPAReplicaPercentage) GetNamespacedRuleList() RuleList {
	return nil
}

func (r RuleHPAReplicaPercentage) GetNotification() Notification {
	return r.Spec.Notification
}

func (r *RuleHPAReplicaPercentage) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	r.Status.SetViolation(name, isViolated)
}

func (r RuleHPAReplicaPercentage) GetResourceList() ResourceList {
	return &autoscalingv1HPAList{}
}

func (r RuleHPAReplicaPercentage) IsNamespacedRule() bool {
	return true
}

func (r RuleHPAReplicaPercentage) GetSelector() *Selector {
	return &r.Spec.Selector
}

func (r RuleHPAReplicaPercentage) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	namespacedName = types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
	return
}

func (r RuleHPAReplicaPercentage) GetObjectMeta() metav1.ObjectMeta {
	return r.ObjectMeta
}

func (r *RuleHPAReplicaPercentage) SetFinalizer(finalizer string) {
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, finalizer)
}

func (r *RuleHPAReplicaPercentage) RemoveFinalizer(finalizer string) {
	removeString(r.ObjectMeta.Finalizers, finalizer)
}

func init() {
	SchemeBuilder.Register(&RuleHPAReplicaPercentage{}, &RuleHPAReplicaPercentageList{})
}
