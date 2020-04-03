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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRuleHPAReplicaPercentageSpec defines the desired state of ClusterRuleHPAReplicaPercentageSpec
type ClusterRuleHPAReplicaPercentageSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
	// Percent is the threshold of percentage for a HPA current replica divided by max replica to be considered as an issue.
	Percent int32 `json:"percent"`
}

// +kubebuilder:object:root=true

// ClusterRuleHPAReplicaPercentageList contains a list of ClusterRuleHPAReplicaPercentage
type ClusterRuleHPAReplicaPercentageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleHPAReplicaPercentage `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleHPAReplicaPercentage is the Schema for the cluster rule hpa replica percentages API
type ClusterRuleHPAReplicaPercentage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAReplicaPercentageSpec `json:"spec,omitempty"`
	Status RuleStatus                          `json:"status,omitempty"`
}

func (r ClusterRuleHPAReplicaPercentage) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error {
	l.Info("Evaluating", "name", r.Name, "rule", GetStructName(r))
	var hpas autoscalingv1.HorizontalPodAutoscalerList

	// resource == nil is from rule changed, check resources for new status
	if resource == (types.NamespacedName{}) {
		if err := cli.List(ctx, &hpas); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("No resources found for evaluation", "name", r.Name)
				return nil
			}
			return err
		}
	} else {
		hpa := autoscalingv1.HorizontalPodAutoscaler{}
		err := cli.Get(ctx, resource, &hpa)
		if apierrs.IsNotFound(err) {
			// resource not found, wont add to the list, and removed it from alert
			l.Info("resource not found - event is from deletion", "name", resource.String())
			r.Status.SetViolation(resource, false)
			newAlert := alert.Alert{
				Suppressed:       r.Spec.Notification.Suppressed,
				Severity:         r.Spec.Notification.Severity,
				MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
				ViolationMessage: "recovered since resource is deleted",
				ResourceKind:     GetStructName(hpa),
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
			hpas.Items = append(hpas.Items, hpa)
		}
	}
	for _, hpa := range hpas.Items {
		namespacedName := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
		isViolated := false
		if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(r.Spec.Percent)/100.0 && !IsStringInSlice(r.Spec.IgnoreNamespaces, hpa.Namespace) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}
		r.Status.SetViolation(namespacedName, isViolated)
		newAlert := alert.Alert{
			Suppressed:       r.Spec.Notification.Suppressed,
			Severity:         r.Spec.Notification.Severity,
			MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
			ViolationMessage: fmt.Sprintf("HPA percentage is > %v%%", r.Spec.Percent),
			ResourceKind:     GetStructName(hpa),
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

func (r ClusterRuleHPAReplicaPercentage) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAReplicaPercentage{}, &ClusterRuleHPAReplicaPercentageList{})
}
