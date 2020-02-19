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
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"text/template"
	"time"
)

// NotifierSpec defines the desired state of Notifier
type NotifierSpec struct {
	// Slack is the notifier for slack
	Slack Slack `json:"slack,omitempty"`
}

// NotifierStatus defines the observed state of Notifier
type NotifierStatus struct {
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

func (n Notifier) Notify(message string) error {
	if n.Spec.Slack.Channel != "" {
		return n.Spec.Slack.SendMessage(message)
	}
	return nil
}

// +kubebuilder:object:root=true

// NotifierList contains a list of Notifier
type NotifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifier `json:"items"`
}

const defaultMessageFormat = "[{{.Severity}}] `{{.ResourceName}}` {{.DefaultMessage}}"

type MessageContents struct {
	Severity       string
	ResourceName   string
	DefaultMessage string
}

func (n NotifierList) NotifyAll(evaluationResult EvaluationResult, l logr.Logger) {

	for _, i := range evaluationResult.Issues {
		for _, n := range n.Items {
			for _, c := range i.Notification.Notifiers {
				channelFound := false
				if c == n.Name {
					channelFound = true
					severity := n.Spec.Slack.Severity
					if i.Notification.Severity != "" {
						severity = i.Notification.Severity
					}
					msgTemplate := defaultMessageFormat
					msgContents := MessageContents{
						Severity:       string(severity),
						ResourceName:   evaluationResult.NamespacedName.String(),
						DefaultMessage: i.DefaultMessage,
					}
					if i.Notification.CustomMessageTemplate != "" {
						msgTemplate = i.Notification.CustomMessageTemplate
					}
					t, err := template.New("msg").Parse(msgTemplate)
					if err != nil {
						l.Error(err, "failed to parse message template.", "template", msgTemplate)
					}
					var buf bytes.Buffer
					if err := t.Execute(&buf, msgContents); err != nil {
						l.Error(err, "failed to apply contents to template")
					}

					if err := n.Notify(buf.String()); err != nil {
						l.Error(err, "failed to send message to slack")
					}
				}
				if !channelFound {
					l.Error(fmt.Errorf("channel not found"), "channel", c)
				}
			}
		}
	}
}

type Slack struct {
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity IssueSeverity `json:"severity"`
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
