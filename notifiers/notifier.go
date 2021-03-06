package notifiers

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/mercari/merlin/alert"
	"github.com/mercari/merlin/alert/slack"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
)

const Separator = merlinv1beta1.Separator

type Notifier struct {
	Resource     *merlinv1beta1.Notifier
	Client       *http.Client
	AlertMetrics *prometheus.GaugeVec
}

func (n *Notifier) Notify() {
	for name, a := range n.Resource.Status.Alerts {
		if a.Suppressed {
			continue
		}
		if a.Status != alert.StatusFiring { // wont send again if already firing
			if n.Resource.Spec.Slack.Channel != "" {
				slackClient := slack.NewClient(n.Client, n.Resource.Spec.Slack.Severity, n.Resource.Spec.Slack.WebhookURL, n.Resource.Spec.Slack.Channel)
				if err := slackClient.SendAlert(a); err != nil {
					a.Error = err.Error()
				} else {
					a.Error = ""
					if a.Status == alert.StatusPending || a.Status == "" {
						a.Status = alert.StatusFiring
					}
				}

			} else {
				// TODO: add pagerduty, note if they'll co-exists then we'll need other Status/Error fields for PagerDuty
			}
		}

		if a.Status == alert.StatusRecovering {
			delete(n.Resource.Status.Alerts, name)
		} else {
			n.Resource.Status.Alerts[name] = a
		}
		n.setPromLabel(name, a)
	}
	n.Resource.Status.CheckedAt = time.Now().Format(time.RFC3339)
}

func (n *Notifier) SetAlert(rule string, newAlert alert.Alert) {
	name := getAlertName(rule, newAlert.ResourceName)
	if newAlert.Severity == alert.SeverityDefault {
		if n.Resource.Spec.Slack.Severity != "" {
			newAlert.Severity = n.Resource.Spec.Slack.Severity
		}
	}

	if newAlert.Violated {
		if a, ok := n.Resource.Status.Alerts[name]; !ok {
			newAlert.Status = alert.StatusPending
		} else if a.Status == alert.StatusRecovering || a.Status == alert.StatusFiring {
			newAlert.Status = alert.StatusFiring
		}
		n.Resource.Status.Alerts[name] = newAlert
	} else {
		if a, ok := n.Resource.Status.Alerts[name]; ok {
			if a.Status == alert.StatusPending {
				delete(n.Resource.Status.Alerts, name)
			} else {
				newAlert.Status = alert.StatusRecovering
				n.Resource.Status.Alerts[name] = newAlert
			}
		}
	}
}

func (n *Notifier) ClearAllAlerts(message string) {
	for k := range n.Resource.Status.Alerts {
		newAlert := n.Resource.Status.Alerts[k]
		newAlert.Status = alert.StatusRecovering
		newAlert.Message = message + " " + n.Resource.Status.Alerts[k].Message
		n.Resource.Status.Alerts[k] = newAlert
	}
	return
}

func (n *Notifier) ClearRuleAlerts(rule, message string) {
	for name, a := range n.Resource.Status.Alerts {
		if rule == getRuleName(name, a.ResourceName) {
			newAlert := n.Resource.Status.Alerts[name]
			newAlert.Status = alert.StatusRecovering
			newAlert.Message = message + " " + n.Resource.Status.Alerts[name].Message
			n.Resource.Status.Alerts[name] = newAlert
		}
	}
	return
}

func (n *Notifier) ClearResourceAlerts(resource, message string) {
	for name := range n.Resource.Status.Alerts {
		if resource == getResourceName(name) {
			newAlert := n.Resource.Status.Alerts[name]
			newAlert.Status = alert.StatusRecovering
			newAlert.Message = message + " " + n.Resource.Status.Alerts[name].Message
			n.Resource.Status.Alerts[name] = newAlert
		}
	}
	return
}

func (n *Notifier) setPromLabel(alertName string, a alert.Alert) {
	names := strings.Split(alertName, Separator)
	if len(names) == 4 {
		label := prometheus.Labels{
			"rule":               names[0],
			"rule_name":          names[1],
			"resource_namespace": names[2],
			"resource_name":      names[3],
			"resource_kind":      a.ResourceKind,
		}
		if a.Violated {
			n.AlertMetrics.With(label).Set(1)
		} else {
			n.AlertMetrics.With(label).Set(0)
		}
	}
}

// alert example: <Rule>/<RuleName>/<ResourceNamespace>/<ResourceName>
func getAlertName(rule, resourceName string) string {
	return strings.Join([]string{rule, resourceName}, Separator)
}

func getRuleName(alertName, resourceName string) string {
	return strings.Replace(alertName, Separator+resourceName, "", -1)
}

func getResourceName(alertName string) string {
	names := strings.Split(alertName, Separator)
	return strings.Join(names[len(names)-2:], Separator)
}
