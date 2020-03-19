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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleServiceInvalidSelector is the Schema for the clusterruleserviceinvalidselector API
type ClusterRuleServiceInvalidSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleServiceInvalidSelectorSpec `json:"spec,omitempty"`
	Status RuleStatus                            `json:"status,omitempty"`
}

func (r ClusterRuleServiceInvalidSelector) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error {
	l.Info("Evaluating Service Endpoints", "name", r.Name)
	var svcs corev1.ServiceList

	// empty resource is from rule changed, check resources for new status
	if resource == (types.NamespacedName{}) {
		if err := cli.List(ctx, &svcs); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No resources found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		svc := corev1.Service{}
		err := cli.Get(ctx, resource, &svc)
		if apierrs.IsNotFound(err) {
			// resource not found, wont add to the list, and removed it from alert
			l.Info("resource not found - event is from deletion", "name", resource.String())
			r.Status.SetViolation(resource, false)
			newAlert := alert.Alert{
				Suppressed:       r.Spec.Notification.Suppressed,
				Severity:         r.Spec.Notification.Severity,
				MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
				ViolationMessage: "recovered since resource is deleted",
				ResourceKind:     GetStructName(svc),
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
			svcs.Items = append(svcs.Items, svc)
		}
	}

	for _, svc := range svcs.Items {
		namespacedName := types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}
		pods := corev1.PodList{}
		if err := cli.List(ctx, &pods, &client.ListOptions{
			Namespace:     svc.Namespace,
			LabelSelector: labels.Set(svc.Spec.Selector).AsSelector(),
		}); err != nil && client.IgnoreNotFound(err) != nil {
			return err
		}

		isViolated := false
		if len(pods.Items) == 0 && !IsStringInSlice(r.Spec.IgnoreNamespaces, svc.Namespace) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}

		r.Status.SetViolation(namespacedName, isViolated)
		newAlert := alert.Alert{
			Suppressed:       r.Spec.Notification.Suppressed,
			Severity:         r.Spec.Notification.Severity,
			MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
			ViolationMessage: "Service has no matched pods",
			ResourceKind:     GetStructName(svc),
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

	l.Info("updating rule", "status", r.Status)
	if err := cli.Update(ctx, &r); err != nil {
		l.Error(err, "unable to update rule status", "rule", r.Name)
		return err
	}
	return nil
}

func (r ClusterRuleServiceInvalidSelector) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRuleServiceInvalidSelector{}, &ClusterRuleServiceInvalidSelectorList{})
}
