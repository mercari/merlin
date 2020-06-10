package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// +kubebuilder:scaffold:imports

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/notifiers"
)

var _ = Describe("PDBControllerTests", func() {
	var ctx = context.Background()

	Context("TestClusterRulePDBInvalidSelector", func() {
		var ruleStructName = GetStructName(merlinv1.ClusterRulePDBInvalidSelector{})
		var isNotifierCreated = false
		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1.ClusterRulePDBInvalidSelector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pdb-cluster-rule-invalid-selector",
			},
			Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1.Notification{
					Notifiers: []string{notifier.Name},
				},
			},
		}
		pdb := &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  "default",
				Name:       "pdb-for-invalid-selector",
				Generation: 0,
			},
			Spec: policyv1beta1.PodDisruptionBudgetSpec{
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "invalid"}},
				MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
			},
		}
		pdbNamespacedName := types.NamespacedName{Namespace: pdb.Namespace, Name: pdb.Name}
		alertKey := ruleStructName + Separator + rule.Name + Separator + pdbNamespacedName.String()

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.Cache.Notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyClusterRule", func() {
			err := k8sClient.Create(ctx, &merlinv1.ClusterRulePDBInvalidSelector{})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.ClusterRulePDBInvalidSelector{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyClusterRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())
		})

		It("TestCreateInvalidPDBShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, pdb)).Should(Succeed())
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRulePDBInvalidSelector{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestRemoveRuleShouldRemoveViolation", func() {
			Expect(k8sClient.Delete(ctx, rule)).Should(Succeed())
			// alert should be removed from notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))

		})

		It("TestRecreateRuleShouldGetViolationsForExistingPDB", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			rule.Status = merlinv1.RuleStatus{}
			alertKey := strings.Join([]string{ruleStructName, rule.Name, pdbNamespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRulePDBInvalidSelector{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
		})
	})

	Context("TestClusterRulePDBMinAllowedDisruption", func() {
		var ruleStructName = GetStructName(merlinv1.ClusterRulePDBMinAllowedDisruption{})
		var isNotifierCreated = false
		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		rule := &merlinv1.ClusterRulePDBMinAllowedDisruption{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pdb-cluster-rule-min-allowed-disruption",
			},
			Spec: merlinv1.ClusterRulePDBMinAllowedDisruptionSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1.Notification{
					Notifiers: []string{notifier.Name},
				},
				MinAllowedDisruption: 2,
			},
		}
		pdb := &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  "default",
				Name:       "pdb-for-cluster-rule-min-allowed-disruption",
				Generation: 0,
			},
			Spec: policyv1beta1.PodDisruptionBudgetSpec{
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "invalid"}},
				MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 2},
			},
		}
		pdbNamespacedName := types.NamespacedName{Namespace: pdb.Namespace, Name: pdb.Name}
		alertKey := ruleStructName + Separator + rule.Name + Separator + pdbNamespacedName.String()

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.Cache.Notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true
		})

		It("TestApplyEmptyClusterRule", func() {
			err := k8sClient.Create(ctx, &merlinv1.ClusterRulePDBMinAllowedDisruption{})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.ClusterRulePDBMinAllowedDisruption{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyClusterRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())
		})

		It("TestCreateViolatedPDBShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, pdb)).Should(Succeed())
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestRemoveRuleShouldRemoveViolation", func() {
			Expect(k8sClient.Delete(ctx, rule)).Should(Succeed())
			// alert should be removed from notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))

		})

		It("TestRecreateRuleShouldGetViolationsForExistingPDB", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			rule.Status = merlinv1.RuleStatus{}
			alertKey := strings.Join([]string{ruleStructName, rule.Name, pdbNamespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
		})

		It("TestCreateEnoughPodsForRuleShouldNotGetViolation", func() {
			labels := map[string]string{"app": "alpine"}
			for i := 0; i < pdb.Spec.MinAvailable.IntValue()+rule.Spec.MinAllowedDisruption; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: pdb.Namespace,
						Labels:    labels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "a",
								Image:   "alpine",
								Command: []string{"top"},
							},
						},
					}}
				pod.Name = fmt.Sprintf("test-%v", i)
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
			}
			pdb.Spec.Selector.MatchLabels = labels
			Expect(k8sClient.Update(ctx, pdb)).Should(Succeed())
			Eventually(func() map[string]string {
				r := &merlinv1.ClusterRulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(pdbNamespacedName.String()))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})

	Context("TestRulePDBMinAllowedDisruption", func() {
		var ruleStructName = GetStructName(merlinv1.RulePDBMinAllowedDisruption{})
		var isNotifierCreated = false
		var isNamespaceCreated = false
		var namespace = strings.ToLower(ruleStructName) + "-ns"

		var notifier = &merlinv1.Notifier{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(ruleStructName) + "-notifier"},
			Spec:       merlinv1.NotifierSpec{NotifyInterval: 1},
		}
		var rule = &merlinv1.RulePDBMinAllowedDisruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "pdb-rule-min-allowed-disruption",
			},
			Spec: merlinv1.RulePDBMinAllowedDisruptionSpec{
				Notification: merlinv1.Notification{
					Notifiers: []string{notifier.Name},
				},
				MinAllowedDisruption: 2,
			},
		}
		var pdb = &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  namespace,
				Name:       "pdb-for-rule-min-allowed-disruption",
				Generation: 0,
			},
			Spec: policyv1beta1.PodDisruptionBudgetSpec{
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "invalid"}},
				MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 2},
			},
		}
		pdbNamespacedName := types.NamespacedName{Namespace: pdb.Namespace, Name: pdb.Name}
		alertKey := ruleStructName + Separator + rule.Name + Separator + pdbNamespacedName.String()

		BeforeEach(func() {
			logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
			if !isNotifierCreated {
				Expect(k8sClient.Create(ctx, notifier)).Should(Succeed())
				Eventually(func() map[string]*notifiers.Notifier {
					return notifierReconciler.Cache.Notifiers
				}, time.Second*5, time.Millisecond*200).Should(HaveKey(notifier.Name))
			}
			isNotifierCreated = true

			if !isNamespaceCreated {
				Expect(k8sClient.Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:      namespace,
						Namespace: namespace,
					}})).Should(Succeed())
			}
			isNamespaceCreated = true
		})

		It("TestApplyEmptyRule", func() {
			err := k8sClient.Create(ctx, &merlinv1.RulePDBMinAllowedDisruption{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}})
			Expect(err).To(HaveOccurred())
			s, ok := err.(interface{}).(*errors.StatusError)
			Expect(ok).To(Equal(true))
			Expect(s.ErrStatus.Code).To(Equal(int32(422)))
			Expect(s.ErrStatus.Details.Group).To(Equal(merlinv1.GROUP))
			Expect(s.ErrStatus.Kind).To(Equal(merlinv1.RulePDBMinAllowedDisruption{}.Kind))
			Expect(s.ErrStatus.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(s.ErrStatus.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("TestApplyRule", func() {
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed())
		})

		It("TestCreateViolatedPDBShouldGetViolations", func() {
			Expect(k8sClient.Create(ctx, pdb)).Should(Succeed())
			Eventually(func() map[string]string {
				r := &merlinv1.RulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
		})

		It("TestRemoveRuleShouldRemoveViolation", func() {
			Expect(k8sClient.Delete(ctx, rule)).Should(Succeed())
			// alert should be removed from notifier status
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))

		})

		It("TestRecreateRuleShouldGetViolationsForExistingPDB", func() {
			rule.Name = rule.Name + "-recreate"
			rule.ResourceVersion = ""
			rule.Status = merlinv1.RuleStatus{}
			alertKey := strings.Join([]string{ruleStructName, rule.Name, pdbNamespacedName.String()}, Separator)
			Expect(k8sClient.Create(ctx, rule)).Should(Succeed(), "Failed to recreate rule")
			Eventually(func() map[string]string {
				r := &merlinv1.RulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).Should(HaveKey(pdbNamespacedName.String()))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).Should(HaveKey(alertKey))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).Should(HaveKey(alertKey))
		})

		It("TestCreateEnoughPodsForRuleShouldNotGetViolation", func() {
			labels := map[string]string{"app": "alpine"}
			for i := 0; i < pdb.Spec.MinAvailable.IntValue()+rule.Spec.MinAllowedDisruption; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: pdb.Namespace,
						Labels:    labels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "a",
								Image:   "alpine",
								Command: []string{"top"},
							},
						},
					}}
				pod.Name = fmt.Sprintf("test-%v", i)
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
			}
			pdb.Spec.Selector.MatchLabels = labels
			Expect(k8sClient.Update(ctx, pdb)).Should(Succeed())
			Eventually(func() map[string]string {
				r := &merlinv1.RulePDBMinAllowedDisruption{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: rule.Name}, r)).Should(Succeed())
				return r.Status.Violations
			}, time.Second*3, time.Millisecond*200).ShouldNot(HaveKey(pdbNamespacedName.String()))
			Eventually(func() map[string]alert.Alert {
				n := &merlinv1.Notifier{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: notifier.Name}, n)).Should(Succeed())
				return n.Status.Alerts
			}, time.Second*5, time.Millisecond*200).ShouldNot(HaveKey(alertKey))
			// alert should be added to notifier status
			Expect(notifierReconciler.Cache.Notifiers[notifier.Name].Resource.Status.Alerts).ShouldNot(HaveKey(alertKey))
		})
	})

})
