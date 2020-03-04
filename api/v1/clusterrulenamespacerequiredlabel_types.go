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
	apierrs "k8s.io/apimachinery/pkg/api/errors"
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
// +kubebuilder:resource:scope=Cluster

// ClusterRuleNamespaceRequiredLabel is the Schema for the clusterrulenamespacerequiredlabels API
type ClusterRuleNamespaceRequiredLabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleNamespaceRequiredLabelSpec `json:"spec,omitempty"`
	Status RuleStatus                            `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRuleNamespaceRequiredLabelList contains a list of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleNamespaceRequiredLabel `json:"items"`
}

func (r ClusterRuleNamespaceRequiredLabel) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}, notifiers map[string]*Notifier) error {
	l.Info("Evaluating Namespace required label", "name", r.Name)
	var namespaces corev1.NamespaceList
	// resource == nil is from rule changed, check resources for new status
	if resource == nil {
		if err := cli.List(ctx, &namespaces); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No namespaces found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		namespace, ok := resource.(corev1.Namespace)
		if !ok {
			return fmt.Errorf("unable to convert resource to type %s", corev1.Namespace{}.Kind)
		}
		namespaces.Items = append(namespaces.Items, namespace)
	}

	for _, ns := range namespaces.Items {
		namespacedName := types.NamespacedName{Namespace: ns.Namespace, Name: ns.Name}
		violation, err := r.Spec.Label.Validate(ns.GetLabels())
		if err != nil {
			return err
		}
		msg, err := r.Spec.Notification.ParseMessage(namespacedName, GetStructName(ns), violation)
		if err != nil {
			return err
		}
		// need to check if namespaced ignore here in case user added it into the list, in such case we need to remove it.
		if violation != "" && !IsStringInSlice(r.Spec.IgnoreNamespaces, ns.Name) {
			l.Info("resource has violation", "resource", namespacedName.String())
			r.Status.AddViolation(namespacedName)
			for _, n := range r.Spec.Notification.Notifiers {
				notifier, ok := notifiers[n]
				if !ok {
					l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
					continue
				}
				notifier.AddAlert(r.Kind, r.Name, namespacedName, msg)
			}
		} else {
			r.Status.RemoveViolation(namespacedName)
			for _, n := range r.Spec.Notification.Notifiers {
				notifier, ok := notifiers[n]
				if !ok {
					l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
					continue
				}
				notifier.RemoveAlert(r.Kind, r.Name, namespacedName, msg)
			}
		}
	}
	r.Status.SetCheckTime()
	if err := cli.Update(ctx, &r); err != nil {
		l.Error(err, "unable to update rule status", "rule", r.Name)
		return err
	}
	return nil
}
func init() {
	SchemeBuilder.Register(&ClusterRuleNamespaceRequiredLabel{}, &ClusterRuleNamespaceRequiredLabelList{})
}