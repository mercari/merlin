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

// ClusterRuleHPAReplicaPercentageSpec defines the desired state of ClusterRuleHPAReplicaPercentageSpec
type ClusterRuleHPAReplicaPercentageSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Percent is the threshold of percentage for a HPA current replica divided by max replica to be considered as an issue.
	Percent int32 `json:"percent"`
}

// +kubebuilder:object:root=true

// ClusterRuleHPAReplicaPercentageList contains a list of ClusterRuleHPAReplicaPercentage
type ClusterRuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleHPAReplicaPercentage `json:"items"`
}

func (c ClusterRuleHPAReplicaPercentageList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRuleHPAReplicaPercentage is the Schema for the cluster rule hpa replica percentages API
type ClusterRuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAReplicaPercentageSpec `json:"spec,omitempty"`
	Status RuleStatus                          `json:"status,omitempty"`
}

func (c ClusterRuleHPAReplicaPercentage) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	l.Info("evaluating", GetStructName(hpa), hpa.Name)

	if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(c.Spec.Percent)/100.0 {
		isViolated = true
		message = fmt.Sprintf("HPA percentage is >= %v%%", c.Spec.Percent)
	} else {
		message = fmt.Sprintf("HPA percentage is within threshold (< %v%%)", c.Spec.Percent)
	}
	return
}

func (c ClusterRuleHPAReplicaPercentage) GetStatus() RuleStatus {
	return c.Status
}

func (c ClusterRuleHPAReplicaPercentage) List() RuleList {
	return &ClusterRuleHPAReplicaPercentageList{}
}

func (c ClusterRuleHPAReplicaPercentage) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRuleHPAReplicaPercentage) GetNamespacedRuleList() RuleList {
	return &RuleHPAReplicaPercentageList{}
}

func (c ClusterRuleHPAReplicaPercentage) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRuleHPAReplicaPercentage) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRuleHPAReplicaPercentage) GetResourceList() ResourceList {
	return &autoscalingv1HPAList{}
}

func (c ClusterRuleHPAReplicaPercentage) IsNamespacedRule() bool {
	return false
}

func (c ClusterRuleHPAReplicaPercentage) GetSelector() *Selector {
	return nil
}

func (c ClusterRuleHPAReplicaPercentage) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	namespacedName = types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
	return
}

func (c ClusterRuleHPAReplicaPercentage) GetObjectMeta() metav1.ObjectMeta {
	return c.ObjectMeta
}

func (c *ClusterRuleHPAReplicaPercentage) SetFinalizer(finalizer string) {
	c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, finalizer)
}

func (c *ClusterRuleHPAReplicaPercentage) RemoveFinalizer(finalizer string) {
	removeString(c.ObjectMeta.Finalizers, finalizer)
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAReplicaPercentage{}, &ClusterRuleHPAReplicaPercentageList{})
}
