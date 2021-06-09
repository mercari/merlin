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
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/mocks"
)

func Test_ServiceInvalidSelectorRule_Basic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1beta1Rule := &merlinv1beta1.ClusterRuleServiceInvalidSelector{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
			Notification: notification,
		},
	}

	r := &ServiceInvalidSelectorRule{resource: merlinv1beta1Rule}
	assert.Equal(t, merlinv1beta1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, merlinv1beta1Rule, r.GetObject())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRuleServiceInvalidSelector/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
	delay, err := r.GetDelaySeconds(&corev1.Namespace{})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
}

func Test_ServiceInvalidSelectorRule_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &ServiceInvalidSelectorRule{}
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
			desc: "non service should have error",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleServiceInvalidSelector{}).
					Return(nil),
			},
			resource:  "non-service",
			expectErr: true,
		},
		{
			desc: "ignored namespace should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleServiceInvalidSelector{}).
					SetArg(2, merlinv1beta1.ClusterRuleServiceInvalidSelector{
						Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
							Notification:     notification,
							IgnoreNamespaces: []string{"ignoredNS"},
						},
					}).
					Return(nil),
			},
			resource: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ignoredNS", Name: "svc"},
			},
			expect: alert.Alert{
				Message:      "namespace is ignored by the rule",
				ResourceKind: "Service",
				ResourceName: "ignoredNS/svc",
			},
		},
		{
			desc: "selector with no matched pods should get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleServiceInvalidSelector{}).
					SetArg(2, merlinv1beta1.ClusterRuleServiceInvalidSelector{
						Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
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
			resource: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "svc"},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"app": "test"},
				},
			},
			expect: alert.Alert{
				Message:      "Service has no matched pods for the selector",
				ResourceKind: "Service",
				ResourceName: "test/svc",
				Violated:     true,
			},
		},
		{
			desc: "selector with matched pods should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleServiceInvalidSelector{}).
					SetArg(2, merlinv1beta1.ClusterRuleServiceInvalidSelector{
						Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
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
			resource: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "svc"},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"app": "test"},
				},
			},
			expect: alert.Alert{
				Message:      "Service has pods for the selector",
				ResourceKind: "Service",
				ResourceName: "test/svc",
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

func Test_ServiceInvalidSelectorRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
	r := &ServiceInvalidSelectorRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1beta1.ClusterRuleServiceInvalidSelector{
			Spec: merlinv1beta1.ClusterRuleServiceInvalidSelectorSpec{
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
					List(ctx, &corev1.ServiceList{}).
					Return(nil),
			},
		},
		{
			desc: "service without pod should returns violated alert and with pods should not return violated alert",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ServiceList{}).
					SetArg(1, corev1.ServiceList{
						Items: []corev1.Service{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "svc"},
								Spec: corev1.ServiceSpec{
									Selector: map[string]string{"app": "test"},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test2", Name: "svc2"},
								Spec: corev1.ServiceSpec{
									Selector: map[string]string{"app": "test2"},
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
					Message:      "Service has no matched pods for the selector",
					ResourceKind: "Service",
					ResourceName: "test/svc",
					Violated:     true,
				},
				{
					Message:      "Service has pods for the selector",
					ResourceKind: "Service",
					ResourceName: "test2/svc2",
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
