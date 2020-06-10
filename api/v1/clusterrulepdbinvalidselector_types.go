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
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRulePDBInvalidSelectorSpec defines the desired state of ClusterRulePDBInvalidSelector
type ClusterRulePDBInvalidSelectorSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
}

// +kubebuilder:object:root=true

// ClusterRulePDBInvalidSelectorList contains a list of ClusterRulePDBInvalidSelector
type ClusterRulePDBInvalidSelectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRulePDBInvalidSelector `json:"items"`
}

func (c ClusterRulePDBInvalidSelectorList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRulePDBInvalidSelector is the Schema for the clusterrulepdbinvalidselectors API
type ClusterRulePDBInvalidSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRulePDBInvalidSelectorSpec `json:"spec,omitempty"`
	Status RuleStatus                        `json:"status,omitempty"`
}

func (c ClusterRulePDBInvalidSelector) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	pdb, ok := object.(policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", pdb)
		return
	}
	l.Info("evaluating", GetStructName(pdb), pdb.Name)

	pods := corev1.PodList{}
	if err = cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     pdb.Namespace,
		LabelSelector: labels.Set(pdb.Spec.Selector.MatchLabels).AsSelector(),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if len(pods.Items) <= 0 {
		isViolated = true
		message = "PDB has no matched pods for the selector"
	} else {
		message = "PDB has pods for the selector"
	}
	return
}

func (c ClusterRulePDBInvalidSelector) GetStatus() RuleStatus {
	return c.Status
}
func (c ClusterRulePDBInvalidSelector) List() RuleList {
	return &ClusterRulePDBInvalidSelectorList{}
}

func (c ClusterRulePDBInvalidSelector) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRulePDBInvalidSelector) GetNamespacedRuleList() RuleList {
	return nil
}

func (c ClusterRulePDBInvalidSelector) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRulePDBInvalidSelector) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRulePDBInvalidSelector) GetResourceList() ResourceList {
	return &policyv1beta1PDBList{}
}

func (c ClusterRulePDBInvalidSelector) IsNamespacedRule() bool {
	return false
}

func (c ClusterRulePDBInvalidSelector) GetSelector() *Selector {
	return nil
}

func (c ClusterRulePDBInvalidSelector) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	pdb, ok := object.(policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", pdb)
		return
	}
	namespacedName = types.NamespacedName{Namespace: pdb.Namespace, Name: pdb.Name}
	return
}

func (c ClusterRulePDBInvalidSelector) GetObjectMeta() metav1.ObjectMeta {
	return c.ObjectMeta
}

func (c *ClusterRulePDBInvalidSelector) SetFinalizer(finalizer string) {
	c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, finalizer)
}

func (c *ClusterRulePDBInvalidSelector) RemoveFinalizer(finalizer string) {
	removeString(c.ObjectMeta.Finalizers, finalizer)
}

func init() {
	SchemeBuilder.Register(&ClusterRulePDBInvalidSelector{}, &ClusterRulePDBInvalidSelectorList{})
}
