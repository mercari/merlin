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
	"github.com/kouzoh/merlin/notifiers/alert"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
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

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRulePDBMinAllowedDisruption is the Schema for the clusterrulepdbminalloweddisruptions API
type ClusterRulePDBMinAllowedDisruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRulePDBMinAllowedDisruptionSpec `json:"spec,omitempty"`
	Status RuleStatus                             `json:"status,omitempty"`
}

func (r ClusterRulePDBMinAllowedDisruption) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error {
	l.Info("Evaluating", "name", r.Name, "rule", GetStructName(r))
	var pdbs policyv1beta1.PodDisruptionBudgetList

	// empty resource is from rule changed, check resources for new status
	if resource == (types.NamespacedName{}) {
		if err := cli.List(ctx, &pdbs); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No resources found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		pdb := policyv1beta1.PodDisruptionBudget{}
		err := cli.Get(ctx, resource, &pdb)
		if apierrs.IsNotFound(err) {
			// resource not found, wont add to the list, and removed it from alert
			l.Info("resource not found - event is from deletion", "name", resource.String())
			r.Status.SetViolation(resource, false)
			newAlert := alert.Alert{
				Suppressed:       r.Spec.Notification.Suppressed,
				Severity:         r.Spec.Notification.Severity,
				MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
				ViolationMessage: "recovered since resource is deleted",
				ResourceKind:     GetStructName(pdb),
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
			pdbs.Items = append(pdbs.Items, pdb)
		}
	}
	minAllowedDisruption := 1
	if r.Spec.MinAllowedDisruption > minAllowedDisruption {
		minAllowedDisruption = r.Spec.MinAllowedDisruption
	}

	for _, p := range pdbs.Items {
		var err error
		var allowedDisruption int
		namespacedName := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
		pods := corev1.PodList{}
		if err := cli.List(ctx, &pods, &client.ListOptions{
			Namespace: p.Namespace,
			Raw: &metav1.ListOptions{
				LabelSelector: labels.Set(p.Spec.Selector.MatchLabels).String(),
			},
		}); err != nil && client.IgnoreNotFound(err) != nil {
			return err
		}
		if p.Spec.MaxUnavailable != nil {
			allowedDisruption, err = intstr.GetValueFromIntOrPercent(p.Spec.MaxUnavailable, int(len(pods.Items)), true)
			if err != nil {
				return err
			}
		} else if p.Spec.MinAvailable != nil {
			var minAvailable int
			minAvailable, err := intstr.GetValueFromIntOrPercent(p.Spec.MinAvailable, int(len(pods.Items)), true)
			if err != nil {
				return err
			}
			allowedDisruption = len(pods.Items) - minAvailable
		}

		isViolated := false
		if allowedDisruption < minAllowedDisruption && !IsStringInSlice(r.Spec.IgnoreNamespaces, p.Namespace) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}

		r.Status.SetViolation(namespacedName, isViolated)
		newAlert := alert.Alert{
			Suppressed:       r.Spec.Notification.Suppressed,
			Severity:         r.Spec.Notification.Severity,
			MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
			ViolationMessage: fmt.Sprintf("PDB doesnt have enough disruption pod (expect %v, but currently is %v)", r.Spec.MinAllowedDisruption, allowedDisruption),
			ResourceKind:     GetStructName(p),
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

func (r ClusterRulePDBMinAllowedDisruption) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRulePDBMinAllowedDisruption{}, &ClusterRulePDBMinAllowedDisruptionList{})
}
