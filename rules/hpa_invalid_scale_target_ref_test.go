package rules

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/go-logr/zapr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/mocks"
)

func Test_HPAInvalidScaleTargetRefRuleBasic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1beta1Rule := &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
			Notification: notification,
		},
	}

	r := &HPAInvalidScaleTargetRefRule{resource: merlinv1beta1Rule}
	assert.Equal(t, merlinv1beta1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, merlinv1beta1Rule, r.GetObject())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRuleHPAInvalidScaleTargetRef/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
	delay, err := r.GetDelaySeconds(&autoscalingv1.HorizontalPodAutoscaler{})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
}

func Test_HPAInvalidScaleTargetRefRule_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &HPAInvalidScaleTargetRefRule{}
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
			desc: "non hpa should have error",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					Return(nil),
			},
			resource:  "non-hpa",
			expectErr: true,
		},
		{
			desc: "ignored namespace should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification:     notification,
							IgnoreNamespaces: []string{"ignoredNS"},
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ignoredNS", Name: "hpa"},
			},
			expect: alert.Alert{
				Message:      "namespace is ignored by the rule",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: "ignoredNS/hpa",
			},
		},
		{
			desc: "non matched deployment should get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.DeploymentList{}, &client.ListOptions{Namespace: "testNS"}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testNS", Name: "hpa"},
				Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{Kind: "Deployment"},
				},
			},
			expect: alert.Alert{
				Message:      "HPA has invalid scale target ref",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: "testNS/hpa",
				Violated:     true,
			},
		},
		{
			desc: "matched deployment should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.DeploymentList{}, &client.ListOptions{Namespace: "testNS"}).
					SetArg(1, appsv1.DeploymentList{
						Items: []appsv1.Deployment{{ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"}}},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testNS", Name: "hpa"},
				Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{Kind: "Deployment", Name: "test-deployment"},
				},
			},
			expect: alert.Alert{
				Message:      "HPA has valid scale target ref",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: "testNS/hpa",
			},
		},
		{
			desc: "non matched replicaSet should get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.ReplicaSetList{}, &client.ListOptions{Namespace: "testNS"}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testNS", Name: "hpa"},
				Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{Kind: "ReplicaSet"},
				},
			},
			expect: alert.Alert{
				Message:      "HPA has invalid scale target ref",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: "testNS/hpa",
				Violated:     true,
			},
		},
		{
			desc: "matched deployment should not get violated alert",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.ReplicaSetList{}, &client.ListOptions{Namespace: "testNS"}).
					SetArg(1, appsv1.ReplicaSetList{
						Items: []appsv1.ReplicaSet{{ObjectMeta: metav1.ObjectMeta{Name: "test-replicaset"}}},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testNS", Name: "hpa"},
				Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{Kind: "ReplicaSet", Name: "test-replicaset"},
				},
			},
			expect: alert.Alert{
				Message:      "HPA has valid scale target ref",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: "testNS/hpa",
			},
		},
		{
			desc: "non deployment or replicasett reference should not get an error",
			key:  ruleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, ruleKey, &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}).
					SetArg(2, merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
						Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
							Notification: notification,
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testNS", Name: "hpa"},
				Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{Kind: "Service", Name: "test-svc"},
				},
			},
			expectErr: true,
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

func Test_HPAInvalidScaleTargetRefRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
	r := &HPAInvalidScaleTargetRefRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{
			Spec: merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRefSpec{
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
			desc: "no resources returns nil alerts",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{}).
					Return(nil),
			},
		},
		{
			desc: "hpa with no matched deployment should have non violated alerts",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{}).
					SetArg(1, autoscalingv1.HorizontalPodAutoscalerList{
						Items: []autoscalingv1.HorizontalPodAutoscaler{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hpa1"},
								Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
									ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
										Kind: "Deployment",
										Name: "test-deployment"},
								},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.DeploymentList{}, &client.ListOptions{Namespace: "default"}).
					SetArg(1, appsv1.DeploymentList{
						Items: []appsv1.Deployment{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "non-matched-deployment"},
							},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "HPA has invalid scale target ref",
					ResourceKind: "HorizontalPodAutoscaler",
					ResourceName: "default/hpa1",
					Violated:     true,
				},
			},
		},
		{
			desc: "hpa with matched deployment/replicaset should have non violated alerts",
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{}).
					SetArg(1, autoscalingv1.HorizontalPodAutoscalerList{
						Items: []autoscalingv1.HorizontalPodAutoscaler{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hpa1"},
								Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
									ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
										Kind: "Deployment",
										Name: "test-deployment"},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hpa2"},
								Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
									ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
										Kind: "ReplicaSet",
										Name: "test-replica"},
								},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.DeploymentList{}, &client.ListOptions{Namespace: "default"}).
					SetArg(1, appsv1.DeploymentList{
						Items: []appsv1.Deployment{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-deployment"},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &appsv1.ReplicaSetList{}, &client.ListOptions{Namespace: "default"}).
					SetArg(1, appsv1.ReplicaSetList{
						Items: []appsv1.ReplicaSet{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-replica"},
							},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "HPA has valid scale target ref",
					ResourceKind: "HorizontalPodAutoscaler",
					ResourceName: "default/hpa1",
				},
				{
					Message:      "HPA has valid scale target ref",
					ResourceKind: "HorizontalPodAutoscaler",
					ResourceName: "default/hpa2",
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
