package controllers

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	// +kubebuilder:scaffold:imports

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/notifiers"
)

var _ = Describe("HPAControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRuleHPAInvalidScaleTargetRef", func() {
		var ruleStructName = GetStructName(merlinv1.ClusterRuleHPAInvalidScaleTargetRef{})
		var isNotifierCreated = false
		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-hpa-cluster-rule",
			},
			Spec: merlinv1.ClusterRuleHPAInvalidScaleTargetRefSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1.Notification{
					Notifiers: []string{notifier.Name},
				},
			},
		}

		min := int32(2)
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  "default",
				Name:       "test-hpa-for-cluster-rule-invalid-scale-ref",
				Generation: 0,
			},
			Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "non-exists",
				},
				MaxReplicas: 10,
				MinReplicas: &min,
			},
		}
		hpaNamespacedName := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
		alertKey := strings.Join([]string{ruleStructName, rule.Name, hpaNamespacedName.String()}, Separator)

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.cache.Notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyRule", func() {
			err := k8sClient.Create(ctx, &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to apply cluster rule")
			Eventually(func() []string {
				r := &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Finalizers
			}, time.Second*5, time.Millisecond*200).Should(ContainElement(FinalizerName))
		})

		It("TestCreateInvalidHPAShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, hpa)).Should(Succeed(), "Failed to create hpa")
			// alert should be added to notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*10, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestRemoveRuleShouldRemoveViolation", func() {
			Expect(k8sClient.Delete(ctx, rule)).Should(Succeed())
			// alert should be removed from notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))

		})

		It("TestRecreateRuleShouldGetViolationsForExistingHPA", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			alertKey := strings.Join([]string{ruleStructName, rule.Name, hpaNamespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")
			// alert should be added to notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestUpdateHPAToValidShouldRemoveViolation", func() {
			alertKey := strings.Join([]string{ruleStructName, rule.Name, hpaNamespacedName.String()}, Separator)
			min := int32(2)
			name := "nginx"
			labels := map[string]string{"app": name}
			Expect(k8sClient.Create(ctx, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: hpa.Namespace,
					Name:      name,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &min,
					Selector: &metav1.LabelSelector{MatchLabels: labels},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: labels},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: name, Image: "nginx:1.17.0"},
							},
						},
					},
				},
			})).Should(Succeed())
			hpa.Spec.ScaleTargetRef.Name = name
			Expect(k8sClient.Update(ctx, hpa)).Should(Succeed())
			// alert should be removed from notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})
})
