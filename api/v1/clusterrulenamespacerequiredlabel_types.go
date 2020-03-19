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
	"github.com/go-logr/logr"
	"github.com/kouzoh/merlin/notifiers/alert"
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

// ClusterRuleNamespaceRequiredLabelList contains a list of ClusterRuleNamespaceRequiredLabel
type ClusterRuleNamespaceRequiredLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleNamespaceRequiredLabel `json:"items"`
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

func (r ClusterRuleNamespaceRequiredLabel) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error {
	l.Info("Evaluating Namespace required label", "name", r.Name)
	var namespaces corev1.NamespaceList

	// empty resource is from rule changed, check resources for new status
	if resource == (types.NamespacedName{}) {
		if err := cli.List(ctx, &namespaces); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No resources found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		ns := corev1.Namespace{}
		err := cli.Get(ctx, resource, &ns)
		if apierrs.IsNotFound(err) {
			// resource not found, wont add to the list, and removed it from alert
			l.Info("resource not found - event is from deletion", "name", resource.String())
			r.Status.SetViolation(resource, false)
			newAlert := alert.Alert{
				Suppressed:       r.Spec.Notification.Suppressed,
				Severity:         r.Spec.Notification.Severity,
				MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
				ViolationMessage: "recovered since resource is deleted",
				ResourceKind:     GetStructName(ns),
				ResourceName:     resource.String(),
			}
			for _, n := range r.Spec.Notification.Notifiers {
				notifier, ok := notifiers[n]
				if !ok {
					l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
					continue
				}
				notifier.SetAlert(r.Kind, r.Name, newAlert, false)
			}
		} else if err != nil {
			return err
		} else {
			namespaces.Items = append(namespaces.Items, ns)
		}
	}

	for _, ns := range namespaces.Items {
		namespacedName := types.NamespacedName{Namespace: ns.Namespace, Name: ns.Name}
		violation, err := r.Spec.Label.Validate(ns.GetLabels())
		if err != nil {
			return err
		}

		isViolated := false
		if violation != "" && !IsStringInSlice(r.Spec.IgnoreNamespaces, ns.Name) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}
		r.Status.SetViolation(namespacedName, isViolated)
		newAlert := alert.Alert{
			Suppressed:       r.Spec.Notification.Suppressed,
			Severity:         r.Spec.Notification.Severity,
			MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
			ViolationMessage: violation,
			ResourceKind:     GetStructName(ns),
			ResourceName:     namespacedName.String(),
		}
		for _, n := range r.Spec.Notification.Notifiers {
			notifier, ok := notifiers[n]
			if !ok {
				l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
				continue
			}
			notifier.SetAlert(r.Kind, r.Name, newAlert, isViolated)
		}
	}

	if err := cli.Update(ctx, &r); err != nil {
		l.Error(err, "unable to update rule status", "rule", r.Name)
		return err
	}
	return nil
}

func (r ClusterRuleNamespaceRequiredLabel) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRuleNamespaceRequiredLabel{}, &ClusterRuleNamespaceRequiredLabelList{})
}
