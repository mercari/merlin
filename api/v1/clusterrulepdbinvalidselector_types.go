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
	apierrs "k8s.io/apimachinery/pkg/api/errors"
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

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRulePDBInvalidSelector is the Schema for the clusterrulepdbinvalidselectors API
type ClusterRulePDBInvalidSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRulePDBInvalidSelectorSpec `json:"spec,omitempty"`
	Status RuleStatus                        `json:"status,omitempty"`
}

func (r ClusterRulePDBInvalidSelector) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}, notifiers map[string]*Notifier) error {
	l.Info("Evaluating PDB invalid selector", "name", r.Name)
	var pdbs policyv1beta1.PodDisruptionBudgetList
	// resource == nil is from rule changed, check resources for new status
	if resource == nil {
		if err := cli.List(ctx, &pdbs); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No pdbs found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		pdb, ok := resource.(policyv1beta1.PodDisruptionBudget)
		if !ok {
			return fmt.Errorf("unable to convert resource to type %s", policyv1beta1.PodDisruptionBudget{}.Kind)
		}
		pdbs.Items = append(pdbs.Items, pdb)
	}
	for _, p := range pdbs.Items {
		namespacedName := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
		msg, err := r.Spec.Notification.ParseMessage(namespacedName, GetStructName(p), "PDB has invalid selector")
		if err != nil {
			return err
		}
		pods := corev1.PodList{}
		if err := cli.List(ctx, &pods, &client.ListOptions{
			Namespace:     p.Namespace,
			LabelSelector: labels.Set(p.Spec.Selector.MatchLabels).AsSelector(),
		}); err != nil && client.IgnoreNotFound(err) != nil {
			return err
		}
		isViolated := false
		if len(pods.Items) == 0 && !IsStringInSlice(r.Spec.IgnoreNamespaces, p.Namespace) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}

		r.Status.SetViolation(namespacedName, isViolated)
		for _, n := range r.Spec.Notification.Notifiers {
			notifier, ok := notifiers[n]
			if !ok {
				l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
				continue
			}
			notifier.SetAlert(r.Kind, r.Name, namespacedName, msg, isViolated)
		}
	}

	if err := cli.Update(ctx, &r); err != nil {
		l.Error(err, "unable to update rule status", "rule", r.Name)
		return err
	}
	return nil
}

func (r ClusterRulePDBInvalidSelector) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRulePDBInvalidSelector{}, &ClusterRulePDBInvalidSelectorList{})
}
