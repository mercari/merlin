package rules

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/mocks"
)

func Test_PDBInvalidSelectorRule_Basic(t *testing.T) {
	notification := merlinv1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1Rule := &merlinv1.ClusterRulePDBInvalidSelector{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
			Notification: notification,
		},
	}

	r := &PDBInvalidSelectorRule{resource: merlinv1Rule}
	assert.Equal(t, merlinv1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, merlinv1Rule, r.GetObject())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRulePDBInvalidSelector/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
	delay, err := r.GetDelaySeconds(&corev1.Namespace{})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
}

func Test_PDBInvalidSelectorRule_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &PDBInvalidSelectorRule{}
	notification := merlinv1.Notification{Notifiers: []string{"testNotifier"}}
	ruleKey := client.ObjectKey{Namespace: "", Name: "rule"}

	cases := []struct {
		desc      string
		key       client.ObjectKey
		mockCalls []*gomock.Call
		expect    alert.Alert
		resource  interface{}
		expectErr bool
	}{
		{
			desc: "non pdb should have error",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1.ClusterRulePDBInvalidSelector{}).
					Return(nil),
			},
			resource:  "non-pdb",
			expectErr: true,
		},
		{
			desc: "ignored namespace should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1.ClusterRulePDBInvalidSelector{}).
					SetArg(2, merlinv1.ClusterRulePDBInvalidSelector{
						Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
							Notification:     notification,
							IgnoreNamespaces: []string{"ignoredNS"},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ignoredNS", Name: "pdb"},
			},
			expect: alert.Alert{
				Message:      "namespace is ignored by the rule",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "ignoredNS/pdb",
			},
		},
		{
			desc: "selector with no matched pods should get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1.ClusterRulePDBInvalidSelector{}).
					SetArg(2, merlinv1.ClusterRulePDBInvalidSelector{
						Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pdb"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
			},
			expect: alert.Alert{
				Message:      "PDB has no matched pods for the selector",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/pdb",
				Violated:     true,
			},
		},
		{
			desc: "selector with matched pods should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1.ClusterRulePDBInvalidSelector{}).
					SetArg(2, merlinv1.ClusterRulePDBInvalidSelector{
						Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod2", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pdb"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
			},
			expect: alert.Alert{
				Message:      "PDB has pods for the selector",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/pdb",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			for _, call := range tc.mockCalls {
				call.Times(1)
			}
			r, err := ruleFactory.New(ctx, mockClient, log, tc.key)
			assert.NoError(tt, err)
			a, err := r.Evaluate(ctx, tc.resource)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				assert.Equal(tt, tc.expect, a)
			}
		})
	}
}

func Test_PDBInvalidSelectorRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	notification := merlinv1.Notification{Notifiers: []string{"testNotifier"}}
	r := &PDBInvalidSelectorRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1.ClusterRulePDBInvalidSelector{
			Spec: merlinv1.ClusterRulePDBInvalidSelectorSpec{
				Notification: notification,
			},
		},
	}

	cases := []struct {
		desc      string
		mockCalls []*gomock.Call
		expect    []alert.Alert
		expectErr bool
	}{
		{
			desc: "no resources should returns nil alerts",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{}).
					Return(nil),
			},
		},
		{
			desc: "pdb without pod should returns violated alert and with pods should not return violated alert",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{}).
					SetArg(1, policyv1beta1.PodDisruptionBudgetList{
						Items: []policyv1beta1.PodDisruptionBudget{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pdb"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test"},
									},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test2", Name: "pdb2"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test2"},
									},
								},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test2",
						LabelSelector: labels.Set(map[string]string{"app": "test2"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test2"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod2", Labels: map[string]string{"app": "test2"}}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "PDB has no matched pods for the selector",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "test/pdb",
					Violated:     true,
				},
				{
					Message:      "PDB has pods for the selector",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "test2/pdb2",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			for _, call := range tc.mockCalls {
				call.Times(1)
			}
			alerts, err := r.EvaluateAll(ctx)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				assert.Equal(tt, tc.expect, alerts)
			}
		})
	}
}
