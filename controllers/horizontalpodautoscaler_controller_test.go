package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"

	// +kubebuilder:scaffold:imports
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

var _ = Describe("HPAControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRuleHPAInvalidScaleTargetRef", func() {
		var notifierName = strings.ToLower("HPAControllerTests") + "-notifier"
		var testNotifierCreated = false
		const ruleName = "test-hpa-cluster-rule"
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
		alertKey := "ClusterRuleHPAInvalidScaleTargetRef" + Separator + ruleName + Separator + hpaNamespacedName.String()

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !testNotifierCreated {
				Expect(k8sClient.Create(ctx, &merlinv1.Notifier{
					ObjectMeta: metav1.ObjectMeta{Name: notifierName},
					Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
				})).Should(Succeed())
			}
			testNotifierCreated = true
		})

		It("TestApplyEmptyClusterRuleHPAInvalidScaleTargetRef", func() {
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

		It("TestApplyClusterRuleHPAInvalidScaleTargetRef", func() {
			Expect(k8sClient.Create(ctx, &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{
				ObjectMeta: metav1.ObjectMeta{
					Name: ruleName,
				},
				Spec: merlinv1.ClusterRuleHPAInvalidScaleTargetRefSpec{
					IgnoreNamespaces: []string{},
					Notification: merlinv1.Notification{
						Notifiers: []string{notifierName},
					},
				},
			})).Should(Succeed(), "Failed to apply cluster rule")
		})

		It("TestCreateInvalidHPAShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, hpa)).Should(Succeed(), "Failed to create hpa")
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: ruleName}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(hpaNamespacedName.String()))
			// alert should be added to notifier status
			Expect(notifierReconciler.NotifiersCache[notifierName].Status.Alerts).Should(HaveKey(alertKey))
			Eventually(func() map[string]merlinv1.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifierName}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
		})

		It("TestUpdateHPAToValidShouldRemoveViolation", func() {
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
			Expect(k8sClient.Update(ctx, hpa)).Should(Succeed(), "Failed to update hpa")
			// violation should be removed from rule status
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: ruleName}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(hpaNamespacedName.String()))
			// alert should be removed from notifier status
			Eventually(func() map[string]merlinv1.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifierName}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.NotifiersCache[notifierName].Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})
})