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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRuleServiceInvalidSelectorSpec defines the desired state of ClusterRuleServiceInvalidSelector
type ClusterRuleServiceInvalidSelectorSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
}

// +kubebuilder:object:root=true

// ClusterRuleServiceInvalidSelectorList contains a list of ClusterRuleServiceInvalidSelector
type ClusterRuleServiceInvalidSelectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleServiceInvalidSelector `json:"items"`
}

func (c ClusterRuleServiceInvalidSelectorList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRuleServiceInvalidSelector is the Schema for the clusterruleserviceinvalidselector API
type ClusterRuleServiceInvalidSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleServiceInvalidSelectorSpec `json:"spec,omitempty"`
	Status RuleStatus                            `json:"status,omitempty"`
}

func (c ClusterRuleServiceInvalidSelector) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	svc, ok := object.(corev1.Service)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %s", GetStructName(svc))
		return
	}
	l.Info("evaluating", GetStructName(svc), svc.Name)

	pods := corev1.PodList{}
	if err = cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     svc.Namespace,
		LabelSelector: labels.Set(svc.Spec.Selector).AsSelector(),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if len(pods.Items) <= 0 {
		isViolated = true
		message = "Service has no matched pods for the selector"
	} else {
		message = "Service has pods for the selector"
	}
	return
}

func (c ClusterRuleServiceInvalidSelector) GetStatus() RuleStatus {
	return c.Status
}

func (c ClusterRuleServiceInvalidSelector) List() RuleList {
	return &ClusterRuleServiceInvalidSelectorList{}
}

func (c ClusterRuleServiceInvalidSelector) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRuleServiceInvalidSelector) GetNamespacedRuleList() RuleList {
	return nil
}

func (c ClusterRuleServiceInvalidSelector) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRuleServiceInvalidSelector) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRuleServiceInvalidSelector) GetResourceList() ResourceList {
	return &coreV1ServiceList{}
}

func (c ClusterRuleServiceInvalidSelector) IsNamespacedRule() bool {
	return false
}

func (c ClusterRuleServiceInvalidSelector) GetSelector() *Selector {
	return nil
}

func (c ClusterRuleServiceInvalidSelector) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	svc, ok := object.(corev1.Service)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", svc)
		return
	}
	namespacedName = types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}
	return
}

func (c ClusterRuleServiceInvalidSelector) GetObjectMeta() metav1.ObjectMeta {
	return c.ObjectMeta
}

func (c *ClusterRuleServiceInvalidSelector) SetFinalizer(finalizer string) {
	c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, finalizer)
}

func (c *ClusterRuleServiceInvalidSelector) RemoveFinalizer(finalizer string) {
	removeString(c.ObjectMeta.Finalizers, finalizer)
}

func init() {
	SchemeBuilder.Register(&ClusterRuleServiceInvalidSelector{}, &ClusterRuleServiceInvalidSelectorList{})
}
