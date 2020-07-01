package controllers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	// +kubebuilder:scaffold:imports

	"github.com/kouzoh/merlin/alert"
	"github.com/kouzoh/merlin/alert/slack"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/notifiers"
)

var _ = Describe("NotifierControllerTests", func() {
	var ctx = context.Background()
	const (
		ruleName = "testRuleKind" + Separator + "testRuleName"
	)

	var testAlert = alert.Alert{
		Severity:     alert.SeverityInfo,
		Message:      "test alert message",
		ResourceKind: "hpa",
		ResourceName: "default/test-resource-for-notifiers",
		Violated:     true,
	}
	var alertKey = ruleName + Separator + testAlert.ResourceName
	var m = http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req := &slack.Request{}
		body, err := ioutil.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(json.Unmarshal(body, req)).Should(Succeed())
		Expect(req.Username).To(Equal("Merlin"))
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	})
	var ts = httptest.NewServer(m)

	var testNotifier = &merlinv1.Notifier{
		ObjectMeta: metav1.ObjectMeta{Name: "test-notifier"},
		Spec: merlinv1.NotifierSpec{
			NotifyInterval: 1,
			Slack: slack.Spec{
				WebhookURL: ts.URL,
				Channel:    "test",
			},
		},
	}

	It("TestApplyEmptyNotifier", func() {
		err := k8sClient.Create(ctx, &merlinv1.Notifier{})
		Expect(err).To(HaveOccurred())
		s, ok := err.(interface{}).(*errors.StatusError)
		Expect(ok).To(Equal(true))
		Expect(s.ErrStatus.Code).To(Equal(int32(422)))
		Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
		Expect(s.ErrStatus.Kind).To(Equal(merlinv1.Notifier{}.Kind))
		Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
	})

	It("TestApplyNotifier", func() {
		Expect(k8sClient.Create(ctx, testNotifier)).Should(Succeed())
		By("test notifier should be added into notifier reconciler's cache")
		Eventually(func() bool {
			_, ok := notifierReconciler.cache.Notifiers[testNotifier.Name]
			return ok
		}, time.Second*2, time.Millisecond*200).Should(Equal(true))
	})

	It("TestAddAlertToNotifier", func() {
		notifier := notifierReconciler.cache.Notifiers[testNotifier.Name]
		notifier.SetAlert(ruleName, testAlert)
		By("Notifier cache should have the status")
		a, ok := notifier.Resource.Status.Alerts[alertKey]
		Expect(ok).To(Equal(true))
		Expect(a).To(Equal(alert.Alert{
			Severity:     testAlert.Severity,
			ResourceKind: testAlert.ResourceKind,
			ResourceName: testAlert.ResourceName,
			Message:      testAlert.Message,
			Status:       alert.StatusPending,
			Violated:     true,
		}))

		By("Notifier status should be updated to k8s")
		Eventually(func() alert.Alert {
			n := &merlinv1.Notifier{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testNotifier.Name}, n)).Should(Succeed())
			return n.Status.Alerts[alertKey]
		}, time.Second*3, time.Millisecond*200).Should(Equal(alert.Alert{
			Severity:     testAlert.Severity,
			ResourceKind: testAlert.ResourceKind,
			ResourceName: testAlert.ResourceName,
			Message:      testAlert.Message,
			Status:       alert.StatusFiring,
			Violated:     true,
		}))

		By("Notifier cache should update the status")
		a, ok = notifier.Resource.Status.Alerts[alertKey]
		Expect(ok).To(Equal(true))
		Expect(a).To(Equal(alert.Alert{
			Severity:     testAlert.Severity,
			ResourceKind: testAlert.ResourceKind,
			ResourceName: testAlert.ResourceName,
			Message:      testAlert.Message,
			Status:       alert.StatusFiring,
			Violated:     true,
		}))

	})

	It("TestRemoveAlertFromNotifier", func() {
		notifier := notifierReconciler.cache.Notifiers[testNotifier.Name]
		testAlert.Violated = false
		notifier.SetAlert(ruleName, testAlert)
		expectAlert := alert.Alert{
			Severity:     testAlert.Severity,
			ResourceKind: testAlert.ResourceKind,
			ResourceName: testAlert.ResourceName,
			Message:      testAlert.Message,
			Status:       alert.StatusRecovering,
			Violated:     false,
		}
		By("Notifier cache should have new status")
		a, ok := notifier.Resource.Status.Alerts[alertKey]
		Expect(ok).To(Equal(true))
		Expect(a).To(Equal(expectAlert))

		By("Notifier status should be updated to k8s")
		Eventually(func() map[string]alert.Alert {
			n := &merlinv1.Notifier{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testNotifier.Name}, n)).Should(Succeed())
			return n.Status.Alerts
		}, time.Second*3, time.Millisecond*200).ShouldNot(HaveKey(alertKey))

		By("Notifier cache should remove the alert")
		Expect(notifier.Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
	})

	It("TestRemoveNotifier", func() {
		Expect(k8sClient.Delete(ctx, testNotifier)).Should(Succeed())
		Eventually(func() map[string]*notifiers.Notifier {
			return notifierReconciler.cache.Notifiers
		}, time.Second*2, time.Millisecond*200).ShouldNot(HaveKey(testNotifier.Name))
	})
})
