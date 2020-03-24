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
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"strings"
	"time"

	"github.com/kouzoh/merlin/notifiers/alert"
	"github.com/kouzoh/merlin/notifiers/slack"
)

const (
	Separator = string(types.Separator)
)

var NotifierNotFoundErr = fmt.Errorf("notifier not found")

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
	Slack slack.Slack `json:"slack,omitempty"`
	// PagerDuty will be another notifier for slack
}

// NotifierStatus defines the observed state of Notifier, example:
// CheckedAt: 2006-01-02T15:04:05Z07:00
// Alerts:
//   ClusterRuleHPA/ResourceNamespacedName1:
//     Status: firing
//     MessageTemplate: <message>
//   ClusterRuleHPA/ResourceNamespacedName2:
//     Status: pending
//     MessageTemplate: <msg>
type NotifierStatus struct {
	// CheckedAt is the last check time of the notifier
	CheckedAt string `json:"checkedAt"`
	// Alerts are the map of alerts currently firing/pending for objects violate the rule
	Alerts map[string]alert.Alert `json:"alerts,omitempty"`
}

func (n NotifierStatus) ListAlerts() (list []string) {
	for k := range n.Alerts {
		list = append(list, k)
	}
	return list
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// Notifier is the Schema for the notifiers API
type Notifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifierSpec   `json:"spec,omitempty"`
	Status NotifierStatus `json:"status,omitempty"`
}

func (n *Notifier) Notify(client *http.Client) {
	for name, a := range n.Status.Alerts {
		if a.Suppressed {
			continue
		}
		if a.Status != alert.StatusFiring { // wont send again if already firing
			if n.Spec.Slack.Channel != "" {
				err := n.Spec.Slack.SendAlert(client, a)
				if err != nil {
					a.Error = err.Error()
					a.Status = alert.StatusError
				} else {
					a.Error = ""
					if a.Status == alert.StatusPending {
						a.Status = alert.StatusFiring
					}
				}
			} else {
				// TODO: add pagerduty, note if they'll co-exists then we'll need other Status/Error fields for PagerDuty
			}

			if a.Status == alert.StatusRecovering {
				delete(n.Status.Alerts, name)
			} else {
				n.Status.Alerts[name] = a
			}
		}
	}
	n.Status.CheckedAt = time.Now().Format(time.RFC3339)
}

func (n *Notifier) SetAlert(ruleKind, ruleName string, newAlert alert.Alert, isViolated bool) {
	name := strings.Join([]string{ruleKind, ruleName, newAlert.ResourceName}, Separator)
	if n.Status.Alerts == nil {
		n.Status.Alerts = map[string]alert.Alert{}
	}

	if newAlert.Severity == alert.SeverityDefault {
		if n.Spec.Slack.Severity != "" {
			newAlert.Severity = n.Spec.Slack.Severity
		}
	}

	if isViolated {
		if a, ok := n.Status.Alerts[name]; !ok {
			newAlert.Status = alert.StatusPending
			n.Status.Alerts[name] = newAlert
		} else {
			switch a.Status {
			case alert.StatusFiring, alert.StatusPending, alert.StatusError:
				// do nothing
			case alert.StatusRecovering:
				// recovering alert gets fired again, set them back to firing.
				newAlert.Status = alert.StatusFiring
				n.Status.Alerts[name] = newAlert
			}
		}
	} else {
		if _, ok := n.Status.Alerts[name]; ok {
			newAlert.Status = alert.StatusRecovering
			n.Status.Alerts[name] = newAlert
		}
	}
}

func init() {
	SchemeBuilder.Register(&Notifier{}, &NotifierList{})
}
