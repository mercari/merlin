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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RuleHPAReplicaPercentageSpec defines the desired state of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageSpec struct {
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Selector selects name or matched labels for a resource to apply this rule
	Selector Selector `json:"selector"`
	// Percent is the threshold of percentage for a HPA current replica divided by max replica to be considered as an issue.
	Percent int32 `json:"percent"`
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentage is the Schema for the rulehpareplicapercentage API
type RuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleHPAReplicaPercentageSpec `json:"spec,omitempty"`
	Status RuleStatus                   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleHPAReplicaPercentageList contains a list of RuleHPAReplicaPercentage
type RuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleHPAReplicaPercentage `json:"items"`
}

func (r RuleHPAReplicaPercentage) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}, notifiers map[string]*Notifier) error {
	l.Info("Evaluating HPA replica percentage", "name", r.Name)
	var hpas autoscalingv1.HorizontalPodAutoscalerList
	// resource == nil is from rule changed, check resources for new status
	if resource == nil {
		if err := cli.List(ctx, &hpas, &client.ListOptions{Namespace: r.Namespace}); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No HPA found for evaluation", "rule", r.Name)
				return nil
			}
			return err
		}
	} else {
		hpa, ok := resource.(autoscalingv1.HorizontalPodAutoscaler)
		if !ok {
			return fmt.Errorf("unable to convert resource to hpa")
		}
		hpas.Items = append(hpas.Items, hpa)
	}
	for _, hpa := range hpas.Items {
		namespacedName := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
		msg, err := r.Spec.Notification.ParseMessage(namespacedName, GetStructName(hpa), fmt.Sprintf("HPA percentage is > %v%%", r.Spec.Percent))
		if err != nil {
			return err
		}
		if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(r.Spec.Percent)/100.0 {
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
	SchemeBuilder.Register(&RuleHPAReplicaPercentage{}, &RuleHPAReplicaPercentageList{})
}
