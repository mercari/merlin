package controllers

import (
	"context"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("HPAControllerTests", func() {
	var ctx = context.Background()

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
				Name: "test-hpa-cluster-rule",
			},
			Spec: merlinv1.ClusterRuleHPAInvalidScaleTargetRefSpec{
				IgnoreNamespaces: []string{},
				Notification: merlinv1.Notification{
					Notifiers: []string{"test"},
				},
			},
		})).NotTo(HaveOccurred(), "Failed to apply cluster rule")
	})
})
