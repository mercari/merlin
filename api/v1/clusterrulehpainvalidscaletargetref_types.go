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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRuleHPAInvalidScaleTargetRefSpec defines the desired state of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefSpec struct {
	// IgnoreNamespaces is the list of namespaces to ignore for this rule
	IgnoreNamespaces []string `json:"ignoreNamespaces,omitempty"`
	// Notification contains the channels and messages to send out to external system, such as slack or pagerduty.
	Notification Notification `json:"notification"`
}

// ClusterRuleHPAInvalidScaleTargetRefStatus defines the observed state of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterRuleHPAInvalidScaleTargetRef is the Schema for the clusterrulehpainvalidscaletargetrefs API
type ClusterRuleHPAInvalidScaleTargetRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleHPAInvalidScaleTargetRefSpec   `json:"spec,omitempty"`
	Status ClusterRuleHPAInvalidScaleTargetRefStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterRuleHPAInvalidScaleTargetRefList contains a list of ClusterRuleHPAInvalidScaleTargetRef
type ClusterRuleHPAInvalidScaleTargetRefList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleHPAInvalidScaleTargetRef `json:"items"`
}

func (r ClusterRuleHPAInvalidScaleTargetRef) Evaluate(ctx context.Context, cli client.Client, l logr.Logger, resource interface{}) *EvaluationResult {
	l.Info("Evaluating HPA ScaleTargetRef validity")
	evaluationResult := &EvaluationResult{}
	hpa, ok := resource.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to hpa type")
		return evaluationResult
	}
	match := false
	switch hpa.Spec.ScaleTargetRef.Kind {
	case "Deployment":
		deployments := appsv1.DeploymentList{}
		if err := cli.List(ctx, &deployments, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to list", "kind", deployments.Kind)
			evaluationResult.Err = err
			return evaluationResult
		}
		for _, d := range deployments.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				match = true
				break
			}
		}
	case "ReplicaSet":
		replicaSets := appsv1.ReplicaSetList{}
		if err := cli.List(ctx, &replicaSets, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to list", "kind", replicaSets.Kind)
			evaluationResult.Err = err
			return evaluationResult
		}
		for _, d := range replicaSets.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				match = true
				break
			}
		}
	default:
		l.Info("Unknown HPA ScaleTargetRef kind", "kind", hpa.Spec.ScaleTargetRef.Kind, "name", hpa.Spec.ScaleTargetRef.Name)
	}
	if !match {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Label:          IssueLabelInvalidScaleTargetRef,
			DefaultMessage: "HPA ScaleTargetRef is incorrect",
			Notification:   r.Spec.Notification,
		})
	}
	return evaluationResult
}

func init() {
	SchemeBuilder.Register(&ClusterRuleHPAInvalidScaleTargetRef{}, &ClusterRuleHPAInvalidScaleTargetRefList{})
}
