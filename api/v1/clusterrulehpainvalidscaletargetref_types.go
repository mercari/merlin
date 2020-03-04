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
// +kubebuilder:resource:scope=Cluster

// ClusterRuleHPAInvalidScaleTargetRef is the Schema for the cluster rule hpa invalid scale target refs API
type ClusterRuleHPAInvalidScaleTargetRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAInvalidScaleTargetRefSpec `json:"spec,omitempty"`
	Status RuleStatus                              `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRuleHPAInvalidScaleTargetRefList contains a list of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleHPAInvalidScaleTargetRef `json:"items"`
}

func (r ClusterRuleHPAInvalidScaleTargetRef) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}, notifiers map[string]*Notifier) error {
	l.Info("Evaluating HPA ScaleTargetRef validity")
	var hpas autoscalingv1.HorizontalPodAutoscalerList
	// cluster rule changed, check all HPA for new status
	if resource == nil {
		if err := cli.List(ctx, &hpas); err != nil {
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
		l.Info("Checking hpa", "hpa", hpa.Name)
		namespacedName := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
		msg, err := r.Spec.Notification.ParseMessage(namespacedName, GetStructName(hpa), "HPA has invalid scale target ref")
		if err != nil {
			return err
		}
		isViolated, err := r.EvaluateHPA(ctx, cli, l, hpa)
		if err != nil {
			return err
		}
		// need to check if namespaced ignore here in case user added it into the list, in such case we need to remove it.
		if isViolated && !IsStringInSlice(r.Spec.IgnoreNamespaces, hpa.Namespace) {
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

func (r ClusterRuleHPAInvalidScaleTargetRef) EvaluateHPA(ctx context.Context, cli client.Client, l logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) (isViolated bool, err error) {
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
	return !match, nil
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAInvalidScaleTargetRef{}, &ClusterRuleHPAInvalidScaleTargetRefList{})
}
