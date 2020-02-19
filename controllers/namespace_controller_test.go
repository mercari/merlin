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
	var isNamespaceCreated = false
	var testNamespace = "testns"

	BeforeEach(func() {
		logf.Log.Info("Running test", "test", CurrentGinkgoTestDescription().FullTestText)
		if !isNamespaceCreated {
			err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}})
			Expect(err).NotTo(HaveOccurred())
		}
		isNamespaceCreated = true
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
		name := "test-ns-cluster-rule"
		notifierName := "test"
		Expect(k8sClient.Create(ctx, &merlinv1.ClusterRuleNamespaceRequiredLabel{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: merlinv1.ClusterRuleNamespaceRequiredLabelSpec{
				IgnoreNamespaces: []string{},
				Notification:     merlinv1.Notification{Notifiers: []string{notifierName}},
				Label:            merlinv1.RequiredLabel{Key: "istio-injection", Value: "enabled"},
			},
		})).Should(Succeed())

		rule := &merlinv1.ClusterRuleNamespaceRequiredLabel{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name}, rule)).Should(Succeed())
		Expect(rule.Name).To(Equal(name))
		Expect(rule.Spec.Notification.Notifiers[0]).To(Equal(notifierName))

		Eventually(func() string {
			n := &corev1.Namespace{}
			k8sClient.Get(ctx, types.NamespacedName{Name: testNamespace}, n)
			annotation, _ := n.ObjectMeta.Annotations[AnnotationIssue]
			return annotation
		}, time.Second*10, time.Millisecond*500).Should(Equal(string(merlinv1.IssueLabelNoRequiredLabel)))

	})
})
