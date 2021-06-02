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

var _ = Describe("ServiceControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRuleServiceInvalidSelector", func() {
		var ruleStructName = GetStructName(merlinv1beta1.ClusterRuleServiceInvalidSelector{})
		var isNotifierCreated = false
		var notifier = &merlinv1beta1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifiers"},
			Spec:       merlinv1beta1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1beta1.ClusterRuleServiceInvalidSelector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "svc-cluster-rule-invalid-selector",
			},
			Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1beta1.Notification{
					Notifiers: []string{notifier.Name},
				},
			},
		}
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  "default",
				Name:       "svc-for-invalid-selector",
				Generation: 0,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "invalid"},
				Ports: []corev1.ServicePort{
					{Port: 8080},
				},
			},
		}
		namespacedName := types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}
		alertKey := ruleStructName + Separator + rule.Name + Separator + namespacedName.String()

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.cache.notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyClusterRule", func() {
			err := k8sClient.Create(ctx, &merlinv1beta1.ClusterRuleServiceInvalidSelector{})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1beta1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1beta1.ClusterRuleServiceInvalidSelector{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyClusterRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())
		})

		It("TestCreateInvalidServiceShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, svc)).Should(Succeed())
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			// alert should be added to notifiers status
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
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

		It("TestRecreateRuleShouldGetViolationsForExistingService", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			alertKey := strings.Join([]string{ruleStructName, rule.Name, namespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")
			// alert should be added to notifiers status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestDeleteServiceShouldRemoveAlert", func() {
			Expect(k8sClient.Delete(ctx, svc)).Should(Succeed())
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1beta1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			// alert should be added to notifiers status
			Expect(notifierReconciler.cache.notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})

})
