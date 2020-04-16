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

func (c ClusterRuleHPAInvalidScaleTargetRefList) ListItems() []Rule {
	var items []Rule
	for _, i := range c.Items {
		items = append(items, &i)
	}
	return items
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ClusterRuleHPAInvalidScaleTargetRef is the Schema for the cluster rule hpa invalid scale target refs API
type ClusterRuleHPAInvalidScaleTargetRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAInvalidScaleTargetRefSpec `json:"spec,omitempty"`
	Status RuleStatus                              `json:"status,omitempty"`
}

func (c ClusterRuleHPAInvalidScaleTargetRef) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, object interface{}) (isViolated bool, message string, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	l.Info("evaluating", GetStructName(hpa), hpa.Name)

	var hasMatch bool
	switch hpa.Spec.ScaleTargetRef.Kind {
	case "Deployment":
		deployments := appsv1.DeploymentList{}
		if err = cli.List(ctx, &deployments, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to list", "kind", deployments.Kind)
			return
		}
		for _, d := range deployments.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				hasMatch = true
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
				hasMatch = true
				break
			}
		}
	default:
		err = fmt.Errorf("unknown HPA ScaleTargetRef kind")
		l.Error(err, "kind", hpa.Spec.ScaleTargetRef.Kind, "name", hpa.Spec.ScaleTargetRef.Name)
		return
	}

	if hasMatch {
		message = "HPA has valid scale target ref"
	} else {
		isViolated = true
		message = "HPA has invalid scale target ref"
	}
	return
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetStatus() RuleStatus {
	return c.Status
}

func (c ClusterRuleHPAInvalidScaleTargetRef) List() RuleList {
	return &ClusterRuleHPAInvalidScaleTargetRefList{}
}

func (c ClusterRuleHPAInvalidScaleTargetRef) IsNamespaceIgnored(namespace string) bool {
	return IsStringInSlice(c.Spec.IgnoreNamespaces, namespace)
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetNamespacedRuleList() RuleList {
	return nil
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetNotification() Notification {
	return c.Spec.Notification
}

func (c *ClusterRuleHPAInvalidScaleTargetRef) SetViolationStatus(name types.NamespacedName, isViolated bool) {
	c.Status.SetViolation(name, isViolated)
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetResourceList() ResourceList {
	return &autoscalingv1HPAList{}
}

func (c ClusterRuleHPAInvalidScaleTargetRef) IsNamespacedRule() bool {
	return false
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetSelector() *Selector {
	return nil
}

func (c ClusterRuleHPAInvalidScaleTargetRef) GetObjectNamespacedName(object interface{}) (namespacedName types.NamespacedName, err error) {
	hpa, ok := object.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", hpa)
		return
	}
	namespacedName = types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
	return
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAInvalidScaleTargetRef{}, &ClusterRuleHPAInvalidScaleTargetRefList{})
}
