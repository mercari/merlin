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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/mocks"
)

func Test_NamespaceRequiredLabelRule_Basic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1beta1Rule := &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1beta1.ClusterRuleNamespaceRequiredLabelSpec{
			Notification: notification,
		},
	}

	r := &NamespaceRequiredLabelRule{resource: merlinv1beta1Rule}
	assert.Equal(t, merlinv1beta1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, merlinv1beta1Rule, r.GetObject())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRuleNamespaceRequiredLabel/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
	delay, err := r.GetDelaySeconds(&corev1.Namespace{})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
}

func Test_NamespaceRequiredLabelRuleBasic_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &NamespaceRequiredLabelRule{}
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
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
			desc: "non namespace should have error",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}).
					Return(nil),
			},
			resource:  "non-namespace",
			expectErr: true,
		},
		{
			desc: "ignored namespace should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}).
					SetArg(2, merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
						Spec: merlinv1beta1.ClusterRuleNamespaceRequiredLabelSpec{
							Notification:     notification,
							IgnoreNamespaces: []string{"ignoredNS"},
						},
					}).
					Return(nil),
			},
			resource: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "ignoredNS"},
			},
			expect: alert.Alert{
				Message:      "namespace is ignored by the rule",
				ResourceKind: "Namespace",
				ResourceName: "/ignoredNS",
			},
		},
		{
			desc: "violated namespace should get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}).
					SetArg(2, merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
						Spec: merlinv1beta1.ClusterRuleNamespaceRequiredLabelSpec{
							Label:        merlinv1beta1.RequiredLabel{Key: "istio-injection", Value: "enabled"},
							Notification: notification,
						},
					}).
					Return(nil),
			},
			resource: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "namespace"},
			},
			expect: alert.Alert{
				Message:      "doenst have required label `istio-injection`",
				ResourceKind: "Namespace",
				ResourceName: "/namespace",
				Violated:     true,
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

func Test_NamespaceRequiredLabelRuleBasic_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
	r := &NamespaceRequiredLabelRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{
			Spec: merlinv1beta1.ClusterRuleNamespaceRequiredLabelSpec{
				Label:        merlinv1beta1.RequiredLabel{Key: "istio-injection", Value: "enabled"},
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
					List(ctx, &corev1.NamespaceList{}).
					Return(nil),
			},
		},
		{
			desc: "namespace without label should returns violated alert",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.NamespaceList{}).
					SetArg(1, corev1.NamespaceList{
						Items: []corev1.Namespace{
							{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "doenst have required label `istio-injection`",
					ResourceKind: "Namespace",
					ResourceName: "/test",
					Violated:     true,
				},
			},
		},
		{
			desc: "namespace without matched label should returns violated alert",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.NamespaceList{}).
					SetArg(1, corev1.NamespaceList{
						Items: []corev1.Namespace{
							{ObjectMeta: metav1.ObjectMeta{Name: "test", Labels: map[string]string{"test": "val"}}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "doenst have required label `istio-injection`",
					ResourceKind: "Namespace",
					ResourceName: "/test",
					Violated:     true,
				},
			},
		},
		{
			desc: "namespace with matched label should not returns violated alert",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.NamespaceList{}).
					SetArg(1, corev1.NamespaceList{
						Items: []corev1.Namespace{
							{ObjectMeta: metav1.ObjectMeta{Name: "test", Labels: map[string]string{"istio-injection": "enabled"}}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					ResourceKind: "Namespace",
					ResourceName: "/test",
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
