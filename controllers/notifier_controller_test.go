package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	// +kubebuilder:scaffold:imports

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

var _ = Describe("HPAControllerTests", func() {
	var ctx = context.Background()
	const notifierName = "test-notifier"
	const ruleKind = "testRuleKind"
	const ruleName = "testRuleName"

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
		Expect(k8sClient.Create(ctx, &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: notifierName},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		})).Should(Succeed(), "Failed to apply cluster rule")
		By("test notifier should be added into notifier reconciler's cache")
		Eventually(func() bool {
			_, ok := notifierReconciler.NotifiersCache[notifierName]
			return ok
		}, time.Second*2, time.Millisecond*200).Should(Equal(true))
	})

	It("TestAddMessageToNotifier", func() {
		testMsg := "test alert message"
		testResourceName := types.NamespacedName{Name: "testresource"}
		alertKey := ruleKind + Separator + ruleName + Separator + testResourceName.String()
		notifier := notifierReconciler.NotifiersCache[notifierName]
		notifier.AddAlert(ruleKind, ruleName, testResourceName, testMsg)
		By("Notifier should have the status")
		alert, ok := notifier.Status.Alerts[alertKey]
		Expect(ok).To(Equal(true))
		Expect(alert.Message).To(Equal(testMsg))
		Expect(alert.Status).To(Equal(merlinv1.AlertStatusPending))

		By("Notifier status should be updated to k8s")
		Eventually(func() merlinv1.Alert {
			n := &merlinv1.Notifier{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: notifierName}, n)).NotTo(HaveOccurred(), "Failed to get notifier")
			return n.Status.Alerts[alertKey]
		}, time.Second*3, time.Millisecond*200).Should(Equal(merlinv1.Alert{Message: testMsg, Status: merlinv1.AlertStatusPending}))
	})
})