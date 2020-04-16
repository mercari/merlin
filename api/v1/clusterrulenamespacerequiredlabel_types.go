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
	"k8s.io/apimachinery/pkg/types"
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

// +kubebuilder:object:root=true

// ClusterRuleNamespaceRequiredLabelList contains a list of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleNamespaceRequiredLabel `json:"items"`
}

func (c ClusterRuleNamespaceRequiredLabelList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRuleNamespaceRequiredLabel is the Schema for the clusterrulenamespacerequiredlabels API
type ClusterRuleNamespaceRequiredLabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleNamespaceRequiredLabelSpec `json:"spec,omitempty"`
	Status RuleStatus                            `json:"status,omitempty"`
}

func (c ClusterRuleNamespaceRequiredLabel) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	namespace, ok := object.(corev1.Namespace)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", namespace)
		return
	}
	l.Info("evaluating", GetStructName(namespace), namespace.Name)

	if message, err = c.Spec.Label.Validate(namespace.GetLabels()); err != nil {
		return
	}
	if message != "" {
		isViolated = true
	}
	return
}

func (c ClusterRuleNamespaceRequiredLabel) GetStatus() RuleStatus {
	return c.Status
}

func (c ClusterRuleNamespaceRequiredLabel) List() RuleList {
	return &ClusterRuleNamespaceRequiredLabelList{}
}

func (c ClusterRuleNamespaceRequiredLabel) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRuleNamespaceRequiredLabel) GetNamespacedRuleList() RuleList {
	return nil
}

func (c ClusterRuleNamespaceRequiredLabel) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRuleNamespaceRequiredLabel) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRuleNamespaceRequiredLabel) GetResourceList() ResourceList {
	return &coreV1NamespaceList{}
}

func (c ClusterRuleNamespaceRequiredLabel) IsNamespacedRule() bool {
	return false
}

func (c ClusterRuleNamespaceRequiredLabel) GetSelector() *Selector {
	return nil
}

func (c ClusterRuleNamespaceRequiredLabel) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	namespace, ok := object.(corev1.Namespace)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", namespace)
		return
	}
	namespacedName = types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	return
}

func init() {
	SchemeBuilder.Register(&ClusterRuleNamespaceRequiredLabel{}, &ClusterRuleNamespaceRequiredLabelList{})
}
