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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRulePDBMinAllowedDisruptionSpec defines the desired state of ClusterRulePDBMinAllowedDisruption
type ClusterRulePDBMinAllowedDisruptionSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// MinAllowedDisruption is the minimal allowed disruption for this rule, should be an integer, default to 1
	MinAllowedDisruption int `json:"minAllowedDisruption,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRulePDBMinAllowedDisruptionList contains a list of ClusterRulePDBMinAllowedDisruption
type ClusterRulePDBMinAllowedDisruptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRulePDBMinAllowedDisruption `json:"items"`
}

func (c ClusterRulePDBMinAllowedDisruptionList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRulePDBMinAllowedDisruption is the Schema for the clusterrulepdbminalloweddisruptions API
type ClusterRulePDBMinAllowedDisruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRulePDBMinAllowedDisruptionSpec `json:"spec,omitempty"`
	Status RuleStatus                             `json:"status,omitempty"`
}

func (c ClusterRulePDBMinAllowedDisruption) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	pdb, ok := object.(policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %s", GetStructName(pdb))
		return
	}
	l.Info("evaluating", GetStructName(pdb), pdb.Name)

	minAllowedDisruption := 1 // default value
	if c.Spec.MinAllowedDisruption > minAllowedDisruption {
		minAllowedDisruption = c.Spec.MinAllowedDisruption
	}

	var allowedDisruption int
	pods := corev1.PodList{}
	if err = cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     pdb.Namespace,
		LabelSelector: labels.SelectorFromSet(pdb.Spec.Selector.MatchLabels),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if pdb.Spec.MaxUnavailable != nil {
		if allowedDisruption, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MaxUnavailable, len(pods.Items), true); err != nil {
			return
		}
	} else if pdb.Spec.MinAvailable != nil {
		var minAvailable int
		if minAvailable, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MinAvailable, len(pods.Items), true); err != nil {
			return
		}
		allowedDisruption = len(pods.Items) - minAvailable
	}

	if allowedDisruption < minAllowedDisruption {
		isViolated = true
		message = fmt.Sprintf("PDB doesnt have enough disruption pod (expect %v, but currently is %v)", minAllowedDisruption, allowedDisruption)
	} else {
		message = fmt.Sprintf("PDB has enough disruption pod (expect %v, currently is %v)", minAllowedDisruption, allowedDisruption)
	}
	return
}

func (c ClusterRulePDBMinAllowedDisruption) GetStatus() RuleStatus {
	return c.Status
}

func (c ClusterRulePDBMinAllowedDisruption) List() RuleList {
	return &ClusterRulePDBMinAllowedDisruptionList{}
}

func (c ClusterRulePDBMinAllowedDisruption) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRulePDBMinAllowedDisruption) GetNamespacedRuleList() RuleList {
	return &RulePDBMinAllowedDisruptionList{}
}

func (c ClusterRulePDBMinAllowedDisruption) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRulePDBMinAllowedDisruption) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRulePDBMinAllowedDisruption) GetResourceList() ResourceList {
	return &policyv1beta1PDBList{}
}

func (c ClusterRulePDBMinAllowedDisruption) IsNamespacedRule() bool {
	return false
}

func (c ClusterRulePDBMinAllowedDisruption) GetSelector() *Selector {
	return nil
}

func (c ClusterRulePDBMinAllowedDisruption) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	pdb, ok := object.(policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", pdb)
		return
	}
	namespacedName = types.NamespacedName{Namespace: pdb.Namespace, Name: pdb.Name}
	return
}

func init() {
	SchemeBuilder.Register(&ClusterRulePDBMinAllowedDisruption{}, &ClusterRulePDBMinAllowedDisruptionList{})
}
