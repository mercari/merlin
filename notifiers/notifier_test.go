package notifiers

import (
	"fmt"
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
		Status: merlinv1.NotifierStatus{Alerts: map[string]alert.Alert{}},
	}
	notifier := Notifier{
		Resource: notifierResource,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	testAlertRuleAResourceA1 := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msgA1",
		ResourceKind: "test-kind",
		ResourceName: "test-resource/A1",
		Violated:     true,
	}

	// test setting alert and notify, status becomes firing
	notifier.SetAlert("RuleA", testAlertRuleAResourceA1)
	notifier.Notify()
	testAlertRuleAResourceA1.Status = alert.StatusFiring
	assert.Equal(t, testAlertRuleAResourceA1, notifier.Resource.Status.Alerts["RuleA/test-resource/A1"])

	// test adding more alerts
	testAlertRuleAResourceA2 := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msgA2",
		ResourceKind: "test-kind",
		ResourceName: "test-resource/A2",
		Violated:     true,
	}
	testAlertRuleBResourceB := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msgB",
		ResourceKind: "test-kind",
		ResourceName: "test-resource/B",
		Violated:     true,
	}
	testAlertRuleBResourceC := alert.Alert{
		Suppressed:   false,
		Severity:     alert.SeverityWarning,
		Message:      "test-msgC",
		ResourceKind: "test-kind",
		ResourceName: "test-resource/C",
		Violated:     true,
	}

	notifier.SetAlert("RuleA", testAlertRuleAResourceA2)
	notifier.SetAlert("RuleB", testAlertRuleBResourceB)
	notifier.SetAlert("RuleB", testAlertRuleBResourceC)
	testAlertRuleAResourceA2.Status = alert.StatusPending
	testAlertRuleBResourceB.Status = alert.StatusPending
	testAlertRuleBResourceC.Status = alert.StatusPending
	assert.Equal(t, testAlertRuleAResourceA2, notifier.Resource.Status.Alerts["RuleA/test-resource/A2"])

	// test clear rule alerts should recover alerts for the rule
	msg := "clear alerts for RuleA"
	notifier.ClearRuleAlerts("RuleA", msg)
	testAlertRuleAResourceA1.Status = alert.StatusRecovering
	testAlertRuleAResourceA2.Status = alert.StatusRecovering
	testAlertRuleAResourceA1.Message = msg + " " + testAlertRuleAResourceA1.Message
	testAlertRuleAResourceA2.Message = msg + " " + testAlertRuleAResourceA2.Message
	assert.Equal(t, testAlertRuleAResourceA1, notifier.Resource.Status.Alerts["RuleA/test-resource/A1"])
	assert.Equal(t, testAlertRuleAResourceA2, notifier.Resource.Status.Alerts["RuleA/test-resource/A2"])
	assert.Equal(t, testAlertRuleBResourceB, notifier.Resource.Status.Alerts["RuleB/test-resource/B"])
	assert.Equal(t, testAlertRuleBResourceC, notifier.Resource.Status.Alerts["RuleB/test-resource/C"])

	// notify should send recovering alert and remove them, but will not remove other rules' alert
	notifier.Notify()
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleA/test-resource/A1"])
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleA/test-resource/A2"])
	testAlertRuleBResourceB.Status = alert.StatusFiring
	testAlertRuleBResourceC.Status = alert.StatusFiring
	assert.Equal(t, testAlertRuleBResourceB, notifier.Resource.Status.Alerts["RuleB/test-resource/B"])
	assert.Equal(t, testAlertRuleBResourceC, notifier.Resource.Status.Alerts["RuleB/test-resource/C"])

	// clear resource alerts should recover alerts for the resource
	msg = "clear resource alerts"
	notifier.ClearResourceAlerts("test-resource/B", msg)
	testAlertRuleBResourceB.Status = alert.StatusRecovering
	testAlertRuleBResourceB.Message = msg + " " + testAlertRuleBResourceB.Message
	assert.Equal(t, testAlertRuleBResourceB, notifier.Resource.Status.Alerts["RuleB/test-resource/B"])

	// notify should send recovering alert and remove them, but will not remove other resources' alert
	notifier.Notify()
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleB/test-resource/B"])
	assert.Equal(t, testAlertRuleBResourceC, notifier.Resource.Status.Alerts["RuleB/test-resource/C"])

	// clear all alerts should recover all alerts
	msg = "clear all alerts"
	notifier.ClearAllAlerts(msg)
	testAlertRuleBResourceC.Status = alert.StatusRecovering
	testAlertRuleBResourceC.Message = msg + " " + testAlertRuleBResourceC.Message
	assert.Equal(t, testAlertRuleBResourceC, notifier.Resource.Status.Alerts["RuleB/test-resource/C"])

	// notify should remove the last recovered alert.
	notifier.Notify()
	assert.Empty(t, notifier.Resource.Status.Alerts["RuleC/test-resource/C"])
}

func Test_getAlertName(t *testing.T) {
	rule := "ruleKind/ruleName"
	resource := "resourceNamespace/resourceName"
	assert.Equal(t, fmt.Sprintf("%s/%s", rule, resource), getAlertName(rule, resource))
}

func Test_getRuleName(t *testing.T) {
	rule := "ruleKind/ruleName"
	resource := "resourceNamespace/resourceName"
	alertName := rule + Separator + resource
	assert.Equal(t, rule, getRuleName(alertName, resource))
}

func Test_getResourceName(t *testing.T) {
	rule := "ruleKind/ruleName"
	resource := "resourceNamespace/resourceName"
	alertName := rule + Separator + resource
	assert.Equal(t, resource, getResourceName(alertName))
}
