package controllers

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// +kubebuilder:scaffold:imports

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
	"github.com/kouzoh/merlin/notifiers"
)

var _ = Describe("NamespaceControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRuleNamespaceRequiredLabel", func() {
		const kubeSystemNamespace = "kube-system"
		var ruleStructName = GetStructName(merlinv1beta1.ClusterRuleNamespaceRequiredLabel{})
		var notifier = &merlinv1beta1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifiers"},
			Spec:       merlinv1beta1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns-cluster-rule"},
			Spec: merlinv1beta1.ClusterRuleNamespaceRequiredLabelSpec{
				IgnoreNamespaces: []string{kubeSystemNamespace},
				Notification:     merlinv1beta1.Notification{Notifiers: []string{notifier.Name}},
				Label:            merlinv1beta1.RequiredLabel{Key: "istio-injection", Value: "enabled"},
			},
		}

		var isNamespaceCreated = false
		var isNotifierCreated = false
		var namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		}
		var namespacedName = types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
		var alertKey = strings.Join([]string{ruleStructName, rule.Name, namespacedName.String()}, Separator)

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNamespaceCreated {
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
			}
			isNamespaceCreated = true

			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.cache.notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyClusterRuleNamespaceRequiredLabel", func() {
			err := k8sClient.Create(ctx, &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.ToLower(CurrentGinkgoTestDescription().TestText),
				},
			})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1beta1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyRule", func() {
			By("Create rule ")
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to apply cluster rule")
			Eventually(func() []string {
				r := &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Finalizers
			}, time.Second*5, time.Millisecond*200).Should(ContainElement(FinalizerName))
			Expect(rule.Name).To(Equal(rule.Name))
			Expect(rule.Spec.Notification.Notifiers[0]).To(Equal(notifier.Name))

			By("Alert should be added to notifiers status")
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))

			By("Ignored Namespace should not have alert")
			ignoredAlertKey := strings.Join([]string{ruleStructName, rule.Name, "", kubeSystemNamespace}, Separator)
			n := &merlinv1beta1.Notifier{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
			Expect(n.Status.Alerts).ShouldNot(HaveKey(ignoredAlertKey))
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(ignoredAlertKey))
		})

		It("TestRemoveRuleShouldRemoveViolation", func() {
			Expect(k8sClient.Delete(ctx, rule)).Should(Succeed())
			// alert should be removed from notifiers status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})

		It("TestRecreateRuleShouldGetViolationsForExistingNamespace", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			alertKey := strings.Join([]string{ruleStructName, rule.Name, namespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")

			By("Alert should be added to notifiers status")
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})
	})
})
