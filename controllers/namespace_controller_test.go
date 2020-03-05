package controllers

import (
	"context"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("NamespaceControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRuleNamespaceRequiredLabel", func() {
		const kubeSystemNamespace = "kube-system"
		var ruleStructName = GetStructName(merlinv1.ClusterRuleNamespaceRequiredLabel{})
		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1.ClusterRuleNamespaceRequiredLabel{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns-cluster-rule"},
			Spec: merlinv1.ClusterRuleNamespaceRequiredLabelSpec{
				IgnoreNamespaces: []string{kubeSystemNamespace},
				Notification:     merlinv1.Notification{Notifiers: []string{notifier.Name}},
				Label:            merlinv1.RequiredLabel{Key: "istio-injection", Value: "enabled"},
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
				Eventually(func() map[string]*merlinv1.Notifier {
					return notifierReconciler.Notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyClusterRuleNamespaceRequiredLabel", func() {
			err := k8sClient.Create(ctx, &merlinv1.ClusterRuleNamespaceRequiredLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.ToLower(CurrentGinkgoTestDescription().TestText),
				},
			})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.ClusterRuleNamespaceRequiredLabel{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyClusterRuleNamespaceRequiredLabel", func() {
			By("Create rule")
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())

			By("Rule can be fetched")
			r := &merlinv1.ClusterRuleNamespaceRequiredLabel{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rule.Name}, r)).Should(Succeed())
			Expect(r.Name).To(Equal(r.Name))
			Expect(r.Spec.Notification.Notifiers[0]).To(Equal(notifier.Name))

			By("Rule has alert")
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRuleNamespaceRequiredLabel{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(namespacedName.String()))

			By("Alert should be added to notifier status")
			Eventually(func() map[string]merlinv1.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.Notifiers[notifier.Name].Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestApplyClusterRuleNamespaceRequiredLabelForIgnoredNamespace", func() {
			ignoredAlertKey := strings.Join([]string{ruleStructName, rule.Name, "", kubeSystemNamespace}, Separator)
			n := &merlinv1.Notifier{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
			Expect(n.Status.Alerts).ShouldNot(HaveKey(ignoredAlertKey))
			Expect(notifierReconciler.Notifiers[notifier.Name].Status.Alerts).ShouldNot(HaveKey(ignoredAlertKey))
		})
	})

})
