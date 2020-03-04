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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"time"
)

type AlertStatusType string

const (
	AlertStatusPending    AlertStatusType = "pending"    // pending to send alert
	AlertStatusFiring     AlertStatusType = "firing"     // alert currently firing
	AlertStatusRecovering AlertStatusType = "recovering" // alert is recovering, but not yet notified to external systems
	AlertStatusError      AlertStatusType = "error"

	Separator = string(types.Separator)
)

var NotifierNotFoundErr = fmt.Errorf("notifier not found")

// NotifierSpec defines the desired state of Notifier
type NotifierSpec struct {
	// NotifyInterval is the interval for notifier to check and sends notifications
	NotifyInterval int64 `json:"notifyInterval"`
	// Slack is the notifier for slack
	Slack Slack `json:"slack,omitempty"`
	// PagerDuty will be another notifier for slack
}

// NotifierStatus defines the observed state of Notifier, example:
// CheckedAt: 2006-01-02T15:04:05Z07:00
// Alerts:
//   ClusterRuleHPA/ResourceNamespacedName1:
//     Status: firing
//     Message: <message>
//   ClusterRuleHPA/ResourceNamespacedName2:
//     Status: pending
//     Message: <msg>
type NotifierStatus struct {
	// CheckedAt is the last check time of the notifier
	CheckedAt string `json:"checkedAt"`
	// Alerts are the map of alerts currently firing/pending for objects violate the rule
	Alerts map[string]Alert `json:"alerts,omitempty"`
}

func (n NotifierStatus) ListAlerts() (list []string) {
	for k := range n.Alerts {
		list = append(list, k)
	}
	return list
}

type Alert struct {
	// Message is the message for the alert, should contain {{.ResourceName}} as only variable.
	Message string `json:"message"`
	// Status is the status of this rule, can be pending, firing, or recovered
	Status AlertStatusType `json:"status"`
	// Error is the err from any issues for sending message to external system
	Error string `json:"error"`
}

// +kubebuilder:object:generate=false
// AlertSeverity indicates the severity of the alert
type AlertSeverity string

const (
	AlertSeverityDefault  AlertSeverity = ""
	AlertSeverityFatal    AlertSeverity = "fatal"
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// Notifier is the Schema for the notifiers API
type Notifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifierSpec   `json:"spec,omitempty"`
	Status NotifierStatus `json:"status,omitempty"`
}

func (n *Notifier) Notify() {
	for name, alert := range n.Status.Alerts {
		if alert.Status == AlertStatusRecovering {
			if n.Spec.Slack.Channel != "" {
				if err := n.Spec.Slack.SendMessage(alert.Message); err != nil {
					alert.Status = AlertStatusError
					alert.Error = err.Error()
					continue
				}
			}
			// TODO: add pagerduty, note currently they cant co-exists. (should we allow both slack & pagerduty exists in same Notifier?)
			delete(n.Status.Alerts, name)
		}
		if alert.Status == AlertStatusPending {
			if n.Spec.Slack.Channel != "" {
				if err := n.Spec.Slack.SendMessage(alert.Message); err != nil {
					alert.Status = AlertStatusError
					alert.Error = err.Error()
					continue
				}
				n.Status.Alerts[name] = Alert{Message: alert.Message, Status: AlertStatusFiring}
			}
			// TODO: add pagerduty
		}
	}
	n.Status.CheckedAt = time.Now().Format(time.RFC3339)

}

func (n *Notifier) AddAlert(ruleKind, ruleName string, objectNamespacedName types.NamespacedName, message string) {
	alertName := ruleKind + Separator + ruleName + Separator + objectNamespacedName.String()
	if n.Status.Alerts == nil {
		n.Status.Alerts = map[string]Alert{}
	}
	if alert, ok := n.Status.Alerts[alertName]; !ok {
		n.Status.Alerts[alertName] = Alert{Message: message, Status: AlertStatusPending}
	} else {
		switch alert.Status {
		case AlertStatusFiring, AlertStatusPending, AlertStatusError:
			// do nothing
		case AlertStatusRecovering:
			// recovering alert gets fired again, set them back to firing.
			n.Status.Alerts[alertName] = Alert{Message: message, Status: AlertStatusFiring}
		}
	}
}

func (n *Notifier) RemoveAlert(ruleKind, ruleName string, objectNamespacedName types.NamespacedName, message string) {
	alertName := ruleKind + Separator + ruleName + Separator + objectNamespacedName.String()
	if n.Status.Alerts == nil {
		n.Status.Alerts = map[string]Alert{}
	}
	if _, ok := n.Status.Alerts[alertName]; ok {
		n.Status.Alerts[alertName] = Alert{Message: "[recovered]" + message, Status: AlertStatusRecovering}
	}
}

// +kubebuilder:object:root=true

// NotifierList contains a list of Notifier
type NotifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifier `json:"items"`
}

type Slack struct {
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity AlertSeverity `json:"severity"`
	// WebhookURL is the WebhookURL from slack
	WebhookURL string `json:"webhookURL"`
	// Channel is the slack channel this notification should use
	Channel string `json:"channel"`
}

type SlackRequest struct {
	Text      string `json:"text"`
	Channel   string `json:"channel"`
	IconEmoji string `json:"icon_emoji"`
	Username  string `json:"username"`
}

func (s *Slack) SendMessage(msg string) error {
	slackBody, _ := json.Marshal(SlackRequest{Text: msg, Channel: s.Channel, IconEmoji: ":merlin:", Username: "Merlin"})
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	if buf.String() != "ok" {
		return errors.New("non-ok response returned from Slack")
	}
	return nil
}

func init() {
	SchemeBuilder.Register(&Notifier{}, &NotifierList{})
}
