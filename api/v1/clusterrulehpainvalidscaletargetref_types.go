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
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRuleHPAInvalidScaleTargetRefSpec defines the desired state of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
}

// +kubebuilder:object:root=true

// ClusterRuleHPAInvalidScaleTargetRefList contains a list of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleHPAInvalidScaleTargetRef `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleHPAInvalidScaleTargetRef is the Schema for the cluster rule hpa invalid scale target refs API
type ClusterRuleHPAInvalidScaleTargetRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAInvalidScaleTargetRefSpec `json:"spec,omitempty"`
	Status RuleStatus                              `json:"status,omitempty"`
}

func (r ClusterRuleHPAInvalidScaleTargetRef) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error {
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
		l.Info("Checking hpa", "hpa", hpa.Name)
		namespacedName := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
		hasMatch, err := r.HPAHasMatch(ctx, cli, l, hpa)
		if err != nil {
			return err
		}

		isViolated := false
		if !hasMatch && !IsStringInSlice(r.Spec.IgnoreNamespaces, hpa.Namespace) {
			l.Info("resource has violation", "resource", namespacedName.String())
			isViolated = true
		}
		r.Status.SetViolation(namespacedName, isViolated)
		newAlert := alert.Alert{
			Suppressed:       r.Spec.Notification.Suppressed,
			Severity:         r.Spec.Notification.Severity,
			MessageTemplate:  r.Spec.Notification.CustomMessageTemplate,
			ViolationMessage: "HPA has invalid scale target ref",
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

func (r ClusterRuleHPAInvalidScaleTargetRef) HPAHasMatch(ctx context.Context, cli client.Client, l logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) (hasMatch bool, err error) {
	match := false
	switch hpa.Spec.ScaleTargetRef.Kind {
	case "Deployment":
		deployments := appsv1.DeploymentList{}
		if err = cli.List(ctx, &deployments, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to list", "kind", deployments.Kind)
			return
		}
		for _, d := range deployments.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				match = true
				break
			}
		}
	case "ReplicaSet":
		replicaSets := appsv1.ReplicaSetList{}
		if err = cli.List(ctx, &replicaSets, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to list", "kind", replicaSets.Kind)
			return
		}
		for _, d := range replicaSets.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				match = true
				break
			}
		}
	default:
		err = fmt.Errorf("unknown HPA ScaleTargetRef kind")
		l.Error(err, "kind", hpa.Spec.ScaleTargetRef.Kind, "name", hpa.Spec.ScaleTargetRef.Name)
		return
	}
	return match, nil
}

func (r ClusterRuleHPAInvalidScaleTargetRef) GetStatus() RuleStatus {
	return r.Status
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAInvalidScaleTargetRef{}, &ClusterRuleHPAInvalidScaleTargetRefList{})
}
