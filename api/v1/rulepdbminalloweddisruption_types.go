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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RulePDBMinAllowedDisruptionSpec defines the desired state of RulePDBMinAllowedDisruption
type RulePDBMinAllowedDisruptionSpec struct {
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Selector selects name or matched labels for a resource to apply this rule
	Selector Selector `json:"selector"`
	// MinAllowedDisruption is the minimal allowed disruption for this rule, should be an integer, default to 1
	MinAllowedDisruption int `json:"minAllowedDisruption,omitempty"`
}

// +kubebuilder:object:root=true

// RulePDBMinAllowedDisruptionList contains a list of RulePDBMinAllowedDisruption
type RulePDBMinAllowedDisruptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RulePDBMinAllowedDisruption `json:"items"`
}

// +kubebuilder:object:root=true

// RulePDBMinAllowedDisruption is the Schema for the rulepdbminalloweddisruptions API
type RulePDBMinAllowedDisruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RulePDBMinAllowedDisruptionSpec `json:"spec,omitempty"`
	Status RuleStatus                      `json:"status,omitempty"`
}

func (r RulePDBMinAllowedDisruption) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}, notifiers map[string]*Notifier) error {
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

	minAllowedDisruption := 1
	if r.Spec.MinAllowedDisruption > minAllowedDisruption {
		minAllowedDisruption = r.Spec.MinAllowedDisruption
	}

	for _, p := range pdbs.Items {
		var err error
		var allowedDisruption int
		msg := "PDB has enough allowed disruption"
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
		msg, err = r.Spec.Notification.ParseMessage(namespacedName, GetStructName(p), fmt.Sprintf("PDB doesnt have enough disruption pod (expect %v, but currently is %v)", r.Spec.MinAllowedDisruption, allowedDisruption))
		if err != nil {
			return err
		}

		isViolated := false
		if allowedDisruption < minAllowedDisruption {
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

func (r RulePDBMinAllowedDisruption) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&RulePDBMinAllowedDisruption{}, &RulePDBMinAllowedDisruptionList{})
}
