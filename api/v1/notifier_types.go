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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kouzoh/merlin/alert"
	"github.com/kouzoh/merlin/alert/slack"
)

const (
	Separator = string(types.Separator)
)

// +kubebuilder:object:root=true

// NotifierList contains a list of Notifier
type NotifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifier `json:"items"`
}

// NotifierSpec defines the desired state of Notifier
type NotifierSpec struct {
	// NotifyInterval is the interval for notifier to check and sends notifications
	NotifyInterval int64 `json:"notifyInterval"`
	// Slack is the notifier for slack
	Slack slack.Spec `json:"slack,omitempty"`
	// PagerDuty will be another notifier for slack
}

// NotifierStatus defines the observed state of Notifier, example:
// status:
//   alerts:
//     <RuleKind>/<RuleName>/<ResourceNamespacedName>:
//       resourceKind: HorizontalPodAutoscaler
//       resourceName: default/nginx
//       severity: warning
//       status: firing
//       suppressed: false
//   checkedAt: 2006-01-02T15:04:05Z07:00
type NotifierStatus struct {
	// CheckedAt is the last check time of the notifier
	CheckedAt string `json:"checkedAt"`
	// Alerts are the map of alerts currently firing/pending for objects violate the rule
	Alerts map[string]alert.Alert `json:"alerts,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// Notifier is the Schema for the notifiers API
type Notifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifierSpec   `json:"spec,omitempty"`
	Status NotifierStatus `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Notifier{}, &NotifierList{})
}
