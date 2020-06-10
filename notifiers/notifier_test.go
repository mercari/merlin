package notifiers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kouzoh/merlin/alert"
	"github.com/kouzoh/merlin/alert/slack"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

func Test_Notifier(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	})
	ts := httptest.NewServer(m)
	defer ts.Close()
	notifierResource := &merlinv1.Notifier{
		Spec: merlinv1.NotifierSpec{
			Slack: slack.Spec{
				Severity:   alert.SeverityWarning,
				WebhookURL: ts.URL,
				Channel:    "test-channel",
			},
		},
	}
	notifier := Notifier{
		Resource: notifierResource,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	testAlertRuleA1 := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msg",
		ResourceKind: "test-kind",
		ResourceName: "test-name-A1",
		Violated:     true,
	}

	// test setting alert and notify, status becomes firing
	notifier.SetAlert("RuleA", testAlertRuleA1)
	notifier.Notify()
	testAlertRuleA1.Status = alert.StatusFiring
	assert.Equal(t, testAlertRuleA1, notifier.Resource.Status.Alerts["RuleA/test-name-A1"])

	// test adding more alerts
	testAlertRuleA2 := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msg",
		ResourceKind: "test-kind",
		ResourceName: "test-name-A2",
		Violated:     true,
	}
	testAlertRuleB := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msg",
		ResourceKind: "test-kind",
		ResourceName: "test-name-B",
		Violated:     true,
	}

	notifier.SetAlert("RuleA", testAlertRuleA2)
	notifier.SetAlert("RuleB", testAlertRuleB)
	testAlertRuleA2.Status = alert.StatusPending
	testAlertRuleB.Status = alert.StatusPending
	assert.Equal(t, testAlertRuleA2, notifier.Resource.Status.Alerts["RuleA/test-name-A2"])

	// test clear rule alerts should recover alerts for the rule
	msg := "clear alerts for RuleA"
	notifier.ClearRuleAlerts("RuleA", msg)
	testAlertRuleA1.Status = alert.StatusRecovering
	testAlertRuleA2.Status = alert.StatusRecovering
	testAlertRuleA1.Message = msg + " " + testAlertRuleA1.Message
	testAlertRuleA2.Message = msg + " " + testAlertRuleA2.Message
	assert.Equal(t, testAlertRuleA1, notifier.Resource.Status.Alerts["RuleA/test-name-A1"])
	assert.Equal(t, testAlertRuleA2, notifier.Resource.Status.Alerts["RuleA/test-name-A2"])
	assert.Equal(t, testAlertRuleB, notifier.Resource.Status.Alerts["RuleB/test-name-B"])

	// notify should send recovering alert and remove them, but will not remove other rules' alert
	notifier.Notify()
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleA/test-name-A1"])
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleA/test-name-A2"])
	testAlertRuleB.Status = alert.StatusFiring
	assert.Equal(t, testAlertRuleB, notifier.Resource.Status.Alerts["RuleB/test-name-B"])

	// clear all alerts should recover all alerts
	msg = "clear all alerts"
	notifier.ClearAllAlerts(msg)
	testAlertRuleB.Status = alert.StatusRecovering
	testAlertRuleB.Message = msg + " " + testAlertRuleB.Message
	assert.Equal(t, testAlertRuleB, notifier.Resource.Status.Alerts["RuleB/test-name-B"])

	// notify should remove the last recovered alert.
	notifier.Notify()
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleB/test-name-B"])
}
