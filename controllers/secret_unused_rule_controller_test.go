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
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/notifiers"
)

var _ = Describe("SecretUnusedRuleControllerTests", func() {
	var ctx = context.Background()
	Context("TestClusterRuleSecretUnused", func() {
		var ruleStructName = GetStructName(merlinv1.ClusterRuleSecretUnused{})
		var isNotifierCreated = false
		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		rule := &merlinv1.ClusterRuleSecretUnused{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-rule-secret-unused",
			},
			Spec: merlinv1.ClusterRuleSecretUnusedSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1.Notification{
					Notifiers: []string{notifier.Name},
				},
				InitialDelaySeconds: 2,
			},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  "default",
				Name:       "secret-for-" + rule.Name,
				Generation: 0,
			},
			StringData: map[string]string{"secret": "test"},
		}
		namespacedName := types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
		alertKey := ruleStructName + Separator + rule.Name + Separator + namespacedName.String()

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

		It("TestApplyEmptyClusterRule", func() {
			err := k8sClient.Create(ctx, &merlinv1.ClusterRuleSecretUnused{})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.ClusterRuleSecretUnused{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyClusterRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())
		})

		It("TestCreateViolatedObjectShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestCreatePodWithSecretShouldNotGetViolation", func() {
			secretEnv := &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name}}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alpine",
					Namespace: secret.Namespace,
					Labels:    map[string]string{"app": "alpine"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "a",
							Image:   "alpine",
							Command: []string{"top"},
							EnvFrom: []corev1.EnvFromSource{{SecretRef: secretEnv}},
						},
					},
				}}
			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})

})
